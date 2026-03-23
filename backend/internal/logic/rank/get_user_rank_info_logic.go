package rank

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserRankInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户段位信息
func NewGetUserRankInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserRankInfoLogic {
	return &GetUserRankInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserRankInfoLogic) GetUserRankInfo(req *types.GetUserRankInfoReq) (resp *types.GetUserRankInfoResp, err error) {
	// todo: add your logic here and delete this line

	return
}
