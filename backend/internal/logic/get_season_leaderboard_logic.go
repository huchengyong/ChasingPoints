package logic

import (
	"context"

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
		svcCtx: svcCtx,
	}
}

func (l *GetSeasonLeaderboardLogic) GetSeasonLeaderboard(req *types.GetSeasonLeaderboardReq) (resp *types.GetSeasonLeaderboardResp, err error) {
	var seasonId int64 = req.SeasonId
	var seasonInfo *types.SeasonInfo
	gameType := 3
	if req != nil && req.GameType > 0 {
		gameType = req.GameType
	}

	if seasonId == 0 {
		season, findErr := l.svcCtx.SeasonModel.FindCurrent()
		if findErr != nil {
			l.Logger.Errorf("查询当前赛季失败: err=%v", findErr)
			return &types.GetSeasonLeaderboardResp{Success: false}, nil
		}
		if season == nil {
			season, findErr = l.svcCtx.SeasonModel.FindLatest()
			if findErr != nil {
				l.Logger.Errorf("查询最近赛季失败: err=%v", findErr)
				return &types.GetSeasonLeaderboardResp{Success: false}, nil
			}
		}

		if season == nil {
			return &types.GetSeasonLeaderboardResp{Success: true, Season: nil, Total: 0, List: []types.SeasonLeaderboardItem{}}, nil
		}

		seasonId = season.Id
		seasonInfo = buildSeasonInfo(season)
	} else {
		season, findErr := l.svcCtx.SeasonModel.FindById(seasonId)
		if findErr != nil {
			l.Logger.Errorf("查询赛季失败: seasonId=%d err=%v", seasonId, findErr)
			return &types.GetSeasonLeaderboardResp{Success: false}, nil
		}
		seasonInfo = buildSeasonInfo(season)
	}

	records, total, queryErr := l.svcCtx.SeasonRecordModel.FindLeaderboardByGameType(seasonId, gameType, req.Page, req.PageSize)
	if queryErr != nil {
		l.Logger.Errorf("查询赛季排行榜失败: seasonId=%d err=%v", seasonId, queryErr)
		return &types.GetSeasonLeaderboardResp{Success: false}, nil
	}

	page, pageSize := normalizeLeaderboardPage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	list := make([]types.SeasonLeaderboardItem, 0, len(records))
	for i, record := range records {
		nickname := ""
		avatar := ""
		if user, userErr := l.svcCtx.UserModel.FindById(record.UserId); userErr == nil && user != nil {
			nickname = user.Nickname
			avatar = user.Avatar
		}

		list = append(list, types.SeasonLeaderboardItem{
			Rank:          offset + i + 1,
			UserId:        record.UserId,
			Nickname:      nickname,
			Avatar:        avatar,
			RankScore:     record.EndRankScore,
			PeakScore:     record.PeakRankScore,
			MatchesPlayed: record.MatchesPlayed,
			Wins:          record.Wins,
		})
	}

	return &types.GetSeasonLeaderboardResp{
		Success: true,
		Season:  seasonInfo,
		Total:   total,
		List:    list,
	}, nil
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
