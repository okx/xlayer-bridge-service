package localcache

import (
	"context"
	"sync"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

const (
	rpcTimeout                   = 3 * time.Second
	blockNumCacheRefreshInterval = 2 * time.Second
)

type BlockNumCache interface {
	GetLatestBlockNum() uint64
	OnChanged(func(ctx context.Context, oldBlockNum, newBlockNum uint64))
}

type blockNumCacheImpl struct {
	client         *ethclient.Client
	latestBlockNum uint64
	lock           sync.RWMutex
	onChangedFuncs []func(context.Context, uint64, uint64)
}

func NewBlockNumCache(rpcURL string) (BlockNumCache, error) {
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, errors.Wrap(err, "ethclient dial error")
	}

	cache := &blockNumCacheImpl{
		client: client,
	}
	// Init the data
	err = cache.doRefresh(ctx)
	if err != nil {
		log.Errorf("init block num cache error: %v", err)
		return nil, errors.Wrap(err, "doRefresh error")
	}
	go cache.refresh(ctx)
	return cache, nil
}

func (c *blockNumCacheImpl) GetLatestBlockNum() uint64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.latestBlockNum
}

func (c *blockNumCacheImpl) OnChanged(fn func(context.Context, uint64, uint64)) {
	c.onChangedFuncs = append(c.onChangedFuncs, fn)
}

func (c *blockNumCacheImpl) refresh(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ticker := time.NewTicker(blockNumCacheRefreshInterval)
	for range ticker.C {
		//log.Debug("start refreshing block num cache")
		err := c.doRefresh(ctx)
		if err != nil {
			log.Errorf("refresh block num cache error[%v]", err)
		}
		log.Debugf("finish refreshing block num cache")
	}
}

func (c *blockNumCacheImpl) doRefresh(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	// Call RPC to get the latest block number
	blockNum, err := c.client.BlockNumber(timeoutCtx)
	if err != nil {
		return errors.Wrap(err, "eth_blockNumber error")
	}

	// Update the cached value
	c.lock.Lock()
	defer c.lock.Unlock()

	// If the block num is changed, notify the OnChanged functions
	if blockNum != c.latestBlockNum {
		for _, fn := range c.onChangedFuncs {
			fn(ctx, c.latestBlockNum, blockNum)
		}
	}

	c.latestBlockNum = blockNum
	return nil
}
