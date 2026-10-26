package gear

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetGearProductsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取装备目录型号
func NewGetGearProductsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGearProductsLogic {
	return &GetGearProductsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetGearProductsLogic) GetGearProducts(req *types.GetGearProductsReq) (resp *types.GetGearProductsResp, err error) {
	failed := func(message string) *types.GetGearProductsResp {
		return &types.GetGearProductsResp{Success: false, Message: message, List: []types.GearProductInfo{}}
	}
	if _, err := utils.GetUserIDFromCtx(l.ctx); err != nil {
		return failed("用户未登录"), nil
	}
	if req == nil || !model.IsGearCategory(req.Category) {
		return failed("装备类别无效"), nil
	}

	page, pageSize := normalizeCatalogPage(req.Page, req.PageSize)
	products, total, err := l.svcCtx.GearModel.ListProductsPage(model.GearProductFilter{
		Category:         req.Category,
		BrandId:          req.BrandId,
		Keyword:          truncateRunes(req.Keyword, keywordMaxRunes),
		HouseOnly:        req.House,
		Status:           1,
		OnlyEnabledBrand: true,
		Page:             page,
		PageSize:         pageSize,
	})
	if err != nil {
		l.Logger.Errorf("读取装备型号失败: category=%s err=%v", req.Category, err)
		return failed("读取型号失败"), nil
	}

	brandIds := make([]int64, 0, len(products))
	productMap := make(map[int64]model.GearProduct, len(products))
	for _, product := range products {
		brandIds = append(brandIds, product.BrandId)
		productMap[product.Id] = product
	}
	brands, err := l.svcCtx.GearModel.FindBrandsByIds(brandIds)
	if err != nil {
		l.Logger.Errorf("读取装备品牌名称失败: err=%v", err)
		return failed("读取型号失败"), nil
	}
	catalog := newGearCatalog(productMap, brands)

	list := make([]types.GearProductInfo, 0, len(products))
	for _, product := range products {
		list = append(list, catalog.productInfo(product))
	}
	return &types.GetGearProductsResp{Success: true, Message: "ok", Total: total, List: list}, nil
}
