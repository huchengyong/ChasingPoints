package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchOpponentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 搜索对手
func NewSearchOpponentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchOpponentLogic {
	return &SearchOpponentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchOpponentLogic) SearchOpponent(req *types.SearchOpponentReq) (resp *types.SearchOpponentResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.SearchOpponentResp{Success: false}, nil
	}

	// 搜索对手
	opponents, err := l.svcCtx.MatchModel.SearchOpponents(userId, req.Keyword, 20)
	if err != nil {
		l.Logger.Errorf("搜索对手失败: %v", err)
		return &types.SearchOpponentResp{Success: false}, nil
	}

	// 转换数据
	list := make([]types.OpponentItem, 0, len(opponents))
	for _, opponent := range opponents {
		// 获取对局次数
		matchCount, _ := l.svcCtx.MatchModel.GetOpponentMatchCount(userId, opponent.Name)
		list = append(list, types.OpponentItem{
			Id:         opponent.Id,
			Name:       opponent.Name,
			Avatar:     opponent.Avatar,
			MatchCount: int(matchCount),
		})
	}

	return &types.SearchOpponentResp{
		Success: true,
		List:    list,
	}, nil
}
