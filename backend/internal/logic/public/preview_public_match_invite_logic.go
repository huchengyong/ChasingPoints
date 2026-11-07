package public

import (
	"context"
	"strings"
	"time"

	logic "chasing_points/internal/logic"
	"chasing_points/internal/pkg/matchinvite"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	previewRateLimitPerMinute = 30
	previewRateLimitKeyPrefix = "rl:invite_preview:"
	previewRateLimitedMessage = "操作太频繁，请稍后再试"
)

type PreviewPublicMatchInviteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 未登录预览匹配邀请（凭短码）
func NewPreviewPublicMatchInviteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewPublicMatchInviteLogic {
	return &PreviewPublicMatchInviteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewPublicMatchInviteLogic) PreviewPublicMatchInvite(req *types.PublicMatchInvitePreviewReq) (resp *types.PublicMatchInvitePreviewResp, err error) {
	if req == nil || !matchinvite.ValidCode(strings.TrimSpace(req.InviteCode)) {
		return &types.PublicMatchInvitePreviewResp{Success: false, Message: logic.MatchInviteExpiredMessage}, nil
	}

	// 先限流再查短码：匿名入口不放大对 Redis 的查询量。
	allowed, limitErr := l.allowRequest()
	if limitErr != nil {
		l.Logger.Errorf("匹配邀请预览限流失败: %v", limitErr)
		return &types.PublicMatchInvitePreviewResp{Success: false, Message: logic.MatchInviteUnavailableMessage}, nil
	}
	if !allowed {
		return &types.PublicMatchInvitePreviewResp{Success: false, Message: previewRateLimitedMessage}, nil
	}

	resolved, message := logic.ResolveMatchInviteCode(l.ctx, l.svcCtx, req.InviteCode)
	if message != "" {
		return &types.PublicMatchInvitePreviewResp{Success: false, Message: message}, nil
	}

	return &types.PublicMatchInvitePreviewResp{
		Success: true,
		Preview: &types.PublicMatchInvitePreviewInfo{
			InviterName:   resolved.Inviter.Nickname,
			InviterAvatar: resolved.Inviter.Avatar,
			ExpiresAt:     resolved.ExpiresAt.Format(time.RFC3339),
		},
	}, nil
}

func (l *PreviewPublicMatchInviteLogic) allowRequest() (bool, error) {
	if l.svcCtx == nil || l.svcCtx.Redis == nil {
		return false, matchinvite.ErrCodeStoreMissing
	}
	// 同一个球房可能共用一个出口 IP，30 次/分钟足够正常扫码；短码本身的熵已让枚举不可行。
	ip := strings.TrimSpace(utils.GetClientIP(l.ctx))
	if ip == "" {
		ip = "unknown"
	}
	key := previewRateLimitKeyPrefix + ip
	pipe := l.svcCtx.Redis.TxPipeline()
	incr := pipe.Incr(l.ctx, key)
	ttl := pipe.TTL(l.ctx, key)
	if _, err := pipe.Exec(l.ctx); err != nil {
		return false, err
	}
	// 只要键没有过期时间就补上（包括上次设置失败的情况），避免计数器永久存在、把整个出口 IP 一直拦住。
	if ttl.Val() < 0 {
		if err := l.svcCtx.Redis.Expire(l.ctx, key, time.Minute).Err(); err != nil {
			return false, err
		}
	}
	return incr.Val() <= previewRateLimitPerMinute, nil
}
