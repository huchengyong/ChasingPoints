package stats

import (
	"context"

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
		svcCtx: svcCtx,
	}
}

func (l *GetStatsByGameTypeLogic) GetStatsByGameType() (resp *types.GetStatsByGameTypeResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetStatsByGameTypeResp{Success: false}, nil
	}

	type gameTypeStatRow struct {
		GameType     int
		TotalMatches int
		Wins         int
		Losses       int
		HighestScore int
	}

	var rows []gameTypeStatRow
	err = l.svcCtx.DB.Table("matches").
		Select(`
			game_type,
			COUNT(*) AS total_matches,
			SUM(CASE WHEN (user_id = ? AND result = 1) OR (opponent_id = ? AND result = 2) THEN 1 ELSE 0 END) AS wins,
			SUM(CASE WHEN (user_id = ? AND result = 2) OR (opponent_id = ? AND result = 1) THEN 1 ELSE 0 END) AS losses,
			COALESCE(MAX(CASE WHEN user_id = ? THEN my_score WHEN opponent_id = ? THEN opponent_score ELSE 0 END), 0) AS highest_score`,
			userIdInt, userIdInt, userIdInt, userIdInt, userIdInt, userIdInt).
		Where("(user_id = ? OR opponent_id = ?) AND status = 2", userIdInt, userIdInt).
		Group("game_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	list := make([]types.GameTypeStats, 0, len(rows))
	for _, row := range rows {
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
