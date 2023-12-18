package redisstorage

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/redis/go-redis/v9"
)

type RedisStorage interface {
	// Coin price storage
	SetCoinPrice(ctx context.Context, prices []*pb.SymbolPrice) error
	GetCoinPrice(ctx context.Context, symbols []*pb.SymbolInfo) ([]*pb.SymbolPrice, error)

	// Block num storage
	SetL1BlockNum(ctx context.Context, blockNum uint64) error
	GetL1BlockNum(ctx context.Context) (uint64, error)

	// General lock
	TryLock(ctx context.Context, lockKey string) (success bool, err error)
	ReleaseLock(ctx context.Context, lockKey string) error
}

type RedisClient interface {
	Ping(ctx context.Context) *redis.StatusCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HMGet(ctx context.Context, key string, fields ...string) *redis.SliceCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}
