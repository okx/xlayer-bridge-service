package etherman

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	mockbridge "github.com/okx/zkevm-bridge-service/test/mocksmartcontracts/xagonzkevmbridge"
	"github.com/okx/zkevm-node/etherman/smartcontracts/matic"
	"github.com/okx/zkevm-node/etherman/smartcontracts/mockverifier"
	"github.com/okx/zkevm-node/etherman/smartcontracts/xagonzkevm"
	"github.com/okx/zkevm-node/etherman/smartcontracts/xagonzkevmbridge"
	"github.com/okx/zkevm-node/etherman/smartcontracts/xagonzkevmglobalexitroot"
)

// NewSimulatedEtherman creates an etherman that uses a simulated blockchain. It's important to notice that the ChainID of the auth
// must be 1337. The address that holds the auth will have an initial balance of 10 ETH
func NewSimulatedEtherman(cfg Config, auth *bind.TransactOpts) (etherman *Client, ethBackend *backends.SimulatedBackend, maticAddr common.Address, mockBridge *mockbridge.Xagonzkevmbridge, err error) {
	// 10000000 ETH in wei
	balance, _ := new(big.Int).SetString("10000000000000000000000000", 10) //nolint:gomnd
	address := auth.From
	genesisAlloc := map[common.Address]core.GenesisAccount{
		address: {
			Balance: balance,
		},
	}
	blockGasLimit := uint64(999999999999999999) //nolint:gomnd
	client := backends.NewSimulatedBackend(genesisAlloc, blockGasLimit)

	// Deploy contracts
	const maticDecimalPlaces = 18
	totalSupply, _ := new(big.Int).SetString("10000000000000000000000000000", 10) //nolint:gomnd
	maticAddr, _, maticContract, err := matic.DeployMatic(auth, client, "Matic Token", "MATIC", maticDecimalPlaces, totalSupply)
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}
	rollupVerifierAddr, _, _, err := mockverifier.DeployMockverifier(auth, client)
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}
	nonce, err := client.PendingNonceAt(context.TODO(), auth.From)
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}
	const posBridge = 1
	calculatedBridgeAddr := crypto.CreateAddress(auth.From, nonce+posBridge)
	const posXagonZkEVM = 2
	calculatedXagonZkEVMAddress := crypto.CreateAddress(auth.From, nonce+posXagonZkEVM)
	genesis := common.HexToHash("0xfd3434cd8f67e59d73488a2b8da242dd1f02849ea5dd99f0ca22c836c3d5b4a9") // Random value. Needs to be different to 0x0
	exitManagerAddr, _, globalExitRoot, err := xagonzkevmglobalexitroot.DeployXagonzkevmglobalexitroot(auth, client, calculatedXagonZkEVMAddress, calculatedBridgeAddr)
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}
	bridgeAddr, _, mockbr, err := mockbridge.DeployXagonzkevmbridge(auth, client)
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}
	xagonZkEVMAddress, _, xagonZkEVMContract, err := xagonzkevm.DeployXagonzkevm(auth, client, exitManagerAddr, maticAddr, rollupVerifierAddr, bridgeAddr, 1000, 1) //nolint
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}
	_, err = mockbr.Initialize(auth, 0, exitManagerAddr, xagonZkEVMAddress)
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}
	br, err := xagonzkevmbridge.NewXagonzkevmbridge(bridgeAddr, client)
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}
	xagonZkEVMParams := xagonzkevm.XagonZkEVMInitializePackedParameters{
		Admin:                    auth.From,
		TrustedSequencer:         auth.From,
		PendingStateTimeout:      10000, //nolint:gomnd
		TrustedAggregator:        auth.From,
		TrustedAggregatorTimeout: 10000, //nolint:gomnd
	}
	_, err = xagonZkEVMContract.Initialize(auth, xagonZkEVMParams, genesis, "http://localhost", "L2", "v1") //nolint:gomnd
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}

	if calculatedBridgeAddr != bridgeAddr {
		return nil, nil, common.Address{}, nil, fmt.Errorf("bridgeAddr (%s) is different from the expected contract address (%s)",
			bridgeAddr.String(), calculatedBridgeAddr.String())
	}
	if calculatedXagonZkEVMAddress != xagonZkEVMAddress {
		return nil, nil, common.Address{}, nil, fmt.Errorf("xagonZkEVMAddress (%s) is different from the expected contract address (%s)",
			xagonZkEVMAddress.String(), calculatedXagonZkEVMAddress.String())
	}

	// Approve the bridge and xagonZkEVM to spend 10000 matic tokens.
	approvedAmount, _ := new(big.Int).SetString("10000000000000000000000", 10) //nolint:gomnd
	_, err = maticContract.Approve(auth, bridgeAddr, approvedAmount)
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}
	_, err = maticContract.Approve(auth, xagonZkEVMAddress, approvedAmount)
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}
	_, err = xagonZkEVMContract.ActivateForceBatches(auth)
	if err != nil {
		return nil, nil, common.Address{}, nil, err
	}

	client.Commit()
	return &Client{EtherClient: client, XagonBridge: br, XagonZkEVMGlobalExitRoot: globalExitRoot, SCAddresses: []common.Address{exitManagerAddr, bridgeAddr}}, client, maticAddr, mockbr, nil
}
