package gear

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGearBrandsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取装备目录品牌
func NewGetGearBrandsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGearBrandsLogic {
	return &GetGearBrandsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetGearBrandsLogic) GetGearBrands(req *types.GetGearBrandsReq) (resp *types.GetGearBrandsResp, err error) {
	if _, err := utils.GetUserIDFromCtx(l.ctx); err != nil {
		return &types.GetGearBrandsResp{Success: false, Message: "用户未登录", List: []types.GearBrandInfo{}}, nil
	}
	if req == nil || !model.IsGearCategory(req.Category) {
		return &types.GetGearBrandsResp{Success: false, Message: "装备类别无效", List: []types.GearBrandInfo{}}, nil
	}

	brands, err := l.svcCtx.GearModel.ListBrandsWithProducts(req.Category)
	if err != nil {
		l.Logger.Errorf("读取装备品牌失败: category=%s err=%v", req.Category, err)
		return &types.GetGearBrandsResp{Success: false, Message: "读取品牌失败", List: []types.GearBrandInfo{}}, nil
	}
	list := make([]types.GearBrandInfo, 0, len(brands))
	for _, brand := range brands {
		list = append(list, types.GearBrandInfo{Id: brand.Id, Name: brand.Name})
	}
	return &types.GetGearBrandsResp{Success: true, Message: "ok", List: list}, nil
}
