package utils

type contextKey string

const (
	TraceID    contextKey = "traceID"
	traceIDLen            = 16
)
