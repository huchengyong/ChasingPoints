package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetH2HHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取交锋历史
func NewGetH2HHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetH2HHistoryLogic {
	return &GetH2HHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetH2HHistoryLogic) GetH2HHistory(req *types.H2HHistoryReq) (resp *types.H2HHistoryResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.H2HHistoryResp{Success: false}, nil
	}

	// 验证必须传入 OpponentId
	if req.OpponentId <= 0 {
		l.Logger.Errorf("缺少 opponent_id 参数")
		return &types.H2HHistoryResp{Success: false}, nil
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

	// 查询与指定对手的交锋历史（支持双向查询）
	matches, total, err := l.svcCtx.MatchModel.ListByOpponentId(userId, req.OpponentId, req.Result, offset, pageSize)
	if err != nil {
		l.Logger.Errorf("查询交锋历史失败: %v", err)
		return &types.H2HHistoryResp{Success: false}, nil
	}

	// 转换数据
	list := make([]types.MatchListItem, 0, len(matches))
	for _, match := range matches {
		list = append(list, types.MatchListItem{
			Id:            match.Id,
			GameType:      match.GameType,
			GameTypeName:  GetGameTypeName(match.GameType),
			OpponentName:  match.OpponentName,
			MyScore:       match.MyScore,
			OpponentScore: match.OpponentScore,
			Result:        match.Result,
			MatchTime:     match.MatchTime.Format("2006-01-02T15:04:05+08:00"),
		})
	}

	return &types.H2HHistoryResp{
		Success: true,
		Total:   total,
		List:    list,
	}, nil
}
