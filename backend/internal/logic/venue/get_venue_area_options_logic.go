package venue

import (
	"context"

	"chasing_points/internal/logic/staticread"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetVenueAreaOptionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取地区选项
func NewGetVenueAreaOptionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVenueAreaOptionsLogic {
	return &GetVenueAreaOptionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetVenueAreaOptionsLogic) GetVenueAreaOptions(req *types.GetVenueAreaOptionsReq) (resp *types.GetVenueAreaOptionsResp, err error) {
	parentId := int64(0)
	if req != nil && req.ParentId > 0 {
		parentId = req.ParentId
	}

	result, err := staticread.Load(l.ctx, l.svcCtx, staticread.Key("areas", parentId), func() (types.GetVenueAreaOptionsResp, error) {
		list, loadErr := l.svcCtx.AreaModel.FindChildren(parentId)
		if loadErr != nil {
			return types.GetVenueAreaOptionsResp{}, loadErr
		}
		options := make([]types.VenueAreaOption, 0, len(list))
		for _, item := range list {
			options = append(options, types.VenueAreaOption{AreaId: item.AreaId, ParentId: item.ParentId, Name: item.Name})
		}
		return types.GetVenueAreaOptionsResp{Success: true, List: options}, nil
	})
	if err != nil {
		l.Logger.Errorf("获取地区选项失败: parentId=%d err=%v", parentId, err)
		return &types.GetVenueAreaOptionsResp{Success: false, List: []types.VenueAreaOption{}}, nil
	}
	return &result, nil
}
