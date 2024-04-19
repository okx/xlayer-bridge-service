package server

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func NewRequestLogInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()
		methodName := info.FullMethod

		// Actual process of the request
		resp, err := handler(ctx, req)

		duration := time.Since(startTime)
		reqJson, _ := protojson.Marshal(req.(proto.Message))
		respJson, _ := protojson.Marshal(resp.(proto.Message))
		log.Infof("method[%v] req[%v] resp[%v] err[%v] processTime[%v]", methodName, string(reqJson), string(respJson), err, duration.String())
		return resp, err
	}
}

func NewIPCheckInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		methodName := info.FullMethod
		if headers, ok := metadata.FromIncomingContext(ctx); ok {
			xForwardedFor := headers.Get("x-forwarded-for")
			log.Debugf("method[%v] xForwardedFor[%v]", methodName, xForwardedFor)
		}
		return handler(ctx, req)
	}
}
