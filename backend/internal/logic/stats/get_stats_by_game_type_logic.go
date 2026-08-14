package stats

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetStatsByGameTypeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分球种统计
func NewGetStatsByGameTypeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetStatsByGameTypeLogic {
	return &GetStatsByGameTypeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetStatsByGameTypeLogic) GetStatsByGameType() (resp *types.GetStatsByGameTypeResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetStatsByGameTypeResp{Success: false}, nil
	}

	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return &types.GetStatsByGameTypeResp{Success: false}, nil
	}
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		return l.getLegacyStatsByGameType(userIdInt)
	}
	if l.svcCtx.CompetitiveReadModel == nil {
		return &types.GetStatsByGameTypeResp{Success: false}, nil
	}
	rows, err := l.svcCtx.CompetitiveReadModel.ListStats(userIdInt)
	if err != nil {
		return nil, err
	}

	list := make([]types.GameTypeStats, 0, len(rows))
	for _, row := range rows {
		if row.GameType <= 0 {
			continue
		}
		list = append(list, types.GameTypeStats{
			GameType:     row.GameType,
			GameTypeName: GetGameTypeName(row.GameType),
			TotalMatches: row.TotalMatches,
			Wins:         row.Wins,
			Losses:       row.Losses,
			WinRate:      calculateWinRatePercent(row.Wins, row.TotalMatches),
			HighestScore: row.HighestScore,
		})
	}

	return &types.GetStatsByGameTypeResp{
		Success: true,
		List:    list,
	}, nil
}

func (l *GetStatsByGameTypeLogic) getLegacyStatsByGameType(userID int64) (*types.GetStatsByGameTypeResp, error) {
	type row struct {
		GameType     int
		TotalMatches int
		Wins         int
		Losses       int
		HighestScore int
	}
	var rows []row
	err := l.svcCtx.DB.WithContext(l.ctx).Table("matches").
		Select(`
			game_type,
			COUNT(*) AS total_matches,
			SUM(CASE WHEN (user_id = ? AND result = 1) OR (opponent_id = ? AND result = 2) THEN 1 ELSE 0 END) AS wins,
			SUM(CASE WHEN (user_id = ? AND result = 2) OR (opponent_id = ? AND result = 1) THEN 1 ELSE 0 END) AS losses,
			COALESCE(MAX(CASE WHEN user_id = ? THEN my_score WHEN opponent_id = ? THEN opponent_score ELSE 0 END), 0) AS highest_score`,
			userID, userID, userID, userID, userID, userID).
		Where("(user_id = ? OR opponent_id = ?) AND status = ?", userID, userID, 2).
		Where("match_mode = ? OR match_mode = '' OR match_mode IS NULL", model.MatchModeRanked).
		Group("game_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	list := make([]types.GameTypeStats, 0, len(rows))
	for _, item := range rows {
		list = append(list, types.GameTypeStats{
			GameType:     item.GameType,
			GameTypeName: GetGameTypeName(item.GameType),
			TotalMatches: item.TotalMatches,
			Wins:         item.Wins,
			Losses:       item.Losses,
			WinRate:      calculateWinRatePercent(item.Wins, item.TotalMatches),
			HighestScore: item.HighestScore,
		})
	}
	return &types.GetStatsByGameTypeResp{Success: true, List: list}, nil
}
