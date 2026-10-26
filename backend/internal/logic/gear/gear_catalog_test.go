package gear

import (
	"context"
	"strconv"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func TestGetGearBrandsOnlyReturnsBrandsWithEnabledProducts(t *testing.T) {
	env := newGearEnv(t, "")

	ctx, svcCtx, _ := env.request(1)
	resp, err := NewGetGearBrandsLogic(ctx, svcCtx).GetGearBrands(&types.GetGearBrandsReq{Category: model.GearCategoryCue})
	if err != nil || !resp.Success {
		t.Fatalf("get brands: resp=%+v err=%v", resp, err)
	}
	if len(resp.List) != 1 || resp.List[0].Id != env.c.brandAlpha {
		t.Fatalf("expected only the enabled Alpha brand, got %+v", resp.List)
	}

	chalk, _ := NewGetGearBrandsLogic(env.requestCtx(1)).GetGearBrands(&types.GetGearBrandsReq{Category: model.GearCategoryChalk})
	if !chalk.Success || len(chalk.List) != 1 {
		t.Fatalf("expected Alpha to be listed for chalk, got %+v", chalk)
	}

	invalid, _ := NewGetGearBrandsLogic(env.requestCtx(1)).GetGearBrands(&types.GetGearBrandsReq{Category: "racket"})
	if invalid.Success || invalid.List == nil {
		t.Fatalf("an invalid category must fail with a non-nil list, got %+v", invalid)
	}
}

func TestGetGearEndpointsRejectAdminAndAnonymousTokens(t *testing.T) {
	env := newGearEnv(t, "")
	for name, userId := range map[string]int64{"admin token": -3, "anonymous": 0} {
		ctx, svcCtx, _ := env.request(userId)
		brands, _ := NewGetGearBrandsLogic(ctx, svcCtx).GetGearBrands(&types.GetGearBrandsReq{Category: model.GearCategoryCue})
		products, _ := NewGetGearProductsLogic(ctx, svcCtx).GetGearProducts(&types.GetGearProductsReq{Category: model.GearCategoryCue})
		mine, _ := NewGetGearMineLogic(ctx, svcCtx).GetGearMine()
		if brands.Success || products.Success || mine.Success {
			t.Fatalf("%s must not read the user-side gear endpoints: %+v %+v %+v", name, brands, products, mine)
		}
	}
}

func TestGetGearProductsHidesDisabledAndPagesStably(t *testing.T) {
	env := newGearEnv(t, "")
	list := func(req types.GetGearProductsReq) *types.GetGearProductsResp {
		ctx, svcCtx, _ := env.request(1)
		resp, err := NewGetGearProductsLogic(ctx, svcCtx).GetGearProducts(&req)
		if err != nil || !resp.Success {
			t.Fatalf("get products %+v: resp=%+v err=%v", req, resp, err)
		}
		return resp
	}

	page1 := list(types.GetGearProductsReq{Category: model.GearCategoryCue, Page: 1, PageSize: 2})
	page2 := list(types.GetGearProductsReq{Category: model.GearCategoryCue, Page: 2, PageSize: 2})
	// 启用的 A-100、A-200 和球房公杆；停用型号 A-OFF 与停用品牌下的 G-1 都不出现。
	if page1.Total != 3 || len(page1.List) != 2 || len(page2.List) != 1 {
		t.Fatalf("unexpected paging: total=%d page1=%d page2=%d", page1.Total, len(page1.List), len(page2.List))
	}
	seen := map[int64]bool{}
	for _, p := range append(page1.List, page2.List...) {
		if seen[p.Id] {
			t.Fatalf("product %d repeated across pages", p.Id)
		}
		seen[p.Id] = true
		if p.Id == env.c.cueOff || p.Id == env.c.cueOfOffBrand {
			t.Fatalf("disabled product or disabled brand leaked into the list: %+v", p)
		}
	}
	// 规格与品牌名随型号返回
	var a100 types.GearProductInfo
	for _, p := range page1.List {
		if p.Id == env.c.cuePool {
			a100 = p
		}
	}
	if a100.BrandName != "Alpha" || a100.LengthMm != 1470 || a100.WeightG != 540 || a100.TipDiameterMm != 12.9 || a100.CueStyle != model.GearCueStylePool {
		t.Fatalf("product info must carry the brand name and official specs: %+v", a100)
	}

	byAlias := list(types.GetGearProductsReq{Category: model.GearCategoryCue, Keyword: "阿尔法"})
	if byAlias.Total != 2 {
		t.Fatalf("the brand alias should find both enabled Alpha cues, got %+v", byAlias)
	}

	house := list(types.GetGearProductsReq{Category: model.GearCategoryCue, House: true})
	if house.Total != 1 || !house.List[0].House || house.List[0].BrandName != "" {
		t.Fatalf("house filter should return only the brandless house cue, got %+v", house)
	}

	capped := list(types.GetGearProductsReq{Category: model.GearCategoryCue, PageSize: 500})
	if len(capped.List) > model.GearCatalogMaxPageSize {
		t.Fatalf("page size must be capped at %d, got %d rows", model.GearCatalogMaxPageSize, len(capped.List))
	}
}

func TestGetGearProductsQueryCountDoesNotGrowWithRows(t *testing.T) {
	count := func(extraProducts int) int {
		env := newGearEnv(t, strconv.Itoa(extraProducts))
		m := model.NewGearModel(env.db)
		for i := 0; i < extraProducts; i++ {
			p := model.GearProduct{Category: model.GearCategoryCue, BrandId: env.c.brandAlpha, Name: "X-" + strconv.Itoa(i), CueStyle: model.GearCueStylePool, Sort: 10 + i, Status: 1}
			if err := m.SaveProduct(&p); err != nil {
				t.Fatalf("seed product: %v", err)
			}
		}
		ctx, svcCtx, metrics := env.request(1)
		resp, err := NewGetGearProductsLogic(ctx, svcCtx).GetGearProducts(&types.GetGearProductsReq{Category: model.GearCategoryCue, PageSize: 50})
		if err != nil || !resp.Success {
			t.Fatalf("get products: %+v err=%v", resp, err)
		}
		return int(metrics.Snapshot().SQLCount)
	}
	few, many := count(1), count(45)
	if few != many || few != 3 {
		t.Fatalf("product list must use a count, a page and one brand lookup regardless of rows: few=%d many=%d", few, many)
	}
}

func (e *gearEnv) requestCtx(userId int64) (context.Context, *svc.ServiceContext) {
	ctx, svcCtx, _ := e.request(userId)
	return ctx, svcCtx
}

// ---------- 展示规则（3.7）：按查看者组装，自填名称不出服务端 ----------

func testCatalog() *gearCatalog {
	return newGearCatalog(
		map[int64]model.GearProduct{
			1: {Id: 1, Category: model.GearCategoryCue, BrandId: 10, Name: "A-100", CueStyle: model.GearCueStylePool, LengthMm: 1470, Status: 1},
			2: {Id: 2, Category: model.GearCategoryCue, BrandId: 0, Name: "球房公杆", House: true, Status: 1},
			3: {Id: 3, Category: model.GearCategoryTip, BrandId: 10, Name: "A-Tip", Status: 1},
			4: {Id: 4, Category: model.GearCategoryBreakJump, BrandId: 10, Name: "A-BJ", Roles: "break,jump", Status: 0},
		},
		map[int64]model.GearBrand{10: {Id: 10, Name: "Alpha"}, 11: {Id: 11, Name: "Beta"}},
	)
}

func TestItemInfoShowsOwnCustomNamesOnlyToTheOwner(t *testing.T) {
	catalog := testCatalog()
	item := model.UserGearItem{
		Id: 7, UserId: 1, Category: model.GearCategoryCue, CustomBrandName: "我的小众品牌", CustomModelName: "私人定制 X",
		CueStyle: model.GearCueStyleSnooker, Status: model.GearItemStatusActive,
	}

	owner := catalog.itemInfo(item, true)
	if !owner.Unlisted || owner.BrandName != "我的小众品牌" || owner.ModelName != "私人定制 X" {
		t.Fatalf("the owner must see the names they typed: %+v", owner)
	}
	other := catalog.itemInfo(item, false)
	if !other.Unlisted || other.BrandName != "其他品牌" || other.ModelName != "未收录型号" {
		t.Fatalf("others must see the generic placeholders: %+v", other)
	}
	if other.CueStyle != model.GearCueStyleSnooker || other.Product != nil {
		t.Fatalf("others still need the cue style to pick the default image, and no product: %+v", other)
	}
}

func TestItemInfoUnlistedWithCatalogBrandShowsBrandToEveryone(t *testing.T) {
	catalog := testCatalog()
	item := model.UserGearItem{
		Id: 8, UserId: 1, Category: model.GearCategoryCue, CustomBrandId: 11, CustomModelName: "Beta Secret",
		CueStyle: model.GearCueStylePool, Status: model.GearItemStatusActive,
	}
	owner, other := catalog.itemInfo(item, true), catalog.itemInfo(item, false)
	if owner.BrandName != "Beta" || other.BrandName != "Beta" {
		t.Fatalf("a catalog brand is public information: owner=%q other=%q", owner.BrandName, other.BrandName)
	}
	if owner.ModelName != "Beta Secret" || other.ModelName != "未收录型号" {
		t.Fatalf("the typed model name stays private: owner=%q other=%q", owner.ModelName, other.ModelName)
	}
}

func TestItemInfoLinkedProductIgnoresCustomFieldsAndViewer(t *testing.T) {
	catalog := testCatalog()
	item := model.UserGearItem{
		Id: 9, UserId: 1, Category: model.GearCategoryCue, ProductId: 1,
		CustomBrandName: "旧自填品牌", CustomModelName: "旧自填型号", CueStyle: model.GearCueStyleSnooker,
		TipProductId: 3, Status: model.GearItemStatusRetired,
	}
	for _, viewerIsOwner := range []bool{true, false} {
		info := catalog.itemInfo(item, viewerIsOwner)
		if info.Unlisted || info.Product == nil || info.Product.Id != 1 {
			t.Fatalf("linked items show the catalog product: %+v", info)
		}
		if info.BrandName != "Alpha" || info.ModelName != "A-100" || info.CueStyle != model.GearCueStylePool {
			t.Fatalf("linked items use catalog names and cue style, not the earlier typed ones: %+v", info)
		}
		if info.TipProduct == nil || info.TipProduct.Name != "A-Tip" || info.ChalkProduct != nil {
			t.Fatalf("tip and chalk should be resolved from the catalog: %+v", info)
		}
		if info.Status != model.GearItemStatusRetired {
			t.Fatalf("status must pass through unchanged: %+v", info)
		}
	}
}

func TestItemInfoHouseCueHasNoBrandAndDisabledProductStillResolves(t *testing.T) {
	catalog := testCatalog()
	house := catalog.itemInfo(model.UserGearItem{Id: 1, Category: model.GearCategoryCue, ProductId: 2, Status: 1}, false)
	if !house.House || house.BrandName != "" || house.ModelName != "球房公杆" {
		t.Fatalf("house cue should have no brand: %+v", house)
	}
	disabled := catalog.itemInfo(model.UserGearItem{Id: 2, Category: model.GearCategoryBreakJump, ProductId: 4, Status: 1}, false)
	if disabled.Product == nil || disabled.ModelName != "A-BJ" || len(disabled.Roles) != 2 {
		t.Fatalf("history must still resolve a product that was later disabled: %+v", disabled)
	}
}
