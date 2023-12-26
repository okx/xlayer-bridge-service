package pushtask

import (
	"context"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"time"
)

const (
	verifyDurationListLen             = 100
	syncL1VerifiedBatchLockKey        = "sync_l1_verified_batch_lock"
	minVerifyDuration                 = 3 * time.Minute
	defaultVerifyDuration             = 10 * time.Minute
	verifiedBatchCacheRefreshInterval = 10 * time.Second
)

type VerifiedBatchHandler struct {
	rpcUrl       string
	redisStorage redisstorage.RedisStorage
}

func NewVerifiedBatchHandler(rpcUrl string, redisStorage redisstorage.RedisStorage) (*VerifiedBatchHandler, error) {
	return &VerifiedBatchHandler{
		rpcUrl:       rpcUrl,
		redisStorage: redisStorage,
	}, nil
}

func (ins *VerifiedBatchHandler) Start(ctx context.Context) {
	log.Debugf("Starting processSyncVerifyBatchTask, interval:%v", verifiedBatchCacheRefreshInterval)
	ticker := time.NewTicker(verifiedBatchCacheRefreshInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ins.processSyncVerifyBatchTask(ctx)
		}
	}
}

func (ins *VerifiedBatchHandler) processSyncVerifyBatchTask(ctx context.Context) {
	lock, err := ins.redisStorage.TryLock(ctx, syncL1VerifiedBatchLockKey)
	if err != nil {
		log.Errorf("sync latest verify batch lock error, so kip, error: %v", err)
		return
	}
	if !lock {
		log.Infof("sync latest verify batch lock failed, another is running, so kip, error: %v", err)
		return
	}
	defer func() {
		err = ins.redisStorage.ReleaseLock(ctx, syncL1CommittedBatchLockKey)
		if err != nil {
			log.Errorf("ReleaseLock key[%v] error: %v", syncL1CommittedBatchLockKey, err)
		}
	}()
	log.Infof("start to sync latest verify batch")
	now := time.Now().Unix()
	latestBatchNum, err := QueryLatestVerifyBatch(ins.rpcUrl)
	if err != nil {
		log.Warnf("query latest verify batch num error, so skip sync latest commit batch!")
		return
	}
	isBatchLegal, err := ins.checkLatestBatchLegal(ctx, latestBatchNum)
	if err != nil {
		log.Warnf("check latest verify batch num error, so skip sync latest commit batch!")
		return
	}
	if !isBatchLegal {
		log.Infof("latest verify batch num is un-legal, so skip sync latest commit batch!")
		return
	}
	err = ins.freshRedisCacheForVerifyDuration(ctx, latestBatchNum, now)
	log.Infof("success process all thing for sync latest verify batch num %v", latestBatchNum)
}

func (ins *VerifiedBatchHandler) freshRedisCacheForVerifyDuration(ctx context.Context, latestBatchNum uint64, currentTimestamp int64) error {
	err := ins.freshRedisForMaxCommitBatchNum(ctx, latestBatchNum)
	if err != nil {
		return err
	}
	err = ins.freshRedisForAvgCommitDuration(ctx, latestBatchNum, currentTimestamp)
	if err != nil {
		return err
	}
	return nil
}

func (ins *VerifiedBatchHandler) freshRedisForMaxCommitBatchNum(ctx context.Context, latestBatchNum uint64) error {
	return ins.redisStorage.SetVerifyBatchNum(ctx, latestBatchNum)
}

func (ins *VerifiedBatchHandler) freshRedisForAvgCommitDuration(ctx context.Context, latestBatchNum uint64, currTimestamp int64) error {
	err := ins.redisStorage.LPushVerifyTime(ctx, currTimestamp)
	if err != nil {
		return err
	}
	listLen, err := ins.redisStorage.LLenVerifyTimeList(ctx)
	if err != nil {
		return err
	}
	if listLen <= verifyDurationListLen {
		log.Infof("redis verify duration list is not enough, so skip count the avg duration!")
		return nil
	}
	fistTimestamp, err := ins.redisStorage.RPopVerifyTime(ctx)
	if err != nil {
		return err
	}
	newAvgDuration := (currTimestamp - fistTimestamp) / listLen
	if !ins.checkAvgDurationLegal(newAvgDuration) {
		log.Errorf("new avg verify is un-legal, so drop it. new duration: %v", newAvgDuration)
		return nil
	}
	err = ins.redisStorage.SetAvgVerifyDuration(ctx, newAvgDuration)
	if err != nil {
		return err
	}
	log.Infof("success fresh the avg verify duration: %v", newAvgDuration)
	return nil
}

func (ins *VerifiedBatchHandler) checkLatestBatchLegal(ctx context.Context, latestBatchNum uint64) (bool, error) {
	oldBatchNum, err := ins.redisStorage.GetVerifyBatchNum(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Errorf("failed to get verify batch num from redis, so skip, error: %v", err)
		return false, errors.Wrap(err, "failed to get verify batch num from redis")
	}
	if oldBatchNum >= latestBatchNum {
		log.Infof("redis verify batch number: %v gt latest num: %v, so skip", oldBatchNum, latestBatchNum)
		return false, nil
	}
	log.Infof("latest verify batch num check pass, num: %v", latestBatchNum)
	return true, nil
}

func (ins *VerifiedBatchHandler) checkAvgDurationLegal(avgDuration int64) bool {
	return avgDuration > int64(minVerifyDuration) && avgDuration < int64(defaultVerifyDuration)
}

func GetAvgVerifyDuration(ctx context.Context, redisStorage redisstorage.RedisStorage) uint64 {
	avgDuration, err := redisStorage.GetAvgVerifyDuration(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		log.Errorf("get avg verify duration from redis failed, error: %v", err)
		return uint64(defaultVerifyDuration)
	}
	if avgDuration == 0 {
		log.Infof("get avg verify duration from redis is 0, so use default")
		return uint64(defaultVerifyDuration)
	}
	return avgDuration
}
