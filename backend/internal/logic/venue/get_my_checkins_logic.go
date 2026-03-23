package venue

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyCheckinsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取我的打卡记录
func NewGetMyCheckinsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyCheckinsLogic {
	return &GetMyCheckinsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMyCheckinsLogic) GetMyCheckins(req *types.GetMyCheckinsReq) (resp *types.GetMyCheckinsResp, err error) {
	// todo: add your logic here and delete this line

	return
}
