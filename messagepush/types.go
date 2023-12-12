package messagepush

import (
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
)

type PushMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

const (
	transactionUpdateType = "BridgeTxUpdate"
)

type TransactionUpdateData struct {
	FromChain uint32               `json:"fromChain"`
	ToChain   uint32               `json:"toChain"`
	TxHash    string               `json:"txHash"`
	Index     uint64               `json:"index"`
	Status    pb.TransactionStatus `json:"status"`
}
