package season

import (
	"context"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSeasonOverviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取当前赛季概览
func NewGetSeasonOverviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSeasonOverviewLogic {
	return &GetSeasonOverviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetSeasonOverviewLogic) GetSeasonOverview(req *types.GetSeasonOverviewReq) (resp *types.GetSeasonOverviewResp, err error) {
	result := &types.GetSeasonOverviewResp{Availability: map[string]bool{}, PartialErrors: []types.ReadPartialError{}, Leaderboard: []types.SeasonLeaderboardItem{}}
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil || l.svcCtx == nil || l.svcCtx.SeasonModel == nil {
		return result, nil
	}
	season, state, resolveErr := ResolveCurrentSeasonLifecycle(l.svcCtx, time.Now())
	result.Success = true
	result.SeasonState = state
	if resolveErr != nil {
		appendSeasonOverviewPartial(result, "season", "赛季状态读取失败")
		return result, nil
	}
	if season == nil {
		result.Availability["season"] = true
		return result, nil
	}
	result.Season = buildSeasonInfo(l.svcCtx, season)
	result.Availability["season"] = true
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		appendSeasonOverviewPartial(result, "record", "竞技读模型尚未切换")
		appendSeasonOverviewPartial(result, "leaderboard", "竞技读模型尚未切换")
		appendSeasonOverviewPartial(result, "competitive_revision", "竞技读模型尚未切换")
		return result, nil
	}
	gameType := 3
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}
	if l.svcCtx.SeasonRecordModel == nil {
		appendSeasonOverviewPartial(result, "record", "赛季记录服务不可用")
		appendSeasonOverviewPartial(result, "leaderboard", "赛季榜单服务不可用")
	} else {
		if record, recordErr := l.svcCtx.SeasonRecordModel.FindBySeasonAndUserAndGameType(season.Id, userID, gameType); recordErr != nil {
			appendSeasonOverviewPartial(result, "record", "赛季记录读取失败")
		} else {
			result.Record = buildSeasonRecordInfo(record, season.Name)
			result.Availability["record"] = true
		}
		if leaderboard, leaderboardErr := loadSeasonLeaderboardPage(l.ctx, l.svcCtx, season.Id, gameType, 1, 20, result.Season); leaderboardErr != nil {
			appendSeasonOverviewPartial(result, "leaderboard", "赛季榜单读取失败")
		} else {
			result.Leaderboard = leaderboard.List
			result.Availability["leaderboard"] = true
		}
	}
	if l.svcCtx.CompetitiveReadModel == nil {
		appendSeasonOverviewPartial(result, "competitive_revision", "竞技快照服务不可用")
	} else if stats, statsErr := l.svcCtx.CompetitiveReadModel.FindStats(userID, 0); statsErr != nil {
		appendSeasonOverviewPartial(result, "competitive_revision", "竞技版本读取失败")
	} else {
		if stats != nil {
			result.CompetitiveRevision = stats.Revision
		}
		result.Availability["competitive_revision"] = true
	}
	return result, nil
}

func appendSeasonOverviewPartial(resp *types.GetSeasonOverviewResp, scope, message string) {
	resp.Availability[scope] = false
	resp.PartialErrors = append(resp.PartialErrors, types.ReadPartialError{Scope: scope, Message: message})
}
