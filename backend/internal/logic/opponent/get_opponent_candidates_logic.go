package opponent

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOpponentCandidatesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取好友与近期对手候选
func NewGetOpponentCandidatesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOpponentCandidatesLogic {
	return &GetOpponentCandidatesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetOpponentCandidatesLogic) GetOpponentCandidates(req *types.GetOpponentCandidatesReq) (resp *types.GetOpponentCandidatesResp, err error) {
	result := &types.GetOpponentCandidatesResp{List: []types.OpponentCandidate{}, Availability: map[string]bool{}, PartialErrors: []types.ReadPartialError{}}
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil || l.svcCtx == nil {
		return result, nil
	}
	limit, gameType := 30, 0
	if req != nil {
		if req.Limit > 0 {
			limit = req.Limit
		}
		gameType = req.GameType
	}
	if limit > 100 {
		limit = 100
	}
	result.Success = true
	seen := make(map[int64]struct{})
	if l.svcCtx.FriendModel == nil {
		appendOpponentCandidatePartial(result, "friends", "好友服务不可用")
	} else if friends, friendsErr := l.svcCtx.FriendModel.ListFriendProfiles(userID, limit); friendsErr != nil {
		appendOpponentCandidatePartial(result, "friends", "好友候选读取失败")
	} else {
		for _, friend := range friends {
			if friend.UserId <= 0 {
				continue
			}
			seen[friend.UserId] = struct{}{}
			result.List = append(result.List, types.OpponentCandidate{UserId: friend.UserId, Name: friend.Nickname, Avatar: friend.Avatar, RankName: friend.RankName, Source: "friend"})
		}
		result.Availability["friends"] = true
	}
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		appendOpponentCandidatePartial(result, "recent", "竞技读模型尚未切换")
		appendOpponentCandidatePartial(result, "competitive_revision", "竞技读模型尚未切换")
		return result, nil
	}
	if l.svcCtx.CompetitiveReadModel == nil {
		appendOpponentCandidatePartial(result, "recent", "竞技快照服务不可用")
		appendOpponentCandidatePartial(result, "competitive_revision", "竞技快照服务不可用")
		return result, nil
	}
	if recent, recentErr := l.svcCtx.CompetitiveReadModel.ListRecentOpponentStats(userID, gameType, limit); recentErr != nil {
		appendOpponentCandidatePartial(result, "recent", "近期对手读取失败")
	} else {
		for _, opponent := range recent {
			if opponent.OpponentUserId <= 0 {
				continue
			}
			if _, exists := seen[opponent.OpponentUserId]; exists {
				continue
			}
			seen[opponent.OpponentUserId] = struct{}{}
			lastMatchAt := ""
			if opponent.LastMatchAt != nil {
				lastMatchAt = opponent.LastMatchAt.Format("2006-01-02T15:04:05+08:00")
			}
			result.List = append(result.List, types.OpponentCandidate{UserId: opponent.OpponentUserId, Name: opponent.OpponentName, Avatar: opponent.OpponentAvatar, RankName: opponent.RankName, Source: "recent", LastMatchAt: lastMatchAt})
			if len(result.List) >= limit {
				break
			}
		}
		result.Availability["recent"] = true
	}
	if stats, statsErr := l.svcCtx.CompetitiveReadModel.FindStats(userID, 0); statsErr != nil {
		appendOpponentCandidatePartial(result, "competitive_revision", "竞技版本读取失败")
	} else {
		if stats != nil {
			result.CompetitiveRevision = stats.Revision
		}
		result.Availability["competitive_revision"] = true
	}
	return result, nil
}

func appendOpponentCandidatePartial(resp *types.GetOpponentCandidatesResp, scope, message string) {
	resp.Availability[scope] = false
	resp.PartialErrors = append(resp.PartialErrors, types.ReadPartialError{Scope: scope, Message: message})
}
