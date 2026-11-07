package match

import (
	"context"
	"net/http"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg"
	"chasing_points/internal/pkg/httperror"
	"chasing_points/internal/pkg/wsticket"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type IssueMatchWSTicketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 签发对局级 WebSocket 连接 ticket
func NewIssueMatchWSTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IssueMatchWSTicketLogic {
	return &IssueMatchWSTicketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *IssueMatchWSTicketLogic) IssueMatchWSTicket(req *types.MatchWSTicketReq) (resp *types.WSTicketResp, err error) {
	userID, err := getWSTicketUserID(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	if req == nil || req.MatchId <= 0 {
		return nil, httperror.New(http.StatusBadRequest, "INVALID_MATCH", "对局不存在或已结束")
	}
	if l.svcCtx == nil || l.svcCtx.MatchModel == nil || l.svcCtx.WSTicketStore == nil {
		return nil, httperror.New(http.StatusServiceUnavailable, "WS_TICKET_UNAVAILABLE", "实时连接暂不可用，请稍后重试")
	}

	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil {
		l.Logger.Errorf("签发对局WebSocket ticket查询对局失败: matchId=%d userId=%d err=%v", req.MatchId, userID, err)
		return nil, err
	}
	if match == nil {
		return nil, httperror.New(http.StatusNotFound, "MATCH_NOT_FOUND", "对局不存在或已结束")
	}
	if model.NormalizeMatchVisibility(match.Visibility, match.MatchMode) == model.MatchVisibilityPrivate && !isMatchWebSocketParticipant(match, userID) {
		return nil, httperror.New(http.StatusForbidden, "MATCH_WS_FORBIDDEN", "无权连接该对局")
	}

	ticket, ttl, err := l.svcCtx.WSTicketStore.Issue(l.ctx, wsticket.Claims{
		UserID:  userID,
		Scope:   wsticket.ScopeMatch,
		MatchID: req.MatchId,
	}, wsticket.DefaultTTL)
	if err != nil {
		l.Logger.Errorf("签发对局WebSocket ticket失败: matchId=%d userId=%d err=%v", req.MatchId, userID, err)
		return nil, httperror.New(http.StatusServiceUnavailable, "WS_TICKET_UNAVAILABLE", "实时连接暂不可用，请稍后重试")
	}

	return &types.WSTicketResp{
		Success:          true,
		Ticket:           ticket,
		Scope:            wsticket.ScopeMatch,
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

func isMatchWebSocketParticipant(match *model.Match, userID int64) bool {
	if match == nil || userID <= 0 {
		return false
	}
	return match.UserId == userID ||
		(match.OpponentId != nil && *match.OpponentId == userID) ||
		(match.RefereeUserId != nil && *match.RefereeUserId == userID)
}
