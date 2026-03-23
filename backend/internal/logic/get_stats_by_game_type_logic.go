package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

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
			SUM(CASE WHEN result = 1 THEN 1 ELSE 0 END) AS wins,
			SUM(CASE WHEN result = 2 THEN 1 ELSE 0 END) AS losses,
			COALESCE(MAX(my_score), 0) AS highest_score`).
		Where("user_id = ? AND status = 2", userIdInt).
		Group("game_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	list := make([]types.GameTypeStats, 0, len(rows))
	for _, row := range rows {
		winRate := 0.0
		if row.TotalMatches > 0 {
			winRate = float64(row.Wins) / float64(row.TotalMatches)
		}

		list = append(list, types.GameTypeStats{
			GameType:     row.GameType,
			GameTypeName: GetGameTypeName(row.GameType),
			TotalMatches: row.TotalMatches,
			Wins:         row.Wins,
			Losses:       row.Losses,
			WinRate:      winRate,
			HighestScore: row.HighestScore,
		})
	}

	return &types.GetStatsByGameTypeResp{
		Success: true,
		List:    list,
	}, nil
}
