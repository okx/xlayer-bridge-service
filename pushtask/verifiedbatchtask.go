package pushtask

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
)

const (
	verifyDurationListLen             = 5
	syncL1VerifiedBatchLockKey        = "sync_l1_verified_batch_lock"
	minVerifyDuration                 = 2
	defaultVerifyDuration             = 10
	maxVerifyDuration                 = 30
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
	logger := log.LoggerFromCtx(ctx).WithFields("component", "VerifiedBatchTask")
	logger.Debugf("Starting processSyncVerifyBatchTask, interval:%v", verifiedBatchCacheRefreshInterval)
	ticker := time.NewTicker(verifiedBatchCacheRefreshInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctx := log.CtxWithLogger(ctx, utils.LoggerWithRandomTraceID(logger))
			ins.processSyncVerifyBatchTask(ctx)
		}
	}
}

func (ins *VerifiedBatchHandler) processSyncVerifyBatchTask(ctx context.Context) {
	logger := log.LoggerFromCtx(ctx)
	lock, err := ins.redisStorage.TryLock(ctx, syncL1VerifiedBatchLockKey)
	if err != nil {
		logger.Errorf("sync latest verify batch lock error, so kip, error: %v", err)
		return
	}
	if !lock {
		logger.Infof("sync latest verify batch lock failed, another is running, so kip, error: %v", err)
		return
	}
	defer func() {
		err = ins.redisStorage.ReleaseLock(ctx, syncL1VerifiedBatchLockKey)
		if err != nil {
			logger.Errorf("ReleaseLock key[%v] error: %v", syncL1VerifiedBatchLockKey, err)
		}
	}()
	logger.Infof("start to sync latest verify batch")
	now := time.Now().Unix()
	latestBatchNum, err := QueryLatestVerifyBatch(ctx, ins.rpcUrl)
	if err != nil {
		logger.Warnf("query latest verify batch num error, so skip sync latest commit batch!")
		return
	}
	isBatchLegal, err := ins.checkLatestBatchLegal(ctx, latestBatchNum)
	if err != nil {
		logger.Warnf("check latest verify batch num error, so skip sync latest commit batch!")
		return
	}
	if !isBatchLegal {
		logger.Infof("latest verify batch num is un-legal, so skip sync latest commit batch!")
		return
	}
	err = ins.freshRedisCacheForVerifyDuration(ctx, latestBatchNum, now)
	logger.Infof("success process all thing for sync latest verify batch num %v", latestBatchNum)
}

func (ins *VerifiedBatchHandler) freshRedisCacheForVerifyDuration(ctx context.Context, latestBatchNum uint64, currentTimestamp int64) error {
	err := ins.freshRedisForMaxCommitBatchNum(ctx, latestBatchNum)
	if err != nil {
		return err
	}
	err = ins.freshRedisForAvgCommitDuration(ctx, currentTimestamp)
	if err != nil {
		return err
	}
	return nil
}

func (ins *VerifiedBatchHandler) freshRedisForMaxCommitBatchNum(ctx context.Context, latestBatchNum uint64) error {
	return ins.redisStorage.SetVerifyBatchNum(ctx, latestBatchNum)
}

func (ins *VerifiedBatchHandler) freshRedisForAvgCommitDuration(ctx context.Context, currTimestamp int64) error {
	logger := log.LoggerFromCtx(ctx)
	err := ins.redisStorage.LPushVerifyTime(ctx, currTimestamp)
	if err != nil {
		return err
	}
	listLen, err := ins.redisStorage.LLenVerifyTimeList(ctx)
	if err != nil {
		return err
	}
	if listLen <= verifyDurationListLen {
		logger.Infof("redis verify duration list is not enough, so skip count the avg duration!")
		return nil
	}
	fistTimestamp, err := ins.redisStorage.RPopVerifyTime(ctx)
	if err != nil {
		return err
	}
	timestampDiff := currTimestamp - fistTimestamp
	newAvgDuration := (timestampDiff) / (listLen - 1) / secondsPreMinute
	remainder := timestampDiff / (listLen - 1) % secondsPreMinute
	if remainder > 0 {
		newAvgDuration++
	}
	if !ins.checkAvgDurationLegal(newAvgDuration) {
		logger.Errorf("new avg verify is un-legal, so drop it. new duration: %v", newAvgDuration)
		return nil
	}
	err = ins.redisStorage.SetAvgVerifyDuration(ctx, newAvgDuration)
	if err != nil {
		return err
	}
	logger.Infof("success fresh the avg verify duration: %v", newAvgDuration)
	return nil
}

func (ins *VerifiedBatchHandler) checkLatestBatchLegal(ctx context.Context, latestBatchNum uint64) (bool, error) {
	logger := log.LoggerFromCtx(ctx)
	oldBatchNum, err := ins.redisStorage.GetVerifyBatchNum(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Errorf("failed to get verify batch num from redis, so skip, error: %v", err)
		return false, errors.Wrap(err, "failed to get verify batch num from redis")
	}
	if oldBatchNum >= latestBatchNum {
		logger.Infof("redis verify batch number: %v gt latest num: %v, so skip", oldBatchNum, latestBatchNum)
		return false, nil
	}
	logger.Infof("latest verify batch num check pass, num: %v", latestBatchNum)
	return true, nil
}

// checkAvgDurationLegal duration has a default range, 2-30 minutes, if over range, maybe dirty data, drop the data
func (ins *VerifiedBatchHandler) checkAvgDurationLegal(avgDuration int64) bool {
	return avgDuration > int64(minVerifyDuration) && avgDuration < int64(maxVerifyDuration)
}

func GetAvgVerifyDuration(ctx context.Context, redisStorage redisstorage.RedisStorage) uint64 {
	logger := log.LoggerFromCtx(ctx)
	avgDuration, err := redisStorage.GetAvgVerifyDuration(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		logger.Errorf("get avg verify duration from redis failed, error: %v", err)
		return uint64(defaultVerifyDuration)
	}
	if avgDuration == 0 {
		logger.Infof("get avg verify duration from redis is 0, so use default")
		return uint64(defaultVerifyDuration)
	}
	return avgDuration
}

func GetLeftVerifyTime(ctx context.Context, redisStorage redisstorage.RedisStorage, blockNumber uint64, depositCreateTime time.Time,
	l2AvgCommitDuration uint64, l2AvgVerifyDuration uint64, currentTime time.Time) int {
	logger := log.LoggerFromCtx(ctx)
	var blockCommitTime time.Time
	commitTimeSecond, _ := redisStorage.GetL2BlockCommitTime(ctx, blockNumber)
	if commitTimeSecond == 0 {
		logger.Debugf("failed to get commit time for block num, so use create time + avg commit duration")
		blockCommitTime = depositCreateTime.Add(time.Minute * time.Duration(l2AvgCommitDuration))
	} else {
		blockCommitTime = time.Unix(int64(commitTimeSecond), 0)
	}
	return int(l2AvgVerifyDuration - uint64(currentTime.Sub(blockCommitTime).Minutes()))
}
