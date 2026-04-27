package stats

import (
	"context"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRecentTrendLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 近N场胜率趋势
func NewGetRecentTrendLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRecentTrendLogic {
	return &GetRecentTrendLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRecentTrendLogic) GetRecentTrend(req *types.GetRecentTrendReq) (resp *types.GetRecentTrendResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetRecentTrendResp{Success: false}, nil
	}

	limit := 30
	if req != nil && req.Limit > 0 {
		limit = req.Limit
	}

	type trendMatchRow struct {
		Id        int64
		UserId    int64
		MatchTime time.Time
		Result    *int
	}

	query := l.svcCtx.DB.Table("matches").
		Select("id, user_id, match_time, result").
		Where("(user_id = ? OR opponent_id = ?) AND status = 2", userIdInt, userIdInt)

	if req != nil && req.GameType > 0 {
		query = query.Where("game_type = ?", req.GameType)
	}

	var rows []trendMatchRow
	err = query.Order("match_time DESC").Limit(limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	list := make([]types.TrendPoint, 0, len(rows))
	wins := 0
	for i, row := range rows {
		result := 0
		if row.Result != nil {
			result = *row.Result
			if row.UserId != userIdInt {
				if result == 1 {
					result = 2
				} else if result == 2 {
					result = 1
				}
			}
		}
		if result == 1 {
			wins++
		}

		winRate := float64(wins) / float64(i+1)
		list = append(list, types.TrendPoint{
			MatchId: row.Id,
			Date:    row.MatchTime.Format("2006-01-02"),
			WinRate: winRate,
			Result:  result,
		})
	}

	return &types.GetRecentTrendResp{
		Success: true,
		List:    list,
	}, nil
}
