package messagebridge

import (
	"math/big"

	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/config/businessconfig"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/apolloconfig/agollo/v4/storage"
	"github.com/ethereum/go-ethereum/common"
)

func InitWstETHProcessor(wstETHContractAddresses, wstETHTokenAddresses []common.Address) {
	initWstETHProcessor(wstETHContractAddresses, wstETHTokenAddresses)
	apolloconfig.RegisterChangeHandler(
		"BusinessConfig",
		&businessconfig.Config{},
		apolloconfig.WithAfterFn(func(_ string, change *storage.ConfigChange, c any) {
			cfg := c.(*businessconfig.Config)
			if change.ChangeType == storage.DELETED || len(cfg.WstETHContractAddresses) == 0 || len(cfg.WstETHTokenAddresses) == 0 {
				delete(processorMap, WstETH)
				return
			}
			initUSDCLxLyProcessor(cfg.WstETHContractAddresses, cfg.WstETHTokenAddresses)
		}))
}

func initWstETHProcessor(wstETHContractAddresses, wstETHTokenAddresses []common.Address) {
	mutex := getMutex(WstETH)
	mutex.Lock()
	defer mutex.Unlock()

	log.Debugf("WstETHMapping: contracts[%v] tokens[%v]", wstETHContractAddresses, wstETHTokenAddresses)
	if len(wstETHContractAddresses) != len(wstETHTokenAddresses) {
		log.Errorf("InitWstETHProcessor: contract addresses (%v) and token addresses (%v) have different length", len(wstETHContractAddresses), len(wstETHTokenAddresses))
	}

	contractToTokenMapping := make(map[common.Address]common.Address)
	l := min(len(wstETHContractAddresses), len(wstETHTokenAddresses))
	for i := 0; i < l; i++ {
		if wstETHTokenAddresses[i] == emptyAddress {
			continue
		}
		contractToTokenMapping[wstETHContractAddresses[i]] = wstETHTokenAddresses[i]
	}

	if len(contractToTokenMapping) > 0 {
		processorMap[WstETH] = &Processor{
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
