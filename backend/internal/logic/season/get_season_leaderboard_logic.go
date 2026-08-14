package season

import (
	"context"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSeasonLeaderboardLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛季排行榜
func NewGetSeasonLeaderboardLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSeasonLeaderboardLogic {
	return &GetSeasonLeaderboardLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetSeasonLeaderboardLogic) GetSeasonLeaderboard(req *types.GetSeasonLeaderboardReq) (resp *types.GetSeasonLeaderboardResp, err error) {
	if req == nil {
		req = &types.GetSeasonLeaderboardReq{}
	}
	seasonId := req.SeasonId
	var seasonInfo *types.SeasonInfo
	gameType := 3
	if req.GameType > 0 {
		gameType = req.GameType
	}

	if seasonId == 0 {
		current, findErr := cachedCurrentSeasonResponse(l.ctx, l.svcCtx, time.Now())
		if findErr != nil {
			l.Logger.Errorf("解析当前赛季失败: err=%v", findErr)
			return &types.GetSeasonLeaderboardResp{Success: false}, nil
		}
		seasonInfo = current.Season
		if seasonInfo == nil && !l.svcCtx.Config.SeasonLifecycle.Enabled && current.SeasonState == "not_started" {
			season, latestErr := l.svcCtx.SeasonModel.FindLatest()
			if latestErr != nil {
				l.Logger.Errorf("查询最近赛季失败: err=%v", latestErr)
				return &types.GetSeasonLeaderboardResp{Success: false}, nil
			}
			seasonInfo = buildSeasonInfo(l.svcCtx, season)
		}
		if seasonInfo == nil {
			return &types.GetSeasonLeaderboardResp{Success: true, Season: nil, Total: 0, List: []types.SeasonLeaderboardItem{}}, nil
		}
		seasonId = seasonInfo.Id
	}

	page, pageSize := normalizeLeaderboardPage(req.Page, req.PageSize)
	result, queryErr := loadSeasonLeaderboardPage(l.ctx, l.svcCtx, seasonId, gameType, page, pageSize, seasonInfo)
	if queryErr != nil {
		l.Logger.Errorf("查询赛季排行榜失败: seasonId=%d err=%v", seasonId, queryErr)
		return &types.GetSeasonLeaderboardResp{Success: false}, nil
	}
	return &result, nil
}

func loadSeasonLeaderboardPage(ctx context.Context, svcCtx *svc.ServiceContext, seasonID int64, gameType, page, pageSize int, seasonInfo *types.SeasonInfo) (types.GetSeasonLeaderboardResp, error) {
	return loadSeasonLeaderboardResponse(ctx, svcCtx, seasonID, gameType, page, pageSize, func() (types.GetSeasonLeaderboardResp, error) {
		if seasonInfo == nil {
			season, err := svcCtx.SeasonModel.FindById(seasonID)
			if err != nil {
				return types.GetSeasonLeaderboardResp{}, err
			}
			if season == nil {
				return types.GetSeasonLeaderboardResp{Success: true, List: []types.SeasonLeaderboardItem{}}, nil
			}
			seasonInfo = buildSeasonInfo(svcCtx, season)
		}
		records, total, err := svcCtx.SeasonRecordModel.FindLeaderboardWithProfilesByGameType(seasonID, gameType, page, pageSize)
		if err != nil {
			return types.GetSeasonLeaderboardResp{}, err
		}
		offset := (page - 1) * pageSize
		list := make([]types.SeasonLeaderboardItem, 0, len(records))
		for index, record := range records {
			list = append(list, types.SeasonLeaderboardItem{
				Rank: offset + index + 1, UserId: record.UserId, Nickname: record.Nickname, Avatar: record.Avatar,
				RankScore: record.EndRankScore, PeakScore: record.PeakRankScore, MatchesPlayed: record.MatchesPlayed, Wins: record.Wins,
			})
		}
		return types.GetSeasonLeaderboardResp{Success: true, Season: seasonInfo, Total: total, List: list}, nil
	})
}

func normalizeLeaderboardPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
