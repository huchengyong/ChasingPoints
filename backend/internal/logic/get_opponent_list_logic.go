package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOpponentListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对手列表
func NewGetOpponentListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOpponentListLogic {
	return &GetOpponentListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOpponentListLogic) GetOpponentList(req *types.GetOpponentListReq) (resp *types.GetOpponentListResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetOpponentListResp{
			Success: false,
			List:    []types.OpponentRecordItem{},
		}, nil
	}

	// 分页参数
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 获取整体统计数据
	totalOpponents, totalWins, err := l.svcCtx.MatchModel.GetOverallOpponentStats(userId)
	if err != nil {
		l.Logger.Errorf("获取整体对手统计失败: %v", err)
		return &types.GetOpponentListResp{
			Success: false,
			List:    []types.OpponentRecordItem{},
		}, nil
	}

	// 获取对手列表
	stats, total, err := l.svcCtx.MatchModel.ListOpponentsWithStats(userId, req.Keyword, offset, pageSize)
	if err != nil {
		l.Logger.Errorf("获取对手列表失败: %v", err)
		return &types.GetOpponentListResp{
			Success: false,
			List:    []types.OpponentRecordItem{},
		}, nil
	}

	// 转换数据
	list := make([]types.OpponentRecordItem, 0, len(stats))
	for _, stat := range stats {
		// 计算胜率
		var winRate float64
		if stat.TotalMatches > 0 {
			winRate = float64(stat.Wins) / float64(stat.TotalMatches) * 100
			// 保留一位小数
			winRate = float64(int(winRate*10)) / 10
		}

		list = append(list, types.OpponentRecordItem{
			Id:           stat.OpponentId,
			Name:         stat.OpponentName,
			Avatar:       stat.Avatar,
			TotalMatches: stat.TotalMatches,
			Wins:         stat.Wins,
			Losses:       stat.Losses,
			WinRate:      winRate,
			LastMatchAt:  stat.LastMatchAt.Format("2006-01-02T15:04:05+08:00"),
		})
	}

	return &types.GetOpponentListResp{
		Success:        true,
		TotalOpponents: totalOpponents,
		TotalWins:      totalWins,
		Total:          total,
		List:           list,
	}, nil
}
