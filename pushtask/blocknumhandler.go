package pushtask

import (
	"context"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/0xPolygonHermez/zkevm-bridge-service/messagepush"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
)

const (
	queryLimit = 100

	l1BlockNumHandlerLockKey = "bridge_l1_block_num_lock"
)

type L1BlockNumHandler struct {
	storage             DBStorage
	redisStorage        redisstorage.RedisStorage
	messagePushProducer messagepush.KafkaProducer
}

func NewL1BlockNumHandler(storage interface{}, redisStorage redisstorage.RedisStorage, producer messagepush.KafkaProducer) *L1BlockNumHandler {
	return &L1BlockNumHandler{
		storage:             storage.(DBStorage),
		redisStorage:        redisStorage,
		messagePushProducer: producer,
	}
}

// HandleChange queries all txs that reached 64 blocks and push a message to notify that it's pending auto claim
func (h *L1BlockNumHandler) HandleChange(ctx context.Context, newBlockNum uint64) {
	ok, err := h.redisStorage.TryLock(ctx, l1BlockNumHandlerLockKey)
	if err != nil {
		log.Errorf("TryLock error: %v", err)
		return
	}

	if !ok {
		log.Debugf("L1BlockNumHandler locked by other process, ignored")
		return
	}

	defer func() {
		err = h.redisStorage.ReleaseLock(ctx, l1BlockNumHandlerLockKey)
		if err != nil {
			log.Errorf("ReleaseLock error: %v", err)
			return
		}
	}()

	// Replace oldBlockNum with the block num from Redis
	oldBlockNum, err := h.redisStorage.GetL1BlockNum(ctx)
	if err != nil {
		log.Errorf("GetL1BlockNum error: %v", err)
		return
	}
	if oldBlockNum == newBlockNum {
		return
	}

	defer func() {
		// Update Redis cached block num
		err = h.redisStorage.SetL1BlockNum(ctx, newBlockNum)
		if err != nil {
			log.Errorf("SetL1BlockNum error: %v", err)
		}
	}()

	var offset = uint(0)
	for {
		deposits, err := h.storage.GetNotReadyTransactionsWithBlockRange(ctx, 0, oldBlockNum+1-utils.L1TargetBlockConfirmations+1,
			newBlockNum-utils.L1TargetBlockConfirmations, queryLimit, offset, nil)
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
