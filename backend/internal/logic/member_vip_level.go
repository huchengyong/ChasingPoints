package logic

import (
	"errors"
	"time"

	"chasing_points/internal/svc"
)

// ResolveVipLevels 批量解析用户在 now 时刻的会员展示档位。
// 结果只包含有效会员，值为 1-5 的成长等级；非会员、已到期或不存在的用户不在结果中，读取时按 0 处理。
// 固定两次批量查询：用户到期时间，以及有效会员的成长值。等级按当前成长规则由成长值换算，
// 不读取 growth_level 列，避免后台调整门槛后与会员中心不一致。
func ResolveVipLevels(svcCtx *svc.ServiceContext, userIDs []int64, now time.Time) (map[int64]int, error) {
	if svcCtx == nil || svcCtx.UserModel == nil || svcCtx.MemberGrowthProfileModel == nil {
		return nil, errors.New("vip level service not ready")
	}
	ids := uniquePositiveUserIDs(userIDs)
	levels := make(map[int64]int, len(ids))
	if len(ids) == 0 {
		return levels, nil
	}

	expiresAt, err := svcCtx.UserModel.FindMemberExpiresAtByIds(ids)
	if err != nil {
		return nil, err
	}
	activeIDs := make([]int64, 0, len(expiresAt))
	for _, id := range ids {
		if expires, ok := expiresAt[id]; ok && MemberActiveAt(&expires, now) {
			activeIDs = append(activeIDs, id)
		}
	}
	if len(activeIDs) == 0 {
		return levels, nil
	}

	points, err := svcCtx.MemberGrowthProfileModel.FindGrowthPointsByUserIds(activeIDs)
	if err != nil {
		return nil, err
	}
	rules := NewMemberGrowthService(svcCtx, nil).resolveGrowthRules()
	for _, id := range activeIDs {
		levels[id] = ResolveMemberGrowthLevelWithRules(points[id], rules)
	}
	return levels, nil
}

// MemberActiveAt 判断会员到期时间在 now 时刻是否仍有效；未设置到期时间视为非会员，恰好到期的时刻已失效。
func MemberActiveAt(expiresAt *time.Time, now time.Time) bool {
	return expiresAt != nil && InUTC8(*expiresAt).After(InUTC8(now))
}

func uniquePositiveUserIDs(userIDs []int64) []int64 {
	seen := make(map[int64]struct{}, len(userIDs))
	ids := make([]int64, 0, len(userIDs))
	for _, id := range userIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}
