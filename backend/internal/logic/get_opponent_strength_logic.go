package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOpponentStrengthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 强弱对手分析
func NewGetOpponentStrengthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOpponentStrengthLogic {
	return &GetOpponentStrengthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOpponentStrengthLogic) GetOpponentStrength() (resp *types.GetOpponentStrengthResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetOpponentStrengthResp{Success: false}, nil
	}

	type opponentStrengthRow struct {
		RankRange string
		Matches   int
		Wins      int
	}

	var rows []opponentStrengthRow
	err = l.svcCtx.DB.Table("matches m").
		Select(`
			CASE
				WHEN ur.rank_score IS NULL OR ur.rank_score <= 1000 THEN '初级(0-1000)'
				WHEN ur.rank_score <= 2000 THEN '中级(1001-2000)'
				ELSE '高级(2001+)'
			END AS rank_range,
			COUNT(*) AS matches,
			SUM(CASE WHEN m.result = 1 THEN 1 ELSE 0 END) AS wins`).
		Joins("LEFT JOIN user_ranking ur ON m.opponent_id = ur.user_id").
		Where("m.user_id = ? AND m.status = 2", userIdInt).
		Group("rank_range").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	list := make([]types.OpponentStrengthItem, 0, len(rows))
	for _, row := range rows {
		winRate := 0.0
		if row.Matches > 0 {
			winRate = float64(row.Wins) / float64(row.Matches)
		}

		list = append(list, types.OpponentStrengthItem{
			RankRange: row.RankRange,
			Matches:   row.Matches,
			Wins:      row.Wins,
			WinRate:   winRate,
		})
	}

	return &types.GetOpponentStrengthResp{
		Success: true,
		List:    list,
	}, nil
}
