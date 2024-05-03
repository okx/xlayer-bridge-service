package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// customHTTPErrorHandler checks the error code and message from the error and write them to the response body
// This is to ensure the response from Bridge gateway is aligned with OKX's standard structure (always returns code 200
// and writes the code and message to the body)
func customHTTPErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	log.Debugf("customHTTPErrorHandler err[%v]", err)
	httpCode := http.StatusOK
	var httpStatus *runtime.HTTPStatusError
	if errors.As(err, &httpStatus) {
		log.Debugf("customHTTPErrorHandler error is HTTPStatusError, err[%v]", httpStatus)
		// Unwrap the http error, use the http code inside
		httpCode = httpStatus.HTTPStatus
		err = httpStatus.Err
	}

	// All errors returned from gRPC endpoints would be converted into status error
	s := status.Convert(err)
	// Convert gRPC status back to our custom error to print out the error info
	sErr := NewStatusError(pb.ErrorCode(s.Code()), s.Message())

	log.Debugf("customHTTPErrorHandler error is CustomStatusError, err[%v]", sErr)
	// Build the response body using the common response structure
	resp := &pb.CommonResponse{
		Code:         uint32(sErr.Code()),
		Msg:          sErr.Msg(),
		ErrorCode:    s.Code().String(),
		ErrorMessage: sErr.Msg(),
	}
	body, mErr := marshaler.Marshal(resp)
	if mErr != nil {
		// Fall back to the default handler
		log.Errorf("response body marshal error: %v", mErr)
		runtime.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
		return
	}

	// Always use 200
	w.WriteHeader(httpCode)
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		log.Errorf("writing response body error: %v", err)
	}
}

type CustomStatusError struct {
	code pb.ErrorCode
	msg  string
}

func NewStatusError(code pb.ErrorCode, msg string) *CustomStatusError {
	if code == pb.ErrorCode_ERROR_OK {
		// If there's no error, the returned error should be nil to prevent unexpected behavior
		// msg will be lost in this case, so please don't include it
		return nil
	}

	return &CustomStatusError{code: code, msg: msg}
}

func (e *CustomStatusError) Code() pb.ErrorCode {
	if e == nil {
		return pb.ErrorCode_ERROR_OK
	}
	return e.code
}

func (e *CustomStatusError) Msg() string {
	if e == nil {
		return ""
	}
	return e.msg
}

// GRPCStatus needs to be implemented to convert our error to gRPC status error during the default error handler
func (e *CustomStatusError) GRPCStatus() *status.Status {
	if e == nil {
		return nil
	}
	return status.New(codes.Code(e.code), e.msg)
}

// Implements error interface
func (e *CustomStatusError) Error() string {
	if e == nil {
		return "nil"
	}
	return fmt.Sprintf("error: code = %s msg = %s", e.code, e.msg)
}
