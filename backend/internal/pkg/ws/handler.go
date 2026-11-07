package ws

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"chasing_points/internal/model"
	pkgx "chasing_points/internal/pkg"
	"chasing_points/internal/pkg/wsticket"
	"chasing_points/internal/svc"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/zeromicro/go-zero/core/logx"
)

// MatchWSHandler WebSocket处理器
// 连接地址: ws://host:port/api/match/ws?match_id=xxx&ticket=xxx
func MatchWSHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取参数
		matchIdStr := r.URL.Query().Get("match_id")
		ticket := r.URL.Query().Get("ticket")

		if matchIdStr == "" {
			http.Error(w, "missing match_id", http.StatusBadRequest)
			return
		}
		if rejectWebSocketQueryToken(w, r) {
			return
		}

		matchId, err := strconv.ParseInt(matchIdStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid match_id", http.StatusBadRequest)
			return
		}

		var userId int64
		if ticket != "" {
			claims, err := consumeWebSocketTicket(r.Context(), svcCtx, ticket, wsticket.ScopeMatch)
			if err != nil {
				writeWebSocketTicketError(w, err)
				return
			}
			if claims.MatchID != matchId {
				http.Error(w, "invalid ticket", http.StatusUnauthorized)
				return
			}
			userId = claims.UserID
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
			if ticket == "" {
				http.Error(w, "missing ticket", http.StatusUnauthorized)
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
		conn, err := newWebSocketUpgrader(svcCtx.Config).Upgrade(w, r, nil)
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

func rejectWebSocketQueryToken(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Query().Get("token") == "" {
		return false
	}
	http.Error(w, "query token is not supported", http.StatusUnauthorized)
	return true
}

func consumeWebSocketTicket(ctx context.Context, svcCtx *svc.ServiceContext, ticket string, scope string) (*wsticket.Claims, error) {
	if svcCtx == nil || svcCtx.WSTicketStore == nil {
		return nil, wsticket.ErrStoreMissing
	}
	return svcCtx.WSTicketStore.Consume(ctx, ticket, scope)
}

func writeWebSocketTicketError(w http.ResponseWriter, err error) {
	if errors.Is(err, wsticket.ErrStoreMissing) {
		http.Error(w, "ticket service unavailable", http.StatusServiceUnavailable)
		return
	}
	if errors.Is(err, wsticket.ErrTicketNotFound) || errors.Is(err, wsticket.ErrInvalidTicket) {
		http.Error(w, "invalid ticket", http.StatusUnauthorized)
		return
	}
	http.Error(w, "ticket service unavailable", http.StatusServiceUnavailable)
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
