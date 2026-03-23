package stats

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

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
	// todo: add your logic here and delete this line

	return
}
