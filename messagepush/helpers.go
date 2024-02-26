package messagepush

import (
	"context"
	"encoding/json"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/pkg/errors"
)

func convertMsgToString(ctx context.Context, msg interface{}) (string, error) {
	var msgString string
	switch v := msg.(type) {
	case string:
		// If message is a string, just send it
		msgString = v
	default:
		// If message is an object, encode to json
		b, err := json.Marshal(msg)
		if err != nil {
			log.LoggerFromCtx(ctx).Errorf("msg cannot be encoded to json: msg[%v] err[%v]", msg, err)
			return "", errors.Wrap(err, "kafka produce: JSON marshal error")
		}
		msgString = string(b)
	}
	return msgString, nil
}
