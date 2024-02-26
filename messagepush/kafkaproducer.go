package messagepush

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/IBM/sarama"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
)

type produceOptions struct {
	topic   string
	pushKey string
}

type produceOptFunc func(opts *produceOptions)

func WithTopic(topic string) produceOptFunc {
	return func(opts *produceOptions) {
		opts.topic = topic
	}
}

func WithPushKey(key string) produceOptFunc {
	return func(opts *produceOptions) {
		opts.pushKey = key
	}
}

type KafkaProducer interface {
	Produce(ctx context.Context, msg interface{}, optFns ...produceOptFunc) error
	PushTransactionUpdate(ctx context.Context, tx *pb.Transaction, optFns ...produceOptFunc) error
	Close() error

	// GetFakeMessages returns the messages from the fake producer
	// Not available for real kafka producer
	GetFakeMessages(ctx context.Context, topic string) []string
}

type kafkaProducerImpl struct {
	producer       sarama.SyncProducer
	defaultTopic   apolloconfig.Entry[string]
	defaultPushKey apolloconfig.Entry[string]
	bizCode        apolloconfig.Entry[string]
}

func NewKafkaProducer(cfg Config) (KafkaProducer, error) {
	if cfg.UseFakeProducer {
		log.Infof("start to init fake kafka producer!")
		return newFakeProducer(cfg), nil
	}
	log.Infof("start to init real kafka producer!")
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	// Enable SASL authentication
	if cfg.Username != "" && cfg.Password != "" && cfg.RootCAPath != "" {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = cfg.Username
		config.Net.SASL.Password = cfg.Password

		// Read the CA cert from file
		rootCA, err := os.ReadFile(cfg.RootCAPath)
		if err != nil {
			return nil, errors.Wrap(err, "NewKafkaProducer read root CA cert fail")
		}

		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(rootCA)); !ok {
			return nil, errors.New("NewKafkaProducer caCertPool.AppendCertsFromPEM")
		}

		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{RootCAs: caCertPool, InsecureSkipVerify: true} // #nosec
	}

	producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, errors.Wrap(err, "NewKafkaProducer: NewSyncProducer error")
	}
	return &kafkaProducerImpl{
		producer:       producer,
		defaultTopic:   apolloconfig.NewStringEntry("MessagePushProducer.Topic", cfg.Topic),
		defaultPushKey: apolloconfig.NewStringEntry("MessagePushProducer.PushKey", cfg.PushKey),
		bizCode:        apolloconfig.NewStringEntry("MessagePushProducer.BizCode", BizCodeBridgeOrder),
	}, nil
}

// Produce send a message to the Kafka topic
// msg should be either a string or an object
// If msg is an object, it will be encoded to JSON before being sent
func (p *kafkaProducerImpl) Produce(ctx context.Context, msg interface{}, optFns ...produceOptFunc) error {
	logger := log.LoggerFromCtx(ctx)
	if p == nil || p.producer == nil {
		logger.Debugf("Kafka producer is nil")
		return nil
	}
	opts := &produceOptions{
		topic:   p.defaultTopic.Get(),
		pushKey: p.defaultPushKey.Get(),
	}
	for _, f := range optFns {
		f(opts)
	}

	msgString, err := convertMsgToString(ctx, msg)
	if err != nil {
		return err
	}

	produceMsg := &sarama.ProducerMessage{
		Topic: opts.topic,
		Value: sarama.StringEncoder(msgString),
	}
	if opts.pushKey != "" {
		produceMsg.Key = sarama.StringEncoder(opts.pushKey)
	}

	// Send message to the topic
	partition, offset, err := p.producer.SendMessage(produceMsg)

	if err != nil {
		return errors.Wrap(err, "kafka SendMessage error")
	}

	logger.Debugf("Produced to Kafka: topic[%v] msg[%v] partition[%v] offset[%v]", opts.topic, msgString, partition, offset)
	return nil
}

func (p *kafkaProducerImpl) PushTransactionUpdate(ctx context.Context, tx *pb.Transaction, optFns ...produceOptFunc) error {
	if tx == nil {
		return nil
	}
	var b []byte
	var err error
	if tx.Status == uint32(pb.TransactionStatus_TX_CREATED) {
		b, err = protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(tx)
	} else {
		//b, err = json.Marshal([]*pb.Transaction{tx})
		b, err = json.Marshal(tx)
	}
	if err != nil {
		return errors.Wrap(err, "json marshal error")
	}

	msg := &PushMessage{
		BizCode:       p.bizCode.Get(),
		WalletAddress: tx.GetDestAddr(),
		RequestID:     utils.GenerateTraceID(),
		PushContent:   fmt.Sprintf("[%v]", string(b)),
		Time:          time.Now().UnixMilli(),
	}

	return p.Produce(ctx, msg, optFns...)
}

func (p *kafkaProducerImpl) Close() error {
	return p.producer.Close()
}

func (p *kafkaProducerImpl) GetFakeMessages(ctx context.Context, topic string) []string {
	log.LoggerFromCtx(ctx).Warnf("GetFakeMessages should only be called from fakeProducer")
	return nil
}
