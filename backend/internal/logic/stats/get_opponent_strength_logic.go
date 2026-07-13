package stats

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

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

func (l *GetOpponentStrengthLogic) GetOpponentStrength(req *types.GetOpponentStrengthReq) (resp *types.GetOpponentStrengthResp, err error) {
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
	query := l.svcCtx.DB.Table("matches m").
		Select(`
			CASE
				WHEN ur.rank_score IS NULL OR ur.rank_score <= 1000 THEN '初级(0-1000)'
				WHEN ur.rank_score <= 2000 THEN '中级(1001-2000)'
				ELSE '高级(2001+)'
			END AS rank_range,
			COUNT(*) AS matches,
			SUM(CASE WHEN (m.user_id = ? AND m.result = 1) OR (m.opponent_id = ? AND m.result = 2) THEN 1 ELSE 0 END) AS wins`,
			userIdInt, userIdInt).
		Joins("LEFT JOIN user_ranking ur ON ur.user_id = CASE WHEN m.user_id = ? THEN m.opponent_id ELSE m.user_id END AND ur.game_type = m.game_type", userIdInt).
		Where("(m.user_id = ? OR m.opponent_id = ?) AND m.status = 2", userIdInt, userIdInt)

	if req != nil && req.GameType > 0 {
		query = query.Where("m.game_type = ?", req.GameType)
	}

	err = query.Group("rank_range").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	list := make([]types.OpponentStrengthItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, types.OpponentStrengthItem{
			RankRange: row.RankRange,
			Matches:   row.Matches,
			Wins:      row.Wins,
			WinRate:   calculateWinRatePercent(row.Wins, row.Matches),
		})
	}

	return &types.GetOpponentStrengthResp{
		Success: true,
		List:    list,
	}, nil
}
