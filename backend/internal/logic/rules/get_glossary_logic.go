package rules

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGlossaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取术语词典
func NewGetGlossaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGlossaryLogic {
	return &GetGlossaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetGlossaryLogic) GetGlossary(req *types.GetGlossaryReq) (resp *types.GetGlossaryResp, err error) {
	// todo: add your logic here and delete this line

	return
}
