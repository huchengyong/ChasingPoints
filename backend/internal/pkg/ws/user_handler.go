package ws

import (
	"encoding/json"
	"net/http"
	"time"

	"chasing_points/internal/pkg/wsticket"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// UserWSHandler 用户级WebSocket处理器
// 连接地址: ws://host:port/api/user/ws?ticket=xxx
// 用于接收匹配通知等用户级别的消息
func UserWSHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rejectWebSocketQueryToken(w, r) {
			return
		}

		ticket := r.URL.Query().Get("ticket")
		if ticket == "" {
			http.Error(w, "missing ticket", http.StatusBadRequest)
			return
		}

		claims, err := consumeWebSocketTicket(r.Context(), svcCtx, ticket, wsticket.ScopeUser)
		if err != nil {
			writeWebSocketTicketError(w, err)
			return
		}
		userId := claims.UserID
		active, err := isActiveWebSocketUser(svcCtx, userId)
		if err != nil {
			logx.Errorf("用户WebSocket状态校验失败: userId=%d err=%v", userId, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if !active {
			http.Error(w, "invalid user session", http.StatusUnauthorized)
			return
		}

		// 升级为WebSocket连接
		conn, err := newWebSocketUpgrader(svcCtx.Config).Upgrade(w, r, nil)
		if err != nil {
			logx.Errorf("WebSocket升级失败: %v", err)
			return
		}

		// 创建客户端
		client := &Client{
			Hub:     GlobalHub,
			Conn:    conn,
			MatchId: 0, // 用户级连接没有MatchId
			UserId:  userId,
			Send:    make(chan []byte, 256),
		}

		// 注册到用户房间
		GlobalHub.RegisterUser <- client

		// 启动读写协程
		go client.WritePump()
		go client.userReadPump() // 使用专门的用户读取方法

		logx.Infof("用户WebSocket连接建立: UserId=%d", userId)
	}
}

// userReadPump 用户连接的读取方法（使用UnregisterUser而非Unregister）
func (c *Client) userReadPump() {
	defer func() {
		c.Hub.UnregisterUser <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(c.Hub.getDeadline())
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(c.Hub.getDeadline())
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		// 处理客户端消息（如心跳）
		c.handleUserMessage(message)
	}
}

// handleUserMessage 处理用户消息
func (c *Client) handleUserMessage(message []byte) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err == nil {
		if msg.Type == "ping" {
			pongMsg, _ := json.Marshal(Message{Type: "pong"})
			c.Send <- pongMsg
		}
	}
}

// getDeadline 获取超时时间
func (h *Hub) getDeadline() time.Time {
	return time.Now().Add(pongWait)
}
