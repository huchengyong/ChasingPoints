package challenge

import (
	"context"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// fillChallengeVipLevels 批量填充约球双方的会员档位，一次调用固定两次查询。
// 会员标识只是装饰信息：读取失败时记录日志并按 0 返回，不影响约球信息本身。
func fillChallengeVipLevels(ctx context.Context, svcCtx *svc.ServiceContext, now time.Time, infos ...*types.ChallengeInfo) {
	if len(infos) == 0 {
		return
	}
	userIDs := make([]int64, 0, len(infos)*2)
	for _, info := range infos {
		userIDs = append(userIDs, info.FromUserId, info.ToUserId)
	}
	levels, err := logicx.ResolveVipLevels(svcCtx, userIDs, now)
	if err != nil {
		logx.WithContext(ctx).Errorf("读取约球双方会员档位失败，按非会员返回: %v", err)
		return
	}
	for _, info := range infos {
		info.FromVipLevel = levels[info.FromUserId]
		info.ToVipLevel = levels[info.ToUserId]
	}
}

func challengeInfoPtrs(list []types.ChallengeInfo) []*types.ChallengeInfo {
	ptrs := make([]*types.ChallengeInfo, 0, len(list))
	for index := range list {
		ptrs = append(ptrs, &list[index])
	}
	return ptrs
}
