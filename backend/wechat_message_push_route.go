package main

import (
	"net/http"
	"path"

	"chasing_points/internal/config"
	wechat "chasing_points/internal/handler/wechat"

	"github.com/zeromicro/go-zero/rest"
)

func withWechatMessagePushRoute(cfg config.Config) rest.StartOption {
	messagePushHandler := wechat.MessagePushHandler(cfg.WechatMiniProgram.MessagePushToken)

	return func(server *http.Server) {
		next := server.Handler
		server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if path.Clean(r.URL.Path) != wechat.MessagePushPath {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			messagePushHandler.ServeHTTP(w, r)
		})
	}
}
