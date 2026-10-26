package admin

import (
	"context"
	"strings"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetGearProductListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取装备型号列表（管理员）
func NewAdminGetGearProductListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetGearProductListLogic {
	return &AdminGetGearProductListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetGearProductListLogic) AdminGetGearProductList(req *types.AdminGearProductListReq) (resp *types.AdminGearProductListResp, err error) {
	fail := func(code int, message string) *types.AdminGearProductListResp {
		return &types.AdminGearProductListResp{Code: code, Success: false, Message: message, List: []types.AdminGearProductInfo{}}
	}
	if !requireGearAdmin(l.ctx) {
		return fail(401, "未登录或登录已过期"), nil
	}
	if req == nil {
		req = &types.AdminGearProductListReq{Status: -1}
	}
	if req.Category != "" && !model.IsGearCategory(req.Category) {
		return fail(400, "装备类别无效"), nil
	}
	if req.Status < -1 || req.Status > 1 {
		return fail(400, "状态无效"), nil
	}

	page, pageSize := normalizeAdminGearPage(req.Page, req.PageSize)
	products, total, err := l.svcCtx.GearModel.ListProductsPage(model.GearProductFilter{
		Category: req.Category,
		BrandId:  req.BrandId,
		Keyword:  strings.TrimSpace(req.Keyword),
		Status:   req.Status,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		l.Logger.Errorf("读取装备型号列表失败: %v", err)
		return fail(500, "读取型号失败"), nil
	}

	brandIds := make([]int64, 0, len(products))
	for _, product := range products {
		brandIds = append(brandIds, product.BrandId)
	}
	brands, err := l.svcCtx.GearModel.FindBrandsByIds(brandIds)
	if err != nil {
		l.Logger.Errorf("读取装备品牌名称失败: %v", err)
		return fail(500, "读取型号失败"), nil
	}

	list := make([]types.AdminGearProductInfo, 0, len(products))
	for _, product := range products {
		list = append(list, types.AdminGearProductInfo{
			Id:            product.Id,
			Category:      product.Category,
			BrandId:       product.BrandId,
			BrandName:     brands[product.BrandId].Name,
			Name:          product.Name,
			House:         product.House,
			CueStyle:      product.CueStyle,
			Roles:         model.ParseGearRoles(product.Roles),
			LengthMm:      product.LengthMm,
			WeightG:       product.WeightG,
			TipDiameterMm: product.TipDiameterMm,
			ImagePath:     product.ImagePath,
			Sort:          product.Sort,
			Status:        product.Status,
		})
	}
	return &types.AdminGearProductListResp{Code: 0, Success: true, Message: "ok", Total: total, List: list}, nil
}
