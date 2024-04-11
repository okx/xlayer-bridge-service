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

type simpleTaskFunc func(ctx context.Context)

func RegisterTask(taskKey string, fn xxl.TaskFunc) {
	logger := getLogger()
	if executor == nil {
		logger.Errorf("xxl-job register task failed: executor is nil")
		return
	}
	executor.RegTask(taskKey, fn)
}

func RegisterSimpleTask(taskKey string, fn simpleTaskFunc) {
	RegisterTask(taskKey, func(ctx context.Context, params *xxl.RunReq) string {
		fn(ctx)
		return "task successful"
	})
}

// Stop should be called when the service exits
func Stop() {
	logger := getLogger()
	if executor != nil {
		logger.Infof("stopping xxl-job executor")
		executor.Stop()
	}
}

func executorMiddleware(fn xxl.TaskFunc) xxl.TaskFunc {
	return func(ctx context.Context, param *xxl.RunReq) string {
		startTime := time.Now()
		logger := getLogger()
		b, _ := json.Marshal(param)
		logger.Debugf("received task: %v", string(b))

		res := fn(ctx, param)
		// Catch panic to print the information, then continue to throw the panic so that the xxl admin can report the failure
		if err := recover(); err != nil {
			logger.Infof("task failed, res[%v] err[%v] processTime[%v]", res, err, time.Since(startTime).String())
			panic(err)
		}
		logger.Infof("task done, res[%v] processTime[%v]", res, time.Since(startTime).String())
		return res
	}
}

type customLogger struct {
	*log.Logger
}

func (l *customLogger) Info(format string, a ...interface{}) {
	if l.Logger != nil {
		l.Logger.Infof(format, a...)
	}
}

func (l *customLogger) Error(format string, a ...interface{}) {
	if l.Logger != nil {
		l.Logger.Errorf(format, a...)
	}
}
