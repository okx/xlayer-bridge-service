package pushtask

import (
	"context"
	"fmt"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/messagepush"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/xxl-job/xxl-job-executor-go"
)

type L1BlockNumTask struct {
	storage             DBStorage
	redisStorage        redisstorage.RedisStorage
	client              *ethclient.Client
	messagePushProducer messagepush.KafkaProducer
	rollupID            uint
}

func NewL1BlockNumTask(rpcURL string, storage interface{}, redisStorage redisstorage.RedisStorage, producer messagepush.KafkaProducer, rollupID uint) (*L1BlockNumTask, error) {
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, errors.Wrap(err, "ethclient dial error")
	}

	return &L1BlockNumTask{
		storage:             storage.(DBStorage),
		redisStorage:        redisStorage,
		client:              client,
		messagePushProducer: producer,
		rollupID:            rollupID,
	}, nil
}

func (t *L1BlockNumTask) Run(ctx context.Context, params *xxl.RunReq) string {
	// Get the latest block num from the chain RPC
	blockNum, err := t.client.BlockNumber(ctx)
	if err != nil {
		log.Errorf("eth_blockNumber error: %v", err)
		panic(err)
	}

	// Get the previous block num from Redis cache and check
	oldBlockNum, err := t.redisStorage.GetL1BlockNum(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Errorf("Get L1 block num from Redis error: %v", err)
		panic(err)
	}

	// If the block num is not changed, no need to do anything
	if blockNum <= oldBlockNum {
		return "latest L1 block number is unchanged, skip task"
	}

	defer func(blockNum uint64) {
		// Update Redis cached block num
		err = t.redisStorage.SetL1BlockNum(ctx, blockNum)
		if err != nil {
			log.Errorf("SetL1BlockNum error: %v", err)
		}
	}(blockNum)

	// Minus 64 to get the target query block num
	oldBlockNum -= utils.Min(utils.L1TargetBlockConfirmations.Get(), oldBlockNum)
	blockNum -= utils.Min(utils.L1TargetBlockConfirmations.Get(), blockNum)
	if blockNum <= oldBlockNum {
		return "latest L1 block number is unchanged, skip task"
	}

	var (
		totalDeposits = 0
	)

	for block := oldBlockNum + 1; block <= blockNum; block++ {
		// For each block num, get the list of deposit and push the events to FE
		deposits, err := t.redisStorage.GetBlockDepositList(ctx, 0, block)
		if err != nil {
			log.Errorf("L1BlockNumTask query Redis error: %v", err)
			panic(err)
		}
		totalDeposits += len(deposits)

		// Notify FE for each transaction
		for _, deposit := range deposits {
			go func(deposit *etherman.Deposit) {
				if t.messagePushProducer == nil {
					return
				}
				if deposit.LeafType != uint8(utils.LeafTypeAsset) {
					log.Infof("transaction is not asset, so skip push update change, hash: %v", deposit.TxHash)
					return
				}
				err := t.messagePushProducer.PushTransactionUpdate(&pb.Transaction{
					FromChain:   uint32(deposit.NetworkID),
					ToChain:     uint32(deposit.DestinationNetwork),
					TxHash:      deposit.TxHash.String(),
					Index:       uint64(deposit.DepositCount),
					Status:      uint32(pb.TransactionStatus_TX_PENDING_AUTO_CLAIM),
					DestAddr:    deposit.DestinationAddress.Hex(),
					GlobalIndex: etherman.GenerateGlobalIndex(true, t.rollupID-1, deposit.DepositCount).String(),
				})
				if err != nil {
					log.Errorf("PushTransactionUpdate error: %v", err)
				}
			}(deposit)
		}
	}

	return fmt.Sprintf("L1BlockNumTask push for %v deposits, block num from %v to %v", totalDeposits, oldBlockNum, blockNum)
}
