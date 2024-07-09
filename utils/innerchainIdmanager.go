package utils

import (
	"sync"

	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/businessconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/apolloconfig/agollo/v4/storage"
)

var (
	standardIdKeyMapper, innerIdKeyMapper map[uint64]uint64
	chainIDMapperLock                     = &sync.RWMutex{}
)

func InnitOkInnerChainIdMapper(cfg businessconfig.Config) {
	initOkInnerChainIdMapper(cfg)

	apolloconfig.RegisterChangeHandler(
		"BusinessConfig",
		&businessconfig.Config{},
		apolloconfig.WithAfterFn(func(_ string, _ *storage.ConfigChange, c any) {
			initOkInnerChainIdMapper(*c.(*businessconfig.Config))
		}))
}

func initOkInnerChainIdMapper(cfg businessconfig.Config) {
	chainIDMapperLock.Lock()
	defer chainIDMapperLock.Unlock()

	standardIdKeyMapper = make(map[uint64]uint64, len(cfg.StandardChainIds))
	innerIdKeyMapper = make(map[uint64]uint64, len(cfg.StandardChainIds))
	if cfg.StandardChainIds == nil {
		log.Infof("inner chain id config is empty, skip init!")
		return
	}
	for i, chainId := range cfg.StandardChainIds {
		innerChainId := cfg.InnerChainIds[i]
		standardIdKeyMapper[chainId] = innerChainId
		innerIdKeyMapper[innerChainId] = chainId
	}
}

func GetStandardChainIdByInnerId(innerChainId uint64) uint64 {
	chainIDMapperLock.RLock()
	defer chainIDMapperLock.RUnlock()

	chainId, found := innerIdKeyMapper[innerChainId]
	if !found {
		return innerChainId
	}
	return chainId
}

func GetInnerChainIdByStandardId(chainId uint64) uint64 {
	chainIDMapperLock.RLock()
	defer chainIDMapperLock.RUnlock()

	innerChainId, found := standardIdKeyMapper[chainId]
	if !found {
		return chainId
	}
	return innerChainId
}
