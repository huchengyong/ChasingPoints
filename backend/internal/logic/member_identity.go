package logic

import (
	"errors"
	"time"

	"chasing_points/internal/svc"
)

// MemberIdentity 是在他人可见位置展示一个用户身份所需的当前信息，按请求实时读取，不进入长期缓存。
type MemberIdentity struct {
	VipLevel  int // 有效会员的成长等级 1-5，0=非会员
	RankLevel int // 指定球种的当前段位等级，只为有效会员读取，其余为 0
}

// LoadMemberIdentities 批量读取用户当前的会员档位，以及有效会员在指定球种的段位等级（头像框只给会员显示）。
// 两部分独立读取：任一部分失败时仍返回已经读到的部分，并返回第一个错误，由调用方记录日志并按 0 展示。
func LoadMemberIdentities(svcCtx *svc.ServiceContext, userIDs []int64, gameType int, now time.Time) (map[int64]MemberIdentity, error) {
	ids := uniquePositiveUserIDs(userIDs)
	identities := make(map[int64]MemberIdentity, len(ids))
	if len(ids) == 0 {
		return identities, nil
	}

	levels, firstErr := ResolveVipLevels(svcCtx, ids, now)
	memberIDs := make([]int64, 0, len(levels))
	for _, id := range ids {
		if level := levels[id]; level > 0 {
			identities[id] = MemberIdentity{VipLevel: level}
			memberIDs = append(memberIDs, id)
		}
	}
	if len(memberIDs) == 0 {
		return identities, firstErr
	}

	if svcCtx.RankingModel == nil {
		if firstErr == nil {
			firstErr = errors.New("ranking model not ready")
		}
		return identities, firstErr
	}
	rankLevels, err := svcCtx.RankingModel.FindRankLevelsByUserIds(memberIDs, gameType)
	if err != nil {
		if firstErr == nil {
			firstErr = err
		}
		return identities, firstErr
	}
	for _, id := range memberIDs {
		identity := identities[id]
		identity.RankLevel = rankLevels[id]
		identities[id] = identity
	}
	return identities, firstErr
}
