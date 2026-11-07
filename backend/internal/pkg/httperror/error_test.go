package httperror

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandleStructuredError(t *testing.T) {
	err := New(http.StatusForbidden, "FORBIDDEN", "暂无权限")
	code, body := Handle(err)
	if code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", code)
	}
	payload, ok := body.(Payload)
	if !ok || payload.Success || payload.Reason != "FORBIDDEN" || payload.Message != "暂无权限" {
		t.Fatalf("unexpected payload: %#v", body)
	}
}

func TestHandlePlainErrorKeepsBadRequestFallback(t *testing.T) {
	err := errors.New("bad request")
	code, body := Handle(err)
	if code != http.StatusBadRequest || body != err {
		t.Fatalf("unexpected fallback: code=%d body=%#v", code, body)
	}
}

func TestHandleGrpcErrorKeepsGoZeroHTTPStatusMapping(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
		want int
	}{
		{name: "invalid argument", code: codes.InvalidArgument, want: http.StatusBadRequest},
		{name: "unauthenticated", code: codes.Unauthenticated, want: http.StatusUnauthorized},
		{name: "permission denied", code: codes.PermissionDenied, want: http.StatusForbidden},
		{name: "not found", code: codes.NotFound, want: http.StatusNotFound},
		{name: "canceled", code: codes.Canceled, want: http.StatusRequestTimeout},
		{name: "already exists", code: codes.AlreadyExists, want: http.StatusConflict},
		{name: "resource exhausted", code: codes.ResourceExhausted, want: http.StatusTooManyRequests},
		{name: "internal", code: codes.Internal, want: http.StatusInternalServerError},
		{name: "unimplemented", code: codes.Unimplemented, want: http.StatusNotImplemented},
		{name: "unavailable", code: codes.Unavailable, want: http.StatusServiceUnavailable},
		{name: "deadline exceeded", code: codes.DeadlineExceeded, want: http.StatusGatewayTimeout},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := status.Error(tc.code, "grpc failure")
			code, body := Handle(err)
			if code != tc.want || body != err {
				t.Fatalf("unexpected grpc fallback: code=%d body=%#v", code, body)
			}
		})
	}
}
