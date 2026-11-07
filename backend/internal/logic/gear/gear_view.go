package gear

import (
	"strings"
	"unicode/utf8"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

const (
	// 他人看到未收录装备时的固定展示，自填内容不会离开服务端。
	unlistedModelName = "未收录型号"
	otherBrandName    = "其他品牌"

	catalogPageSizeDefault = 20
	keywordMaxRunes        = 32
)

// gearCatalog 是一次请求里用到的目录型号与品牌，由固定的两条批量查询读出。
type gearCatalog struct {
	products map[int64]model.GearProduct
	brands   map[int64]model.GearBrand
}

func newGearCatalog(products map[int64]model.GearProduct, brands map[int64]model.GearBrand) *gearCatalog {
	if products == nil {
		products = map[int64]model.GearProduct{}
	}
	if brands == nil {
		brands = map[int64]model.GearBrand{}
	}
	return &gearCatalog{products: products, brands: brands}
}

// loadGearCatalog 一次读出装备引用到的全部型号（含皮头、巧克）以及它们和自填品牌引用的品牌。
func loadGearCatalog(gears *model.GearModel, items []model.UserGearItem, extraProductIds ...int64) (*gearCatalog, error) {
	productIds := append([]int64{}, extraProductIds...)
	brandIds := make([]int64, 0, len(items))
	for _, item := range items {
		productIds = append(productIds, item.ProductId, item.TipProductId, item.ChalkProductId)
		brandIds = append(brandIds, item.CustomBrandId)
	}
	products, err := gears.FindProductsByIds(productIds)
	if err != nil {
		return nil, err
	}
	for _, product := range products {
		brandIds = append(brandIds, product.BrandId)
	}
	brands, err := gears.FindBrandsByIds(brandIds)
	if err != nil {
		return nil, err
	}
	return newGearCatalog(products, brands), nil
}

func (c *gearCatalog) brandName(brandId int64) string {
	if brandId <= 0 {
		return ""
	}
	return c.brands[brandId].Name
}

func (c *gearCatalog) productInfo(product model.GearProduct) types.GearProductInfo {
	return types.GearProductInfo{
		Id:            product.Id,
		Category:      product.Category,
		BrandId:       product.BrandId,
		BrandName:     c.brandName(product.BrandId),
		Name:          product.Name,
		House:         product.House,
		CueStyle:      product.CueStyle,
		Roles:         model.ParseGearRoles(product.Roles),
		LengthMm:      product.LengthMm,
		WeightG:       product.WeightG,
		TipDiameterMm: product.TipDiameterMm,
		ImagePath:     product.ImagePath,
	}
}

// productInfoByID 对 0 或目录里已不存在的型号返回 nil。
func (c *gearCatalog) productInfoByID(id int64) *types.GearProductInfo {
	if id <= 0 {
		return nil
	}
	product, ok := c.products[id]
	if !ok {
		return nil
	}
	info := c.productInfo(product)
	return &info
}

// itemInfo 按查看者组装一件装备的展示字段。
// 已关联目录型号的装备一律展示目录信息；未收录装备只有本人能看到自填的品牌与型号名，
// 其他人看到「品牌 · 未收录型号」，自填品牌显示为「其他品牌」。
func (c *gearCatalog) itemInfo(item model.UserGearItem, viewerIsOwner bool) types.GearItemInfo {
	info := types.GearItemInfo{
		Id:           item.Id,
		Category:     item.Category,
		Status:       item.Status,
		CueStyle:     item.CueStyle,
		Roles:        model.ParseGearRoles(item.Roles),
		TipProduct:   c.productInfoByID(item.TipProductId),
		ChalkProduct: c.productInfoByID(item.ChalkProductId),
		CreatedAt:    logicx.FormatUTC8Time(item.CreatedAt),
	}

	if product := c.productInfoByID(item.ProductId); product != nil {
		info.Product = product
		info.House = product.House
		info.BrandName = product.BrandName
		info.ModelName = product.Name
		info.CueStyle = product.CueStyle
		info.Roles = product.Roles
		return info
	}

	info.Unlisted = true
	catalogBrand := c.brandName(item.CustomBrandId)
	switch {
	case viewerIsOwner:
		info.BrandName = catalogBrand
		if info.BrandName == "" {
			info.BrandName = item.CustomBrandName
		}
		info.ModelName = item.CustomModelName
	default:
		info.BrandName = catalogBrand
		if info.BrandName == "" {
			info.BrandName = otherBrandName
		}
		info.ModelName = unlistedModelName
	}
	return info
}

func (c *gearCatalog) itemInfos(items []model.UserGearItem, viewerIsOwner bool) []types.GearItemInfo {
	list := make([]types.GearItemInfo, 0, len(items))
	for _, item := range items {
		list = append(list, c.itemInfo(item, viewerIsOwner))
	}
	return list
}

func defaultInfos(defaults []model.UserGearDefault) []types.GearDefaultInfo {
	list := make([]types.GearDefaultInfo, 0, len(defaults))
	for _, d := range defaults {
		list = append(list, types.GearDefaultInfo{GameType: d.GameType, Slot: d.Slot, ItemId: d.ItemId})
	}
	return list
}

func normalizeCatalogPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = catalogPageSizeDefault
	}
	if pageSize > model.GearCatalogMaxPageSize {
		pageSize = model.GearCatalogMaxPageSize
	}
	return page, pageSize
}

func truncateRunes(text string, limit int) string {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit])
}

// supportsBreak 返回该球种是否记录冲杆：九球追分、中式八球、美式九球记录，斯诺克不记录。
func supportsBreak(gameType int) bool {
	return model.GearGameTypeRecordsBreak(gameType)
}

func isValidGameType(gameType int) bool {
	return model.IsGearGameType(gameType)
}
