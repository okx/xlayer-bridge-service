package utils

var (
	x1TestNetMapper = createChainIdMapper(19500, 195)
	//x1MainNetMapper = createChainIdMapper(196, 196)
)

type ChainIdMapper struct {
	InnerChainId uint64
	ChainId      uint64
}

func createChainIdMapper(innerChainId uint64, chanId uint64) *ChainIdMapper {
	return &ChainIdMapper{
		InnerChainId: innerChainId,
		ChainId:      chanId,
	}
}

func GetChainIdByInnerId(innerChainId uint64) uint64 {
	if x1TestNetMapper.InnerChainId == innerChainId {
		return x1TestNetMapper.ChainId
	}
	//if x1MainNetMapper.InnerChainId == innerChainId {
	//	return x1MainNetMapper.ChainId
	//}
	return innerChainId
}

func GetInnerChainIdByChainId(chainId uint64) uint64 {
	if x1TestNetMapper.ChainId == chainId {
		return x1TestNetMapper.InnerChainId
	}
	//if x1MainNetMapper.ChainId == chainId {
	//	return x1MainNetMapper.InnerChainId
	//}
	return chainId
}
