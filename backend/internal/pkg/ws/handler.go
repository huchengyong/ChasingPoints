package ws

import (
	"errors"
	"net/http"
	"strconv"

	"chasing_points/internal/model"
	pkgx "chasing_points/internal/pkg"
	"chasing_points/internal/svc"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应限制
	},
}

// MatchWSHandler WebSocket处理器
// 连接地址: ws://host:port/api/match/ws?match_id=xxx&token=xxx
func MatchWSHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取参数
		matchIdStr := r.URL.Query().Get("match_id")
		token := r.URL.Query().Get("token")

		if matchIdStr == "" {
			http.Error(w, "missing match_id", http.StatusBadRequest)
			return
		}

		matchId, err := strconv.ParseInt(matchIdStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid match_id", http.StatusBadRequest)
			return
		}

		var userId int64
		if token != "" {
			// 验证token
			userId, err = parseAccessTokenUserID(token, svcCtx.Config.Auth.AccessSecret)
			if err != nil {
				logx.Errorf("WebSocket token验证失败: %v", err)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			active, err := isActiveWebSocketUser(svcCtx, userId)
			if err != nil {
				logx.Errorf("WebSocket用户状态校验失败: userId=%d err=%v", userId, err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			if !active {
				http.Error(w, "invalid user session", http.StatusUnauthorized)
				return
			}
		}

		match, err := svcCtx.MatchModel.FindById(matchId)
		if err != nil || match == nil {
			http.Error(w, "match not found", http.StatusNotFound)
			return
		}
		if model.NormalizeMatchVisibility(match.Visibility, match.MatchMode) == model.MatchVisibilityPrivate {
			if token == "" {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}
			isParticipant := match.UserId == userId ||
				(match.OpponentId != nil && *match.OpponentId == userId) ||
				(match.RefereeUserId != nil && *match.RefereeUserId == userId)
			if !isParticipant {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		// 升级为WebSocket连接
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logx.Errorf("WebSocket升级失败: %v", err)
			return
		}

		// 创建客户端
		client := &Client{
			Hub:     GlobalHub,
			Conn:    conn,
			SvcCtx:  svcCtx,
			MatchId: matchId,
			UserId:  userId,
			Send:    make(chan []byte, 256),
		}

		// 注册到Hub
		GlobalHub.Register <- client

		// 启动读写协程
		go client.WritePump()
		go client.ReadPump()
		go client.sendMatchSync()

		logx.Infof("WebSocket连接建立: MatchId=%d, UserId=%d", matchId, userId)
	}
}

func parseAccessTokenUserID(tokenString string, secret string) (int64, error) {
	claims := &pkgx.JwtClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return 0, err
	}
	if !token.Valid || claims.UserId <= 0 {
		return 0, jwt.ErrTokenInvalidClaims
	}
	if claims.TokenType != "" && claims.TokenType != pkgx.AccessTokenType {
		return 0, jwt.ErrTokenInvalidClaims
	}
	return claims.UserId, nil
}

func isActiveWebSocketUser(svcCtx *svc.ServiceContext, userID int64) (bool, error) {
	if svcCtx == nil || svcCtx.UserModel == nil {
		return false, errors.New("user model missing")
	}
	user, err := svcCtx.UserModel.FindById(userID)
	if err != nil {
		return false, err
	}
	return user != nil && user.Status == 1, nil
}
