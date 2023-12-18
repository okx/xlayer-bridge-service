package pushtask

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/messagepush"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
)

const (
	queryLimit = 100
)

type L1BlockNumHandler struct {
	storage             DBStorage
	messagePushProducer messagepush.KafkaProducer
}

func NewL1BlockNumHandler(storage interface{}, producer messagepush.KafkaProducer) *L1BlockNumHandler {
	return &L1BlockNumHandler{
		storage:             storage.(DBStorage),
		messagePushProducer: producer,
	}
}

// HandleChange queries all txs that reached 64 blocks and push a message to notify that it's pending auto claim
func (h *L1BlockNumHandler) HandleChange(ctx context.Context, oldBlockNum, newBlockNum uint64) {
	// Minus 64 to find the transactions that reached 64 block confirmations
	newBlockNum -= utils.L1TargetBlockConfirmations
	oldBlockNum -= utils.L1TargetBlockConfirmations

	var offset = uint(0)
	for {
		deposits, err := h.storage.GetNotReadyTransactionsWithBlockRange(ctx, 0, oldBlockNum+1, newBlockNum, queryLimit, offset, nil)
		if err != nil {
			log.Errorf("L1BlockNumHandler query error: %v", err)
			return
		}

		// Notify FE for each transaction
		for _, deposit := range deposits {
			go func(deposit *etherman.Deposit) {
				if h.messagePushProducer == nil {
					return
				}
				err := h.messagePushProducer.Produce(&pb.Transaction{
					FromChain: uint32(deposit.NetworkID),
					ToChain:   uint32(deposit.DestinationNetwork),
					TxHash:    deposit.TxHash.String(),
					Index:     uint64(deposit.DepositCount),
					Status:    pb.TransactionStatus_TX_PENDING_AUTO_CLAIM,
					DestAddr:  deposit.DestinationAddress.Hex(),
				})
				if err != nil {
					log.Errorf("PushTransactionUpdate error: %v", err)
				}
			}(deposit)
		}

		if len(deposits) < queryLimit {
			break
		}
		offset += queryLimit
	}
}
