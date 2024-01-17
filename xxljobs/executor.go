package xxljobs

import (
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/xxl-job/xxl-job-executor-go"
)

var (
	executor xxl.Executor
)

func getLogger() *log.Logger {
	return log.WithFields("component", "xxljobexecutor")
}

func InitExecutor(cfg Config) {
	logger := getLogger()

	// Prepare the params for initialization
	opts := []xxl.Option{
		xxl.ServerAddr(cfg.ServerAddr),
		xxl.AccessToken(cfg.AccessToken),
		xxl.SetLogger(&customLogger{logger}),
	}
	if cfg.ExecutorPort != "" {
		opts = append(opts, xxl.ExecutorPort(cfg.ExecutorPort))
	}

	executor = xxl.NewExecutor(opts...)
	executor.Init()
	log.Fatal(executor.Run())
}

type customLogger struct {
	*log.Logger
}

func (l *customLogger) Info(format string, a ...interface{}) {
	if l.Logger != nil {
		l.Logger.Debugf(format, a...)
	}
}

func (l *customLogger) Error(format string, a ...interface{}) {
	if l.Logger != nil {
		l.Logger.Errorf(format, a...)
	}
}
