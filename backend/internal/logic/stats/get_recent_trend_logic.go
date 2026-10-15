package stats

import (
	"context"
	"time"

	"chasing_points/internal/model"
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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetRecentTrendLogic) GetRecentTrend(req *types.GetRecentTrendReq) (resp *types.GetRecentTrendResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetRecentTrendResp{Success: false}, nil
	}

	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return &types.GetRecentTrendResp{Success: false}, nil
	}
	if !l.svcCtx.CompetitiveReadModelsEnabled() {
		return l.getLegacyRecentTrend(userIdInt, req)
	}
	if l.svcCtx.CompetitiveReadModel == nil {
		return &types.GetRecentTrendResp{Success: false}, nil
	}
	limit := 30
	gameType := 0
	if req != nil {
		gameType = req.GameType
		if req.Limit > 0 {
			limit = req.Limit
		}
	}
	rows, err := l.svcCtx.CompetitiveReadModel.ListRecentParticipantResults(userIdInt, gameType, limit)
	if err != nil {
		return nil, err
	}

	list := make([]types.TrendPoint, 0, len(rows))
	wins := 0
	for i, row := range rows {
		if row.Result == 1 {
			wins++
		}
		list = append(list, types.TrendPoint{
			MatchId: row.MatchId,
			Date:    row.CompletedAt.Format("2006-01-02"),
			WinRate: float64(wins) / float64(i+1),
			Result:  row.Result,
		})
	}

	return &types.GetRecentTrendResp{
		Success: true,
		List:    list,
	}, nil
}

func (l *GetRecentTrendLogic) getLegacyRecentTrend(userID int64, req *types.GetRecentTrendReq) (*types.GetRecentTrendResp, error) {
	limit, gameType := 30, 0
	if req != nil {
		gameType = req.GameType
		if req.Limit > 0 {
			limit = req.Limit
		}
	}
	if limit > 100 {
		limit = 100
	}
	type row struct {
		Id        int64
		UserId    int64
		MatchTime time.Time
		Result    *int
	}
	query := l.svcCtx.DB.WithContext(l.ctx).Table("matches").
		Select("id, user_id, match_time, result").
		Where("(user_id = ? OR opponent_id = ?) AND status = ?", userID, userID, 2).
		Where("match_mode = ? OR match_mode = '' OR match_mode IS NULL", model.MatchModeRanked)
	if gameType > 0 {
		query = query.Where("game_type = ?", gameType)
	}
	var rows []row
	if err := query.Order("match_time DESC, id DESC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]types.TrendPoint, 0, len(rows))
	wins := 0
	for index, item := range rows {
		result := 0
		if item.Result != nil {
			result = *item.Result
			if item.UserId != userID {
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
		list = append(list, types.TrendPoint{
			MatchId: item.Id,
			Date:    item.MatchTime.Format("2006-01-02"),
			WinRate: float64(wins) / float64(index+1),
			Result:  result,
		})
	}
	return &types.GetRecentTrendResp{Success: true, List: list}, nil
}
