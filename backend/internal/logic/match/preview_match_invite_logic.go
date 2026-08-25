package match

import (
	"context"
	"strings"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewMatchInviteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 匹配邀请预览
func NewPreviewMatchInviteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewMatchInviteLogic {
	return &PreviewMatchInviteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewMatchInviteLogic) PreviewMatchInvite(req *types.MatchInvitePreviewReq) (resp *types.MatchInvitePreviewResp, err error) {
	if req == nil || strings.TrimSpace(req.InviteToken) == "" {
		return &types.MatchInvitePreviewResp{Success: false, Message: "无效的匹配二维码"}, nil
	}
	claims, message := verifyMatchInvite(l.svcCtx, req.InviteToken)
	if message != "" {
		return &types.MatchInvitePreviewResp{Success: false, Message: message}, nil
	}
	inviter, err := l.svcCtx.UserModel.FindByIdWithContext(l.ctx, claims.InviterUserID)
	if err != nil {
		l.Logger.Errorf("读取匹配邀请发起人失败: userId=%d err=%v", claims.InviterUserID, err)
		return &types.MatchInvitePreviewResp{Success: false, Message: "匹配二维码服务暂不可用"}, nil
	}
	if inviter == nil || inviter.Status != 1 {
		return &types.MatchInvitePreviewResp{Success: false, Message: "匹配二维码已失效，请让对方刷新二维码"}, nil
	}

	return &types.MatchInvitePreviewResp{
		Success: true,
		Preview: &types.MatchInvitePreviewInfo{
			OpponentId:     inviter.Id,
			OpponentName:   inviter.Nickname,
			OpponentAvatar: inviter.Avatar,
			ExpiresAt:      time.Unix(claims.ExpiresAt, 0).UTC().Format(time.RFC3339),
		},
	}, nil
}
