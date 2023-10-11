package main

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/okx/zkevm-bridge-service/utils"
	"github.com/okx/zkevm-node/etherman/smartcontracts/xagonzkevm"
	"github.com/okx/zkevm-node/log"
)

const (
	l1AccHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	l1NetworkURL         = "http://localhost:8545"
	xagonZkEVMAddressHex = "0x610178dA211FEF7D417bC0e6FeD39F05609AD788"
	maticTokenAddressHex = "0x5FbDB2315678afecb367f032d93F642f64180aa3" //nolint:gosec
)

func main() {
	ctx := context.Background()
	// Eth client
	log.Infof("Connecting to l1")
	client, err := utils.NewClient(ctx, l1NetworkURL, common.Address{})
	if err != nil {
		log.Fatal("Error: ", err)
	}
	auth, err := client.GetSigner(ctx, l1AccHexPrivateKey)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	xagonZkEVMAddress := common.HexToAddress(xagonZkEVMAddressHex)
	xagonZkEVM, err := xagonzkevm.NewXagonzkevm(xagonZkEVMAddress, client)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	maticAmount, err := xagonZkEVM.GetForcedBatchFee(&bind.CallOpts{Pending: false})
	if err != nil {
		log.Fatal("Error getting collateral amount from smc: ", err)
	}
	err = client.ApproveERC20(ctx, common.HexToAddress(maticTokenAddressHex), xagonZkEVMAddress, maticAmount, auth)
	if err != nil {
		log.Fatal("Error approving matics: ", err)
	}
	tx, err := xagonZkEVM.SequenceBatches(auth, nil, auth.From)
	if err != nil {
		log.Fatal("Error sending the batch: ", err)
	}

	// Wait eth transfer to be mined
	log.Infof("Waiting tx to be mined")
	const txETHTransferTimeout = 60 * time.Second
	err = utils.WaitTxToBeMined(ctx, client.Client, tx, txETHTransferTimeout)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	log.Info("Batch succefully sent!")
}
