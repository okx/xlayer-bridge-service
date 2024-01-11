package apolloconfig

import (
	"github.com/0xPolygonHermez/zkevm-node/log"
)

func getLogger() *log.Logger {
	return log.WithFields(loggerFieldKey, loggerFieldValue)
}
