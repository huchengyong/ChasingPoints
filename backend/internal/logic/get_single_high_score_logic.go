package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSingleHighScoreLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 单杆最高分记录
func NewGetSingleHighScoreLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSingleHighScoreLogic {
	return &GetSingleHighScoreLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSingleHighScoreLogic) GetSingleHighScore(req *types.GetSingleHighScoreReq) (resp *types.GetSingleHighScoreResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetSingleHighScoreResp{Success: false}, nil
	}

	limit := 10
	if req != nil && req.Limit > 0 {
		limit = req.Limit
	}

	records, err := loadUserSingleHighScoreRecords(l.svcCtx, userIdInt, req.GameType, limit)
	if err != nil {
		return nil, err
	}

	return &types.GetSingleHighScoreResp{
		Success: true,
		List:    records,
	}, nil
}
