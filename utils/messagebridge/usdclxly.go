package messagebridge

import (
	"math/big"

	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/businessconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/apolloconfig/agollo/v4/storage"
	"github.com/ethereum/go-ethereum/common"
)

func InitUSDCLxLyProcessor(usdcContractAddresses, usdcTokenAddresses []common.Address) {
	initUSDCLxLyProcessor(usdcContractAddresses, usdcTokenAddresses)
	apolloconfig.RegisterChangeHandler(
		"BusinessConfig",
		&businessconfig.Config{},
		apolloconfig.WithAfterFn(func(_ string, change *storage.ConfigChange, c any) {
			cfg := c.(*businessconfig.Config)
			if change.ChangeType == storage.DELETED || len(cfg.USDCContractAddresses) == 0 || len(cfg.USDCContractAddresses) == 0 {
				delete(processorMap, USDC)
				return
			}
			initUSDCLxLyProcessor(cfg.USDCContractAddresses, cfg.USDCTokenAddresses)
		}))
}

func initUSDCLxLyProcessor(usdcContractAddresses, usdcTokenAddresses []common.Address) {
	mutex := getMutex(USDC)
	mutex.Lock()
	defer mutex.Unlock()

	log.Debugf("USDCLxLyMapping: contracts[%v] tokens[%v]", usdcContractAddresses, usdcTokenAddresses)
	if len(usdcContractAddresses) != len(usdcTokenAddresses) {
		log.Errorf("InitUSDCLxLyProcessor: contract addresses (%v) and token addresses (%v) have different length", len(usdcContractAddresses), len(usdcTokenAddresses))
	}

	contractToTokenMapping := make(map[common.Address]common.Address)
	l := min(len(usdcContractAddresses), len(usdcTokenAddresses))
	for i := 0; i < l; i++ {
		if usdcTokenAddresses[i] == emptyAddress {
			continue
		}
		contractToTokenMapping[usdcContractAddresses[i]] = usdcTokenAddresses[i]
	}

	if len(contractToTokenMapping) > 0 {
		processorMap[USDC] = &Processor{
			contractToTokenMapping: contractToTokenMapping,
			DecodeMetadataFn: func(metadata []byte) (common.Address, *big.Int) {
				// Metadata structure:
				// - Destination address: 32 bytes
				// - Bridging amount: 32 bytes
				// Maybe there's a more elegant way?
				return common.BytesToAddress(metadata[:32]), new(big.Int).SetBytes(metadata[32:]) //nolint:gomnd
			},
		}
	}
}
