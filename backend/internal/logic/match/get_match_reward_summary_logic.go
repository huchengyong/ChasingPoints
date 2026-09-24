package match

import (
	"context"

	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchRewardSummaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取本场新解锁荣誉
func NewGetMatchRewardSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchRewardSummaryLogic {
	return &GetMatchRewardSummaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMatchRewardSummaryLogic) GetMatchRewardSummary(req *types.GetMatchRewardSummaryReq) (resp *types.GetMatchRewardSummaryResp, err error) {
	emptyList := make([]types.MatchRewardItem, 0)
	if req == nil || req.MatchId <= 0 || l.svcCtx == nil || l.svcCtx.MatchModel == nil {
		return &types.GetMatchRewardSummaryResp{Success: false, List: emptyList}, nil
	}

	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMatchRewardSummaryResp{Success: false, List: emptyList}, nil
	}
	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil {
		return nil, err
	}
	if match == nil {
		return &types.GetMatchRewardSummaryResp{Success: false, Message: "对局不存在", List: emptyList}, nil
	}
	if match.UserId != userId && (match.OpponentId == nil || *match.OpponentId != userId) {
		return &types.GetMatchRewardSummaryResp{Success: false, Message: "你不是本场对局参与者", List: emptyList}, nil
	}
	if match.Status != 2 {
		return pendingMatchRewardSummary(emptyList), nil
	}

	if match.AchievementSyncedAt == nil {
		if syncErr := NewFinishMatchLogic(l.ctx, l.svcCtx).syncAchievementProgressForCompletedMatchWithError(match); syncErr != nil {
			l.Logger.Errorf("补偿同步对局成就失败: matchId=%d, err=%v", match.Id, syncErr)
		}
		match, err = l.svcCtx.MatchModel.FindById(req.MatchId)
		if err != nil {
			return nil, err
		}
		if match == nil || match.AchievementSyncedAt == nil {
			return pendingMatchRewardSummary(emptyList), nil
		}
	}

	if l.svcCtx.UserAchievementModel == nil || l.svcCtx.AchievementModel == nil || l.svcCtx.UserTitleModel == nil {
		return &types.GetMatchRewardSummaryResp{Success: false, Message: "奖励数据暂不可用", List: emptyList}, nil
	}
	unlocks, err := l.svcCtx.UserAchievementModel.FindUnlockedBySource(userId, achievementx.SourceTypeMatch, match.Id)
	if err != nil {
		return nil, err
	}
	if len(unlocks) == 0 {
		return &types.GetMatchRewardSummaryResp{Success: true, Status: "ready", List: emptyList}, nil
	}

	achievementIds := make([]int64, 0, len(unlocks))
	for _, unlock := range unlocks {
		achievementIds = append(achievementIds, unlock.AchievementId)
	}
	achievements, err := l.svcCtx.AchievementModel.FindByIds(achievementIds)
	if err != nil {
		return nil, err
	}
	achievementById := make(map[int64]model.Achievement, len(achievements))
	for _, achievement := range achievements {
		achievementById[achievement.Id] = achievement
	}

	titles, err := l.svcCtx.UserTitleModel.FindByUserId(userId)
	if err != nil {
		return nil, err
	}
	titleByAchievementId := make(map[int64]model.UserTitle)
	for _, title := range titles {
		achievementId := title.SourceRefId
		if title.GrantedByAchievementId != nil {
			achievementId = *title.GrantedByAchievementId
		}
		if title.SourceType == achievementx.SourceTypeAchievement && achievementId > 0 {
			titleByAchievementId[achievementId] = title
		}
	}

	list := make([]types.MatchRewardItem, 0, len(unlocks))
	for _, unlock := range unlocks {
		achievement, ok := achievementById[unlock.AchievementId]
		if !ok {
			continue
		}
		item := types.MatchRewardItem{
			AchievementId:   achievement.Id,
			AchievementKey:  achievement.Key,
			AchievementName: achievement.Name,
			Description:     achievement.Description,
			Icon:            achievement.Icon,
		}
		if unlock.UnlockedAt != nil {
			item.UnlockedAt = unlock.UnlockedAt.Format("2006-01-02 15:04:05")
		}
		if title, ok := titleByAchievementId[achievement.Id]; ok {
			item.RewardTitleId = title.Id
			item.RewardTitleName = title.TitleName
		}
		list = append(list, item)
	}

	return &types.GetMatchRewardSummaryResp{Success: true, Status: "ready", List: list}, nil
}

func pendingMatchRewardSummary(emptyList []types.MatchRewardItem) *types.GetMatchRewardSummaryResp {
	return &types.GetMatchRewardSummaryResp{
		Success: true,
		Status:  "pending",
		Message: "本场荣誉奖励仍在处理中",
		List:    emptyList,
	}
}
