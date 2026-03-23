package season

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMySeasonRecordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取我的赛季记录
func NewGetMySeasonRecordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMySeasonRecordLogic {
	return &GetMySeasonRecordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMySeasonRecordLogic) GetMySeasonRecord(req *types.GetMySeasonRecordReq) (resp *types.GetMySeasonRecordResp, err error) {
	// todo: add your logic here and delete this line

	return
}
