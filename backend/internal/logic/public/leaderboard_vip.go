package public

import (
	"context"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// fillLeaderboardVipLevels 批量填充榜单行的会员档位，一次调用固定两次查询。
// 会员标识只是装饰信息：读取失败时记录日志并按 0 返回，不影响榜单本身。
func fillLeaderboardVipLevels(ctx context.Context, svcCtx *svc.ServiceContext, now time.Time, items ...*types.LeaderboardItem) {
	if len(items) == 0 {
		return
	}
	userIDs := make([]int64, 0, len(items))
	for _, item := range items {
		userIDs = append(userIDs, item.UserId)
	}
	levels, err := logicx.ResolveVipLevels(svcCtx, userIDs, now)
	if err != nil {
		logx.WithContext(ctx).Errorf("读取排行榜会员档位失败，按非会员返回: %v", err)
		return
	}
	for _, item := range items {
		item.VipLevel = levels[item.UserId]
	}
}

func leaderboardItemPtrs(groups ...[]types.LeaderboardItem) []*types.LeaderboardItem {
	count := 0
	for _, group := range groups {
		count += len(group)
	}
	ptrs := make([]*types.LeaderboardItem, 0, count)
	for _, group := range groups {
		for index := range group {
			ptrs = append(ptrs, &group[index])
		}
	}
	return ptrs
}
