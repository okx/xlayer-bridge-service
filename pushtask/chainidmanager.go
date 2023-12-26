package pushtask

type ChainIdManager struct {
	chainIDs map[uint]uint32
}

var instance ChainIdManager

func InitChainIdManager(networks []uint, chainIds []uint) {
	var chainIDs = make(map[uint]uint32)
	for i, network := range networks {
		chainIDs[network] = uint32(chainIds[i])
	}
	instance = ChainIdManager{
		chainIDs: chainIDs,
	}
}

func GetChainIdByNetworkId(networkId uint) uint32 {
	return instance.chainIDs[networkId]
}
