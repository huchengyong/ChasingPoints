package match

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetH2HOverviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取交锋首屏概览
func NewGetH2HOverviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetH2HOverviewLogic {
	return &GetH2HOverviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetH2HOverviewLogic) GetH2HOverview(req *types.H2HOverviewReq) (resp *types.H2HOverviewResp, err error) {
	result := &types.H2HOverviewResp{Availability: map[string]bool{}, PartialErrors: []types.ReadPartialError{}, List: []types.MatchListItem{}}
	if req == nil {
		req = &types.H2HOverviewReq{}
	}
	viewerUserID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil || l.svcCtx == nil {
		return result, nil
	}
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		if l.svcCtx.MatchModel == nil {
			return result, nil
		}
		return l.getLegacyH2HOverview(req)
	}
	if l.svcCtx.CompetitiveReadModel == nil {
		return result, nil
	}
	target, err := resolveH2HReadTarget(l.svcCtx, viewerUserID, req.TargetUserId, req.OpponentId, req.OpponentName)
	if err != nil {
		return result, nil
	}
	result.Success = true
	result.Opponent = &types.H2HOpponent{Id: target.opponentUserID, Name: target.opponentName, Avatar: target.opponentAvatar}
	result.Availability["opponent"] = true
	gameType := req.GameType

	if stats, statsErr := l.svcCtx.CompetitiveReadModel.FindOpponentStats(target.subjectUserID, target.opponentUserID, target.opponentNameKey, gameType); statsErr != nil {
		appendH2HOverviewPartial(result, "stats", "交锋统计读取失败")
	} else {
		result.Stats = h2hStatsPayload(stats)
		result.Availability["stats"] = true
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	startTime, endTime, dateErr := parseH2HHistoryDateRange(req.StartDate, req.EndDate)
	if dateErr != nil {
		appendH2HOverviewPartial(result, "history", "日期格式错误")
	} else if rows, total, historyErr := l.svcCtx.CompetitiveReadModel.ListParticipantH2HPage(target.subjectUserID, target.opponentUserID, target.opponentNameKey, gameType, 0, startTime, endTime, 0, pageSize); historyErr != nil {
		appendH2HOverviewPartial(result, "history", "交锋历史读取失败")
	} else {
		result.Total = total
		result.List = h2hHistoryItems(rows)
		result.HasMore = int64(len(rows)) < total
		result.Availability["history"] = true
	}
	if overall, revisionErr := l.svcCtx.CompetitiveReadModel.FindStats(target.subjectUserID, 0); revisionErr != nil {
		appendH2HOverviewPartial(result, "competitive_revision", "竞技版本读取失败")
	} else {
		if overall != nil {
			result.CompetitiveRevision = overall.Revision
		}
		result.Availability["competitive_revision"] = true
	}
	return result, nil
}

func (l *GetH2HOverviewLogic) getLegacyH2HOverview(req *types.H2HOverviewReq) (*types.H2HOverviewResp, error) {
	result := &types.H2HOverviewResp{Availability: map[string]bool{}, PartialErrors: []types.ReadPartialError{}, List: []types.MatchListItem{}}
	stats, err := NewGetH2HStatsLogic(l.ctx, l.svcCtx).GetH2HStats(&types.H2HStatsReq{
		TargetUserId: req.TargetUserId,
		OpponentId:   req.OpponentId,
		OpponentName: req.OpponentName,
	})
	if err != nil || stats == nil || !stats.Success {
		return result, err
	}
	result.Success = true
	result.Opponent = stats.Opponent
	result.Stats = stats.Stats
	result.Availability["opponent"] = true
	result.Availability["stats"] = true

	history, err := NewGetH2HHistoryLogic(l.ctx, l.svcCtx).GetH2HHistory(&types.H2HHistoryReq{
		TargetUserId: req.TargetUserId,
		OpponentId:   req.OpponentId,
		OpponentName: req.OpponentName,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		Page:         1,
		PageSize:     req.PageSize,
	})
	if err != nil || history == nil || !history.Success {
		appendH2HOverviewPartial(result, "history", "交锋历史读取失败")
		return result, err
	}
	result.Total = history.Total
	result.List = history.List
	result.HasMore = int64(len(history.List)) < history.Total
	result.Availability["history"] = true
	appendH2HOverviewPartial(result, "competitive_revision", "竞技读模型尚未切换")
	return result, nil
}

func h2hStatsPayload(stats *model.UserOpponentStats) *types.H2HStats {
	result := &types.H2HStats{}
	if stats == nil {
		return result
	}
	result.TotalMatches = stats.TotalMatches
	result.MyWins = stats.Wins
	result.OpponentWins = stats.Losses
	result.MaxWinStreak = stats.MaxWinStreak
	if stats.TotalMatches > 0 {
		result.WinRate = float64(stats.Wins) / float64(stats.TotalMatches) * 100
		result.AvgScoreDiff = float64(stats.ScoreDiffSum) / float64(stats.TotalMatches)
	}
	return result
}

func h2hHistoryItems(matches []model.MatchParticipantResult) []types.MatchListItem {
	list := make([]types.MatchListItem, 0, len(matches))
	for _, match := range matches {
		list = append(list, types.MatchListItem{
			Id:             match.MatchId,
			OpponentId:     match.OpponentUserId,
			GameType:       match.GameType,
			GameTypeName:   GetGameTypeName(match.GameType),
			OpponentName:   match.OpponentName,
			OpponentAvatar: match.OpponentAvatar,
			MyScore:        match.MyScore,
			OpponentScore:  match.OpponentScore,
			Result:         match.Result,
			MatchTime:      match.CompletedAt.Format("2006-01-02T15:04:05+08:00"),
		})
	}
	return list
}

func appendH2HOverviewPartial(resp *types.H2HOverviewResp, scope, message string) {
	resp.Availability[scope] = false
	resp.PartialErrors = append(resp.PartialErrors, types.ReadPartialError{Scope: scope, Message: message})
}
