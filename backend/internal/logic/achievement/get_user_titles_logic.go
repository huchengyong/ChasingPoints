package achievement

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserTitlesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户称号列表
func NewGetUserTitlesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserTitlesLogic {
	return &GetUserTitlesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserTitlesLogic) GetUserTitles() (resp *types.GetUserTitlesResp, err error) {
	// todo: add your logic here and delete this line

	return
}
