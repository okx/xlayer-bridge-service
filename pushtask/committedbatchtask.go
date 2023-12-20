package pushtask

import (
	"context"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/messagepush"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"time"
)

const (
	committedBatchCacheRefreshInterval = 2 * time.Second
	defaultCommitDuration              = 10 * time.Minute
	minCommitDuration                  = 5 * time.Minute
	commitDurationListLen              = 5
	l1NetWorkId                        = 1
	l1PendingDepositQueryLimit         = 100
	syncL1CommittedBatchLockKey        = "sync_l1_committed_batch_lock"
)

var oldMaxBlockNum uint64 = 0

type CommittedBatchHandler struct {
	rpcUrl              string
	client              *ethclient.Client
	storage             DBStorage
	redisStorage        redisstorage.RedisStorage
	messagePushProducer messagepush.KafkaProducer
}

func NewCommittedBatchHandler(rpcUrl string, storage interface{}, redisStorage redisstorage.RedisStorage, producer messagepush.KafkaProducer) (*CommittedBatchHandler, error) {
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, errors.Wrap(err, "ethclient dial error")
	}
	return &CommittedBatchHandler{
		rpcUrl:              rpcUrl,
		client:              client,
		storage:             storage.(DBStorage),
		redisStorage:        redisStorage,
		messagePushProducer: producer,
	}, nil
}

func (ins *CommittedBatchHandler) Start(ctx context.Context) {
	ticker := time.NewTicker(committedBatchCacheRefreshInterval)
	for range ticker.C {
		lock, err := ins.redisStorage.TryLock(ctx, syncL1CommittedBatchLockKey)
		if err != nil {
			log.Errorf("sync latest commit batch lock error, so kip, error: %v", err)
			continue
		}
		if !lock {
			log.Infof("sync latest commit batch lock failed, another is running, so kip, error: %v", err)
			continue
		}
		log.Infof("start to sync latest commit batch")
		latestBatchNum, err := QueryLatestCommitBatch(ins.rpcUrl)
		if err != nil {
			log.Warnf("query latest commit batch num error, so skip sync latest commit batch!")
			continue
		}
		now := time.Now().Unix()
		isBatchLegal, err := ins.checkLatestBatchLegal(ctx, latestBatchNum)
		if err != nil {
			log.Warnf("check latest commit batch num error, so skip sync latest commit batch!")
			continue
		}
		if !isBatchLegal {
			log.Infof("latest commit batch num is un-legal, so skip sync latest commit batch!")
			continue
		}
		err = ins.freshRedisByLatestBatch(ctx, latestBatchNum, now)
		if err != nil {
			log.Warnf("fresh redis for latest commit batch num error, so skip sync latest commit batch!")
			continue
		}
		err = ins.pushStatusChangedMsg(ctx, latestBatchNum)
		if err != nil {
			log.Warnf("push msg for latest commit batch num error, so skip sync latest commit batch!")
			continue
		}
		log.Infof("success process all thing for sync latest commit batch num %v", latestBatchNum)
	}
}

func (ins *CommittedBatchHandler) freshRedisByLatestBatch(ctx context.Context, latestBatchNum uint64, currTimestamp int64) error {
	err := ins.freshRedisForMaxCommitBatchNum(ctx, latestBatchNum)
	if err != nil {
		log.Errorf("fresh redis for max commit batch num err, num: %v, err: %v", latestBatchNum, err)
		return err
	}
	err = ins.freshRedisForMaxCommitBlockNum(ctx, latestBatchNum)
	if err != nil {
		log.Errorf("fresh redis for max commit block num err, num: %v, err: %v", latestBatchNum, err)
		return err
	}
	err = ins.freshRedisForAvgCommitDuration(ctx, latestBatchNum, currTimestamp)
	if err != nil {
		log.Errorf("fresh redis for avg commit duration err, num: %v, err: %v", latestBatchNum, err)
		return err
	}
	log.Infof("success fresh redis cache of latest committed batch by batch %v", latestBatchNum)
	return nil
}

func (ins *CommittedBatchHandler) getMaxBlockNumByBatchNum(ctx context.Context, batchNum uint64) (uint64, error) {
	maxBlockHash, err := QueryMaxBlockHashByBatchNum(ins.rpcUrl, batchNum)
	if err != nil {
		return 0, err
	}
	maxBlockNum, err := QueryBlockNumByBlockHash(ctx, ins.client, maxBlockHash)
	if err != nil {
		return 0, err
	}
	return maxBlockNum, nil
}

func (ins *CommittedBatchHandler) checkLatestBatchLegal(ctx context.Context, latestBatchNum uint64) (bool, error) {
	oldBatchNum, err := ins.redisStorage.GetCommitBatchNum(ctx)
	if err != nil {
		log.Errorf("failed to get batch num from redis, so skip")
		return false, errors.Wrap(err, "failed to get batch num from redis")
	}
	if oldBatchNum >= latestBatchNum {
		log.Infof("redis committed batch number: %v gt latest num: %v, so skip", oldBatchNum, latestBatchNum)
		return false, nil
	}
	log.Infof("latest committed batch num check pass, num: %v", latestBatchNum)
	return true, nil
}

func (ins *CommittedBatchHandler) freshRedisForMaxCommitBatchNum(ctx context.Context, latestBatchNum uint64) error {
	return ins.redisStorage.SetCommitBatchNum(ctx, latestBatchNum)
}

func (ins *CommittedBatchHandler) freshRedisForMaxCommitBlockNum(ctx context.Context, latestBatchNum uint64) error {
	maxBlockNum, err := ins.getMaxBlockNumByBatchNum(ctx, latestBatchNum)
	if err != nil {
		return err
	}
	oldMaxBlockNum, err = ins.redisStorage.GetCommitMaxBlockNum(ctx)
	if err != nil {
		return err
	}
	return ins.redisStorage.SetCommitMaxBlockNum(ctx, maxBlockNum)
}

func (ins *CommittedBatchHandler) freshRedisForAvgCommitDuration(ctx context.Context, latestBatchNum uint64, currTimestamp int64) error {
	listLen, err := ins.redisStorage.LLenCommitTimeList(ctx)
	if err != nil {
		return err
	}
	err = ins.redisStorage.LPushCommitTime(ctx, currTimestamp)
	if err != nil {
		return err
	}
	if listLen < commitDurationListLen {
		log.Infof("redis duration list is not enough, so skip count the avg duration!")
		return nil
	}
	fistTimestamp, err := ins.redisStorage.RPopCommitTime(ctx)
	if err != nil {
		return err
	}
	newAvgDuration := (currTimestamp - fistTimestamp) / listLen
	if !ins.checkAvgDurationLegal(newAvgDuration) {
		log.Errorf("new avg commit is un-legal, so drop it. new duration: %v", newAvgDuration)
		return nil
	}
	err = ins.redisStorage.SetAvgCommitDuration(ctx, newAvgDuration)
	if err != nil {
		return err
	}
	log.Infof("success fresh the avg commit duration: %v", newAvgDuration)
	return nil
}

func (ins *CommittedBatchHandler) pushStatusChangedMsg(ctx context.Context, latestBlockNum uint64) error {
	// Scan the DB and push events to FE
	var offset = uint(0)
	for {
		deposits, err := ins.storage.GetNotReadyTransactionsWithBlockRange(ctx, l1NetWorkId, oldMaxBlockNum+1,
			latestBlockNum, l1PendingDepositQueryLimit, offset, nil)
		if err != nil {
			log.Errorf("query l1 pending deposits error: %v", err)
			return nil
		}
		// Notify FE for each transaction
		for _, deposit := range deposits {
			ins.pushMsgForDeposit(deposit)
		}
		if len(deposits) < l1PendingDepositQueryLimit {
			break
		}
		offset += l1PendingDepositQueryLimit
	}
	return nil
}

func (ins *CommittedBatchHandler) pushMsgForDeposit(deposit *etherman.Deposit) {
	go func(deposit *etherman.Deposit) {
		if ins.messagePushProducer == nil {
			return
		}
		err := ins.messagePushProducer.Produce(&pb.Transaction{
			FromChain: uint32(deposit.NetworkID),
			ToChain:   uint32(deposit.DestinationNetwork),
			TxHash:    deposit.TxHash.String(),
			Index:     uint64(deposit.DepositCount),
			Status:    pb.TransactionStatus_TX_PENDING_VERIFICATION,
			DestAddr:  deposit.DestinationAddress.Hex(),
		})
		if err != nil {
			log.Errorf("PushTransactionUpdate for pending-verify error: %v", err)
		}
	}(deposit)
}

func (ins *CommittedBatchHandler) checkAvgDurationLegal(avgDuration int64) bool {
	return avgDuration > int64(minCommitDuration) && avgDuration < int64(defaultCommitDuration)
}

func GetAvgCommitDuration(ctx context.Context, redisStorage redisstorage.RedisStorage) uint64 {
	avgDuration, err := redisStorage.GetAvgCommitDuration(ctx)
	if err != nil {
		log.Errorf("get avg commit duration from redis failed, error: %v", err)
		return uint64(defaultCommitDuration)
	}
	if avgDuration == 0 {
		log.Infof("get avg commit duration from redis is 0, so use default")
		return uint64(defaultCommitDuration)
	}
	return avgDuration
}
