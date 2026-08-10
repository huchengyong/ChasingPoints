package season

import (
	"context"
	"fmt"
	"time"

	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSeasonReportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛季报告
func NewGetSeasonReportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSeasonReportLogic {
	return &GetSeasonReportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSeasonReportLogic) GetSeasonReport(req *types.GetSeasonReportReq) (resp *types.GetSeasonReportResp, err error) {
	if req.SeasonId <= 0 {
		return nil, fmt.Errorf("season_id is required")
	}
	gameType := 3
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetSeasonReportResp{Success: false}, nil
	}

	season, err := l.svcCtx.SeasonModel.FindById(req.SeasonId)
	if err != nil {
		return nil, err
	}
	if season == nil {
		return nil, fmt.Errorf("season not found")
	}
	startAt, endExclusive, err := seasonx.BoundsForConfig(l.svcCtx.Config.SeasonLifecycle, season)
	if err != nil {
		return nil, err
	}

	record, err := l.svcCtx.SeasonRecordModel.FindBySeasonAndUserAndGameType(season.Id, userIdInt, gameType)
	if err != nil {
		return nil, err
	}

	var recordInfo *types.SeasonRecordInfo
	if record != nil {
		recordInfo = buildSeasonRecordInfo(record, season.Name)
	}

	seasonLogs, err := l.svcCtx.RankingModel.ListRankChangesByUserAndGameTypeBetweenHalfOpen(userIdInt, gameType, startAt, endExclusive)
	if err != nil {
		return nil, err
	}
	beforeLog, err := l.svcCtx.RankingModel.FindLatestRankChangeBeforeByGameType(userIdInt, gameType, startAt)
	if err != nil {
		return nil, err
	}

	startScore, endScore, peakScore := buildSeasonSnapshotFromLogs(beforeLog, seasonLogs)
	rankTrend := buildSeasonRankTrendFromLogs(seasonLogs)
	recordInfo = applyHistoricalMatchSeasonSnapshot(recordInfo, season, startScore, endScore, peakScore, len(seasonLogs) > 0)

	winRateByType, err := l.svcCtx.SeasonRecordModel.FindUserWinByTypeInSeasonHalfOpen(userIdInt, startAt, endExclusive)
	if err != nil {
		return nil, err
	}

	topAchievements, err := l.getTopAchievementsInSeason(userIdInt, startAt, endExclusive)
	if err != nil {
		return nil, err
	}

	return &types.GetSeasonReportResp{
		Success: true,
		Report: &types.SeasonReportData{
			Record:          recordInfo,
			RankTrend:       rankTrend,
			WinRateByType:   winRateByType,
			TopAchievements: topAchievements,
		},
	}, nil
}

func (l *GetSeasonReportLogic) getTopAchievementsInSeason(userId int64, startDate, endDate time.Time) ([]types.AchievementDef, error) {
	unlockedAchievements, err := l.svcCtx.UserAchievementModel.FindUnlockedByUserIdBetweenHalfOpen(userId, startDate, endDate, 3)
	if err != nil {
		return nil, err
	}
	if len(unlockedAchievements) == 0 {
		return []types.AchievementDef{}, nil
	}

	achievementIds := make([]int64, 0, len(unlockedAchievements))
	for _, item := range unlockedAchievements {
		achievementIds = append(achievementIds, item.AchievementId)
	}

	achievementDefs, err := l.svcCtx.AchievementModel.FindByIds(achievementIds)
	if err != nil {
		return nil, err
	}

	return buildSeasonTopAchievements(unlockedAchievements, achievementDefs), nil
}

func buildSeasonTopAchievements(unlockedAchievements []model.UserAchievement, achievementDefs []model.Achievement) []types.AchievementDef {
	if len(unlockedAchievements) == 0 || len(achievementDefs) == 0 {
		return []types.AchievementDef{}
	}

	defMap := make(map[int64]model.Achievement, len(achievementDefs))
	for _, item := range achievementDefs {
		defMap[item.Id] = item
	}

	list := make([]types.AchievementDef, 0, len(unlockedAchievements))
	for _, unlocked := range unlockedAchievements {
		def, ok := defMap[unlocked.AchievementId]
		if !ok {
			continue
		}

		unlockedAt := ""
		if unlocked.UnlockedAt != nil {
			unlockedAt = unlocked.UnlockedAt.Format("2006-01-02 15:04:05")
		}

		list = append(list, types.AchievementDef{
			Id:              def.Id,
			Key:             def.Key,
			Name:            def.Name,
			Description:     def.Description,
			Icon:            def.Icon,
			Category:        def.Category,
			GameType:        def.GameType,
			RewardTitleName: def.RewardTitleName,
			Threshold:       def.Threshold,
			Progress:        unlocked.Progress,
			Unlocked:        unlocked.Unlocked == 1,
			UnlockedAt:      unlockedAt,
		})
	}

	return list
}
