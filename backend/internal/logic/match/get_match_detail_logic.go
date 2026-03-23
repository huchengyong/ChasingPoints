package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对局详情
func NewGetMatchDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchDetailLogic {
	return &GetMatchDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMatchDetailLogic) GetMatchDetail(req *types.GetMatchDetailReq) (resp *types.GetMatchDetailResp, err error) {
	// todo: add your logic here and delete this line

	return
}
