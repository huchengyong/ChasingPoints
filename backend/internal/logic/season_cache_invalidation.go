package logic

import (
	"context"

	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

func invalidateSeasonInfoCache(ctx context.Context, svcCtx *svc.ServiceContext) {
	if svcCtx == nil || svcCtx.Redis == nil {
		return
	}
	if err := svcCtx.Redis.Incr(ctx, "season:info:version").Err(); err != nil {
		logx.WithContext(ctx).Errorf("失效赛季公共信息缓存失败: err=%v", err)
	}
}
