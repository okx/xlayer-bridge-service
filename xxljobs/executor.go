package xxljobs

import (
	"context"
	"encoding/json"
	"time"

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
	logger.Infof("Starting to initialize xxl-job executor")

	// Prepare the params for initialization
	opts := []xxl.Option{
		xxl.ServerAddr(cfg.ServerAddr),
		xxl.AccessToken(cfg.AccessToken),
		xxl.SetLogger(&customLogger{logger}),
	}
	if cfg.ExecutorPort != "" {
		opts = append(opts, xxl.ExecutorPort(cfg.ExecutorPort))
	}
	if cfg.RegistryKey != "" {
		opts = append(opts, xxl.RegistryKey(cfg.RegistryKey))
	}

	executor = xxl.NewExecutor(opts...)
	executor.Init()
	executor.Use(executorMiddleware)
	go executor.Run()
}

func RegisterTask(taskKey string, fn xxl.TaskFunc) {
	logger := getLogger()
	if executor == nil {
		logger.Errorf("xxl-job register task failed: executor is nil")
		return
	}
	executor.RegTask(taskKey, fn)
}

func executorMiddleware(fn xxl.TaskFunc) xxl.TaskFunc {
	return func(ctx context.Context, param *xxl.RunReq) string {
		startTime := time.Now()
		logger := getLogger()
		b, _ := json.Marshal(param)
		logger.Debugf("received task: %v", string(b))

		res := fn(ctx, param)
		logger.Infof("task done, res[%v] processTime[%v]", res, time.Since(startTime).String())
		return res
	}
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
