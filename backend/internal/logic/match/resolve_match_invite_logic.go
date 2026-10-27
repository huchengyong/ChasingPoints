package match

import (
	"context"
	"time"

	logic "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResolveMatchInviteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 用匹配码短码兑换邀请 token
func NewResolveMatchInviteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResolveMatchInviteLogic {
	return &ResolveMatchInviteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResolveMatchInviteLogic) ResolveMatchInvite(req *types.MatchInviteResolveReq) (resp *types.MatchInviteResolveResp, err error) {
	if req == nil {
		return &types.MatchInviteResolveResp{Success: false, Message: logic.MatchInviteExpiredMessage}, nil
	}
	resolved, message := logic.ResolveMatchInviteCode(l.ctx, l.svcCtx, req.InviteCode)
	if message != "" {
		return &types.MatchInviteResolveResp{Success: false, Message: message}, nil
	}

	// 兑换不消耗短码与 token，开局仍由现有开局接口凭 token 校验。
	return &types.MatchInviteResolveResp{
		Success:     true,
		InviteToken: resolved.Token,
		Preview: &types.MatchInvitePreviewInfo{
			OpponentId:     resolved.Inviter.Id,
			OpponentName:   resolved.Inviter.Nickname,
			OpponentAvatar: resolved.Inviter.Avatar,
			ExpiresAt:      resolved.ExpiresAt.Format(time.RFC3339),
		},
	}, nil
}
