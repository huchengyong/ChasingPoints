package user

import (
	"context"
	"net/http"

	"chasing_points/internal/pkg"
	"chasing_points/internal/pkg/httperror"
	"chasing_points/internal/pkg/wsticket"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type IssueUserWSTicketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 签发用户级 WebSocket 连接 ticket
func NewIssueUserWSTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IssueUserWSTicketLogic {
	return &IssueUserWSTicketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *IssueUserWSTicketLogic) IssueUserWSTicket() (resp *types.WSTicketResp, err error) {
	userID, err := getWSTicketUserID(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}

	ticket, ttl, err := l.svcCtx.WSTicketStore.Issue(l.ctx, wsticket.Claims{
		UserID: userID,
		Scope:  wsticket.ScopeUser,
	}, wsticket.DefaultTTL)
	if err != nil {
		l.Logger.Errorf("签发用户WebSocket ticket失败: userId=%d err=%v", userID, err)
		return nil, httperror.New(http.StatusServiceUnavailable, "WS_TICKET_UNAVAILABLE", "实时连接暂不可用，请稍后重试")
	}

	return &types.WSTicketResp{
		Success:          true,
		Ticket:           ticket,
		Scope:            wsticket.ScopeUser,
		ExpiresInSeconds: int64(ttl.Seconds()),
	}, nil
}

func getWSTicketUserID(ctx context.Context, svcCtx *svc.ServiceContext) (int64, error) {
	userID, err := utils.GetUserIDFromCtx(ctx)
	if err != nil {
		return 0, httperror.New(http.StatusUnauthorized, "UNAUTHORIZED", "请先完成登录")
	}
	tokenType, err := utils.GetOptionalTokenTypeFromCtx(ctx)
	if err != nil {
		return 0, httperror.New(http.StatusInternalServerError, "TOKEN_CONTEXT_INVALID", "登录状态异常")
	}
	if tokenType != "" && tokenType != pkg.AccessTokenType {
		return 0, httperror.New(http.StatusUnauthorized, "INVALID_ACCESS_TOKEN", "无效的访问令牌")
	}
	if svcCtx == nil || svcCtx.UserModel == nil || svcCtx.WSTicketStore == nil {
		return 0, httperror.New(http.StatusServiceUnavailable, "WS_TICKET_UNAVAILABLE", "实时连接暂不可用，请稍后重试")
	}
	user, err := svcCtx.UserModel.FindByIdWithContext(ctx, userID)
	if err != nil {
		return 0, err
	}
	if user == nil || user.Status != 1 {
		return 0, httperror.New(http.StatusUnauthorized, "SESSION_INVALID", "登录状态已失效，请重新登录")
	}
	return userID, nil
}
