package ws

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/pkg/wsticket"
	"chasing_points/internal/svc"
)

// failingTicketStore 模拟票据服务不可用：store 缺失，或 Redis 连接失败。
type failingTicketStore struct {
	err error
}

func (s failingTicketStore) Issue(ctx context.Context, claims wsticket.Claims, ttl time.Duration) (string, time.Duration, error) {
	return "", 0, s.err
}

func (s failingTicketStore) Consume(ctx context.Context, ticket string, expectedScope string) (*wsticket.Claims, error) {
	return nil, s.err
}

// 票据服务故障时必须失败关闭：返回 503，且不升级连接、不注册用户。
func TestUserWSHandlerFailsClosedWhenTicketServiceUnavailable(t *testing.T) {
	cases := map[string]error{
		"store missing":     wsticket.ErrStoreMissing,
		"redis unreachable": errors.New("dial tcp 127.0.0.1:6379: connect: connection refused"),
	}
	for name, storeErr := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := config.Config{}
			cfg.Security.WebSocketAllowedOrigins = userWSAllowedOrigin
			hub := NewHub()
			previousHub := GlobalHub
			GlobalHub = hub
			t.Cleanup(func() { GlobalHub = previousHub })

			server := httptest.NewServer(UserWSHandler(&svc.ServiceContext{
				Config:        cfg,
				WSTicketStore: failingTicketStore{err: storeErr},
			}))
			t.Cleanup(server.Close)

			conn, response, err := dialUserWS(t, server, "any-ticket")
			if conn != nil {
				conn.Close()
				t.Fatal("connection must not be upgraded while the ticket service is unavailable")
			}
			if err == nil || response == nil || response.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("expected 503 fail-closed response, got response=%v err=%v", response, err)
			}
			select {
			case client := <-hub.RegisterUser:
				t.Fatalf("user must not be registered, got %+v", client)
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
}
