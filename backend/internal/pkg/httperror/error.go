package httperror

import (
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Error struct {
	statusCode int
	body       any
	message    string
}

func (e *Error) Error() string {
	return e.message
}

type Payload struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

func New(statusCode int, reason, message string) error {
	return &Error{
		statusCode: statusCode,
		body: Payload{
			Success: false,
			Reason:  reason,
			Message: message,
		},
		message: message,
	}
}

func Handle(err error) (int, any) {
	var target *Error
	if errors.As(err, &target) {
		return target.statusCode, target.body
	}
	if _, ok := err.(interface{ GRPCStatus() *status.Status }); ok {
		return grpcHTTPStatus(status.Code(err)), err
	}
	return http.StatusBadRequest, err
}

func grpcHTTPStatus(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Internal, codes.DataLoss, codes.Unknown:
		return http.StatusInternalServerError
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

func Configure() {
	httpx.SetErrorHandler(Handle)
}
