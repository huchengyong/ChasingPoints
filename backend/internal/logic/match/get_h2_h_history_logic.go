package match

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

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
	// todo: add your logic here and delete this line

	return
}
