package admin

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newGearAdminSvc(t *testing.T) (*svc.ServiceContext, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := testsupport.PrepareGearSchema(db); err != nil {
		t.Fatalf("prepare gear schema: %v", err)
	}
	return &svc.ServiceContext{GearModel: model.NewGearModel(db)}, db
}

func adminGearCtx() context.Context {
	return context.WithValue(context.Background(), "user_id", int64(-1))
}
func userGearCtx() context.Context {
	return context.WithValue(context.Background(), "user_id", int64(42))
}
func anonymousGearCtx() context.Context { return context.Background() }

func TestAdminGearEndpointsRejectUserAndAnonymousTokens(t *testing.T) {
	svcCtx, db := newGearAdminSvc(t)
	gears := model.NewGearModel(db)
	brand := model.GearBrand{Name: "Seed", Status: 1}
	if err := gears.SaveBrand(&brand); err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	product := model.GearProduct{Category: model.GearCategoryCue, BrandId: brand.Id, Name: "Seed-1", CueStyle: model.GearCueStylePool, Status: 1}
	if err := gears.SaveProduct(&product); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	pending := model.UserGearItem{UserId: 1, Category: model.GearCategoryCue, CustomKey: "cue|nx|y", CustomBrandName: "x", CustomModelName: "y", Status: 1}
	if err := gears.CreateItemWithTx(nil, &pending); err != nil {
		t.Fatalf("seed pending: %v", err)
	}

	for name, ctx := range map[string]context.Context{"user token": userGearCtx(), "anonymous": anonymousGearCtx()} {
		brands, _ := NewAdminGetGearBrandListLogic(ctx, svcCtx).AdminGetGearBrandList(&types.AdminGearBrandListReq{})
		products, _ := NewAdminGetGearProductListLogic(ctx, svcCtx).AdminGetGearProductList(&types.AdminGearProductListReq{Status: -1})
		pendings, _ := NewAdminGetGearPendingListLogic(ctx, svcCtx).AdminGetGearPendingList(&types.AdminGearPendingListReq{})
		for label, got := range map[string]struct {
			code    int
			success bool
			total   int64
			listLen int
		}{
			"brand list":   {brands.Code, brands.Success, brands.Total, len(brands.List)},
			"product list": {products.Code, products.Success, products.Total, len(products.List)},
			"pending list": {pendings.Code, pendings.Success, pendings.Total, len(pendings.List)},
		} {
			if got.code != 401 || got.success || got.total != 0 || got.listLen != 0 {
				t.Fatalf("%s: %s must be refused without leaking rows: %+v", name, label, got)
			}
		}

		saveBrand, _ := NewAdminSaveGearBrandLogic(ctx, svcCtx).AdminSaveGearBrand(&types.AdminGearBrandSaveReq{Name: "Hacked", Status: 1})
		saveProduct, _ := NewAdminSaveGearProductLogic(ctx, svcCtx).AdminSaveGearProduct(&types.AdminGearProductSaveReq{Category: model.GearCategoryTip, BrandId: brand.Id, Name: "Hacked", Status: 1})
		link, _ := NewAdminLinkGearPendingLogic(ctx, svcCtx).AdminLinkGearPending(&types.AdminGearPendingLinkReq{CustomKey: pending.CustomKey, ProductId: product.Id})
		if saveBrand.Code != 401 || saveProduct.Code != 401 || link.Code != 401 || saveBrand.Success || saveProduct.Success || link.Success {
			t.Fatalf("%s: writes must be refused: %+v %+v %+v", name, saveBrand, saveProduct, link)
		}
	}

	var brandCount, productCount int64
	db.Model(&model.GearBrand{}).Count(&brandCount)
	db.Model(&model.GearProduct{}).Where("name = ?", "Hacked").Count(&productCount)
	var stillPending model.UserGearItem
	db.First(&stillPending, pending.Id)
	if brandCount != 1 || productCount != 0 || stillPending.ProductId != 0 {
		t.Fatalf("a refused request must not change anything: brands=%d hacked=%d pendingProduct=%d", brandCount, productCount, stillPending.ProductId)
	}

	if ok, _ := NewAdminGetGearBrandListLogic(adminGearCtx(), svcCtx).AdminGetGearBrandList(&types.AdminGearBrandListReq{}); !ok.Success || ok.Total != 1 {
		t.Fatalf("the admin token itself must work: %+v", ok)
	}
}

func TestAdminSaveGearBrandValidatesAndRejectsDuplicates(t *testing.T) {
	svcCtx, _ := newGearAdminSvc(t)
	save := func(req types.AdminGearBrandSaveReq) *types.AdminGearSaveResp {
		resp, err := NewAdminSaveGearBrandLogic(adminGearCtx(), svcCtx).AdminSaveGearBrand(&req)
		if err != nil {
			t.Fatalf("save brand returned an error: %v", err)
		}
		return resp
	}

	created := save(types.AdminGearBrandSaveReq{Name: "  Alpha  ", Aliases: " 阿尔法 ，, alp ,", Sort: 3, Status: 1})
	if !created.Success || created.Id == 0 {
		t.Fatalf("create brand: %+v", created)
	}
	stored, _ := svcCtx.GearModel.FindBrand(created.Id)
	if stored.Name != "Alpha" || stored.Aliases != "阿尔法,alp" || stored.Sort != 3 {
		t.Fatalf("name must be trimmed and aliases normalized: %+v", stored)
	}

	if dup := save(types.AdminGearBrandSaveReq{Name: "Alpha", Status: 1}); dup.Success || dup.Code != 409 {
		t.Fatalf("a duplicate brand name must conflict: %+v", dup)
	}
	renamed := save(types.AdminGearBrandSaveReq{Id: created.Id, Name: "Alpha Pro", Status: 0})
	if !renamed.Success || renamed.Id != created.Id {
		t.Fatalf("update brand: %+v", renamed)
	}
	stored, _ = svcCtx.GearModel.FindBrand(created.Id)
	if stored.Name != "Alpha Pro" || stored.Status != 0 || stored.CreatedAt.IsZero() {
		t.Fatalf("update must keep created_at and apply the disable: %+v", stored)
	}
	if again := save(types.AdminGearBrandSaveReq{Id: created.Id, Name: "Alpha Pro", Status: 0}); !again.Success {
		t.Fatalf("saving a brand with its own name must not conflict with itself: %+v", again)
	}

	for name, req := range map[string]types.AdminGearBrandSaveReq{
		"empty name":       {Name: "   ", Status: 1},
		"name too long":    {Name: strings.Repeat("牌", 65), Status: 1},
		"aliases too long": {Name: "B", Aliases: strings.Repeat("别", 256), Status: 1},
		"bad status":       {Name: "C", Status: 2},
		"unknown id":       {Id: 999, Name: "D", Status: 1},
	} {
		if resp := save(req); resp.Success {
			t.Fatalf("%s should be rejected: %+v", name, resp)
		}
	}
}

func TestAdminSaveGearProductValidation(t *testing.T) {
	svcCtx, db := newGearAdminSvc(t)
	brand := model.GearBrand{Name: "Alpha", Status: 1}
	if err := svcCtx.GearModel.SaveBrand(&brand); err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	save := func(req types.AdminGearProductSaveReq) *types.AdminGearSaveResp {
		resp, err := NewAdminSaveGearProductLogic(adminGearCtx(), svcCtx).AdminSaveGearProduct(&req)
		if err != nil {
			t.Fatalf("save product returned an error: %v", err)
		}
		return resp
	}
	b := brand.Id

	valid := map[string]types.AdminGearProductSaveReq{
		"pool cue with specs": {Category: model.GearCategoryCue, BrandId: b, Name: "P-1", CueStyle: model.GearCueStylePool, LengthMm: 1470, WeightG: 540, TipDiameterMm: 12.9, Status: 1},
		"snooker cue":         {Category: model.GearCategoryCue, BrandId: b, Name: "S-1", CueStyle: model.GearCueStyleSnooker, Status: 1},
		"house cue":           {Category: model.GearCategoryCue, BrandId: 0, Name: "另一种公杆", Status: 1},
		"house chalk":         {Category: model.GearCategoryChalk, BrandId: 0, Name: "另一种球房巧克", Status: 1},
		"break cue":           {Category: model.GearCategoryBreakJump, BrandId: b, Name: "B-1", Roles: []string{"break"}, Status: 1},
		"dual cue":            {Category: model.GearCategoryBreakJump, BrandId: b, Name: "BJ-1", Roles: []string{"jump", "break"}, Status: 1},
		"tip":                 {Category: model.GearCategoryTip, BrandId: b, Name: "T-1", Status: 1},
		"chalk":               {Category: model.GearCategoryChalk, BrandId: b, Name: "C-1", Status: 0},
		"image png":           {Category: model.GearCategoryTip, BrandId: b, Name: "T-2", ImagePath: "/static/images/gear/tip-2.png", Status: 1},
		"image webp":          {Category: model.GearCategoryTip, BrandId: b, Name: "T-3", ImagePath: "/static/images/gear/T_3-v2.webp", Status: 1},
	}
	for name, req := range valid {
		if resp := save(req); !resp.Success || resp.Id == 0 {
			t.Fatalf("%s should be accepted: %+v", name, resp)
		}
	}

	invalid := map[string]types.AdminGearProductSaveReq{
		"category":              {Category: "racket", BrandId: b, Name: "X", Status: 1},
		"empty name":            {Category: model.GearCategoryTip, BrandId: b, Name: " ", Status: 1},
		"name too long":         {Category: model.GearCategoryTip, BrandId: b, Name: strings.Repeat("型", 129), Status: 1},
		"bad status":            {Category: model.GearCategoryTip, BrandId: b, Name: "X", Status: 5},
		"unknown brand":         {Category: model.GearCategoryTip, BrandId: 999, Name: "X", Status: 1},
		"negative brand":        {Category: model.GearCategoryTip, BrandId: -1, Name: "X", Status: 1},
		"brandless tip":         {Category: model.GearCategoryTip, BrandId: 0, Name: "X", Status: 1},
		"brandless break cue":   {Category: model.GearCategoryBreakJump, BrandId: 0, Name: "X", Roles: []string{"break"}, Status: 1},
		"cue without style":     {Category: model.GearCategoryCue, BrandId: b, Name: "X", Status: 1},
		"cue with bad style":    {Category: model.GearCategoryCue, BrandId: b, Name: "X", CueStyle: "carom", Status: 1},
		"cue with roles":        {Category: model.GearCategoryCue, BrandId: b, Name: "X", CueStyle: model.GearCueStylePool, Roles: []string{"break"}, Status: 1},
		"break/jump no roles":   {Category: model.GearCategoryBreakJump, BrandId: b, Name: "X", Status: 1},
		"break/jump bad roles":  {Category: model.GearCategoryBreakJump, BrandId: b, Name: "X", Roles: []string{"fly"}, Status: 1},
		"break/jump with style": {Category: model.GearCategoryBreakJump, BrandId: b, Name: "X", Roles: []string{"break"}, CueStyle: model.GearCueStylePool, Status: 1},
		"tip with style":        {Category: model.GearCategoryTip, BrandId: b, Name: "X", CueStyle: model.GearCueStylePool, Status: 1},
		"chalk with roles":      {Category: model.GearCategoryChalk, BrandId: b, Name: "X", Roles: []string{"jump"}, Status: 1},
		"negative length":       {Category: model.GearCategoryTip, BrandId: b, Name: "X", LengthMm: -1, Status: 1},
		"too long":              {Category: model.GearCategoryTip, BrandId: b, Name: "X", LengthMm: 3001, Status: 1},
		"too heavy":             {Category: model.GearCategoryTip, BrandId: b, Name: "X", WeightG: 2001, Status: 1},
		"tip diameter":          {Category: model.GearCategoryTip, BrandId: b, Name: "X", TipDiameterMm: 31, Status: 1},
		"external image":        {Category: model.GearCategoryTip, BrandId: b, Name: "X", ImagePath: "https://evil.example/a.png", Status: 1},
		"image traversal":       {Category: model.GearCategoryTip, BrandId: b, Name: "X", ImagePath: "/static/images/gear/../ranks/a.png", Status: 1},
		"image subdirectory":    {Category: model.GearCategoryTip, BrandId: b, Name: "X", ImagePath: "/static/images/gear/sub/a.png", Status: 1},
		"image other directory": {Category: model.GearCategoryTip, BrandId: b, Name: "X", ImagePath: "/static/images/ranks/a.png", Status: 1},
		"image wrong extension": {Category: model.GearCategoryTip, BrandId: b, Name: "X", ImagePath: "/static/images/gear/a.gif", Status: 1},
		"image hidden file":     {Category: model.GearCategoryTip, BrandId: b, Name: "X", ImagePath: "/static/images/gear/.png", Status: 1},
		"duplicate":             {Category: model.GearCategoryCue, BrandId: b, Name: "P-1", CueStyle: model.GearCueStylePool, Status: 1},
	}
	for name, req := range invalid {
		if resp := save(req); resp.Success {
			t.Fatalf("%s should be rejected: %+v", name, resp)
		}
	}

	var house model.GearProduct
	db.Where("name = ?", "另一种公杆").First(&house)
	if !house.House || house.BrandId != 0 {
		t.Fatalf("a brandless save must be flagged as a house entry: %+v", house)
	}
	var withSpecs model.GearProduct
	db.Where("name = ?", "P-1").First(&withSpecs)
	if withSpecs.LengthMm != 1470 || withSpecs.WeightG != 540 || withSpecs.TipDiameterMm != 12.9 {
		t.Fatalf("official specs must be stored as given: %+v", withSpecs)
	}
}

func TestAdminSaveGearProductUpdateKeepsCategoryAndAppliesChanges(t *testing.T) {
	svcCtx, _ := newGearAdminSvc(t)
	brand := model.GearBrand{Name: "Alpha", Status: 1}
	if err := svcCtx.GearModel.SaveBrand(&brand); err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	save := func(req types.AdminGearProductSaveReq) *types.AdminGearSaveResp {
		resp, _ := NewAdminSaveGearProductLogic(adminGearCtx(), svcCtx).AdminSaveGearProduct(&req)
		return resp
	}
	created := save(types.AdminGearProductSaveReq{Category: model.GearCategoryCue, BrandId: brand.Id, Name: "A-1", CueStyle: model.GearCueStylePool, WeightG: 540, Status: 1})
	if !created.Success {
		t.Fatalf("create: %+v", created)
	}

	fixed := save(types.AdminGearProductSaveReq{Id: created.Id, Category: model.GearCategoryCue, BrandId: brand.Id, Name: "A-1", CueStyle: model.GearCueStylePool, WeightG: 525, Status: 0})
	if !fixed.Success || fixed.Id != created.Id {
		t.Fatalf("correct the official weight and disable: %+v", fixed)
	}
	stored, _ := svcCtx.GearModel.FindProduct(created.Id)
	if stored.WeightG != 525 || stored.Status != 0 || stored.CreatedAt.IsZero() {
		t.Fatalf("the update should apply and keep created_at: %+v", stored)
	}

	if changed := save(types.AdminGearProductSaveReq{Id: created.Id, Category: model.GearCategoryTip, BrandId: brand.Id, Name: "A-1", Status: 1}); changed.Success {
		t.Fatalf("changing the category of an existing product must be refused: %+v", changed)
	}
	if missing := save(types.AdminGearProductSaveReq{Id: 999, Category: model.GearCategoryCue, BrandId: brand.Id, Name: "Z", CueStyle: model.GearCueStylePool, Status: 1}); missing.Success || missing.Code != 404 {
		t.Fatalf("an unknown id is not found: %+v", missing)
	}
}

func TestAdminGetGearProductListFiltersAndNamesBrands(t *testing.T) {
	svcCtx, _ := newGearAdminSvc(t)
	gears := svcCtx.GearModel
	alpha, beta := model.GearBrand{Name: "Alpha", Aliases: "阿尔法", Status: 1}, model.GearBrand{Name: "Beta", Status: 1}
	for _, b := range []*model.GearBrand{&alpha, &beta} {
		if err := gears.SaveBrand(b); err != nil {
			t.Fatalf("seed brand: %v", err)
		}
	}
	for _, p := range []model.GearProduct{
		{Category: model.GearCategoryCue, BrandId: alpha.Id, Name: "A-1", CueStyle: model.GearCueStylePool, Status: 1},
		{Category: model.GearCategoryCue, BrandId: alpha.Id, Name: "A-2", CueStyle: model.GearCueStylePool, Status: 0},
		{Category: model.GearCategoryTip, BrandId: beta.Id, Name: "B-T", Status: 1},
	} {
		p := p
		if err := gears.SaveProduct(&p); err != nil {
			t.Fatalf("seed product: %v", err)
		}
	}
	list := func(req types.AdminGearProductListReq) *types.AdminGearProductListResp {
		resp, err := NewAdminGetGearProductListLogic(adminGearCtx(), svcCtx).AdminGetGearProductList(&req)
		if err != nil || !resp.Success {
			t.Fatalf("list %+v: %+v err=%v", req, resp, err)
		}
		return resp
	}

	all := list(types.AdminGearProductListReq{Status: -1, PageSize: 1000})
	// 含迁移预置的球房公杆与球房巧克
	if all.Total != 5 || len(all.List) != 5 {
		t.Fatalf("admin sees every product including disabled ones: %d %+v", all.Total, all.List)
	}
	cues := list(types.AdminGearProductListReq{Category: model.GearCategoryCue, BrandId: alpha.Id, Status: -1})
	if cues.Total != 2 || cues.List[0].BrandName != "Alpha" {
		t.Fatalf("filter by category and brand: %+v", cues)
	}
	if disabled := list(types.AdminGearProductListReq{Status: 0}); disabled.Total != 1 || disabled.List[0].Name != "A-2" {
		t.Fatalf("filter disabled only: %+v", disabled)
	}
	if byAlias := list(types.AdminGearProductListReq{Keyword: "阿尔法", Status: -1}); byAlias.Total != 2 {
		t.Fatalf("keyword matches the brand alias: %+v", byAlias)
	}
	for name, req := range map[string]types.AdminGearProductListReq{
		"bad category": {Category: "racket", Status: -1},
		"bad status":   {Status: 7},
	} {
		resp, _ := NewAdminGetGearProductListLogic(adminGearCtx(), svcCtx).AdminGetGearProductList(&req)
		if resp.Success {
			t.Fatalf("%s should be rejected: %+v", name, resp)
		}
	}
}

func TestAdminGearPendingListGroupsAndLinksInBatches(t *testing.T) {
	svcCtx, db := newGearAdminSvc(t)
	gears := svcCtx.GearModel
	alpha := model.GearBrand{Name: "Alpha", Status: 1}
	if err := gears.SaveBrand(&alpha); err != nil {
		t.Fatalf("seed brand: %v", err)
	}
	target := model.GearProduct{Category: model.GearCategoryCue, BrandId: alpha.Id, Name: "A-Real", CueStyle: model.GearCueStylePool, Status: 1}
	disabled := model.GearProduct{Category: model.GearCategoryCue, BrandId: alpha.Id, Name: "A-Off", CueStyle: model.GearCueStylePool, Status: 0}
	tip := model.GearProduct{Category: model.GearCategoryTip, BrandId: alpha.Id, Name: "A-Tip", Status: 1}
	for _, p := range []*model.GearProduct{&target, &disabled, &tip} {
		if err := gears.SaveProduct(p); err != nil {
			t.Fatalf("seed product: %v", err)
		}
	}

	add := func(userId int64, brandId int64, brandName, modelName string) model.UserGearItem {
		item := model.UserGearItem{
			UserId: userId, Category: model.GearCategoryCue, CustomBrandId: brandId, CustomBrandName: brandName, CustomModelName: modelName,
			CustomKey: model.BuildGearCustomKey(model.GearCategoryCue, brandId, brandName, modelName), CueStyle: model.GearCueStylePool, Status: 1,
		}
		if err := gears.CreateItemWithTx(nil, &item); err != nil {
			t.Fatalf("seed item: %v", err)
		}
		return item
	}
	add(1, alpha.Id, "", "Alpha Real")
	last := add(2, alpha.Id, "", "alpha real")
	add(3, 0, "Solo", "Solo 9")

	list, err := NewAdminGetGearPendingListLogic(adminGearCtx(), svcCtx).AdminGetGearPendingList(&types.AdminGearPendingListReq{PageSize: 10})
	if err != nil || !list.Success || list.Total != 2 || len(list.List) != 2 {
		t.Fatalf("pending list: %+v err=%v", list, err)
	}
	top := list.List[0]
	if top.Count != 2 || top.CustomKey != last.CustomKey || top.BrandName != "Alpha" || top.ModelName != "alpha real" || top.Category != model.GearCategoryCue {
		t.Fatalf("the biggest group comes first with the catalog brand name resolved: %+v", top)
	}
	if list.List[1].BrandName != "Solo" {
		t.Fatalf("a typed brand is shown as typed to the admin: %+v", list.List[1])
	}
	if bad, _ := NewAdminGetGearPendingListLogic(adminGearCtx(), svcCtx).AdminGetGearPendingList(&types.AdminGearPendingListReq{Category: model.GearCategoryTip}); bad.Success {
		t.Fatalf("tips never appear in the pending list: %+v", bad)
	}

	link := func(key string, productId int64) *types.AdminGearPendingLinkResp {
		resp, err := NewAdminLinkGearPendingLogic(adminGearCtx(), svcCtx).AdminLinkGearPending(&types.AdminGearPendingLinkReq{CustomKey: key, ProductId: productId})
		if err != nil {
			t.Fatalf("link returned an error: %v", err)
		}
		return resp
	}
	if resp := link(top.CustomKey, tip.Id); resp.Success {
		t.Fatalf("a different category must be refused: %+v", resp)
	}
	if resp := link(top.CustomKey, disabled.Id); resp.Success {
		t.Fatalf("a disabled target must be refused: %+v", resp)
	}
	if resp := link(top.CustomKey, 999); resp.Success || resp.Code != 404 {
		t.Fatalf("an unknown target is not found: %+v", resp)
	}
	if resp := link("tip|nx|y", target.Id); resp.Success {
		t.Fatalf("a key from a non-pending category is invalid: %+v", resp)
	}
	if resp := link("cue|nnothing|here", target.Id); resp.Success || resp.Code != 404 {
		t.Fatalf("an unknown group is not found: %+v", resp)
	}
	if resp, _ := NewAdminLinkGearPendingLogic(adminGearCtx(), svcCtx).AdminLinkGearPending(nil); resp.Success {
		t.Fatalf("a nil request is invalid: %+v", resp)
	}

	ok := link(top.CustomKey, target.Id)
	if !ok.Success || ok.Updated != 2 || ok.Remaining != 0 {
		t.Fatalf("linking the group: %+v", ok)
	}
	var linked model.UserGearItem
	db.First(&linked, last.Id)
	if linked.ProductId != target.Id || linked.CustomModelName != "alpha real" {
		t.Fatalf("link sets the product and keeps what the user typed: %+v", linked)
	}
	if again := link(top.CustomKey, target.Id); again.Success || again.Code != 404 {
		t.Fatalf("a fully linked group is gone: %+v", again)
	}
	after, _ := NewAdminGetGearPendingListLogic(adminGearCtx(), svcCtx).AdminGetGearPendingList(&types.AdminGearPendingListReq{})
	if after.Total != 1 {
		t.Fatalf("the linked group leaves the pending list: %+v", after)
	}

	// 超过一批的分组分两次关联完
	for i := 0; i < model.GearPendingLinkBatchSize+1; i++ {
		add(int64(100+i), 0, "Big", fmt.Sprintf("Mass %s", "Model"))
	}
	bigKey := model.BuildGearCustomKey(model.GearCategoryCue, 0, "Big", "Mass Model")
	first := link(bigKey, target.Id)
	if !first.Success || first.Updated != int64(model.GearPendingLinkBatchSize) || first.Remaining != 1 || !strings.Contains(first.Message, "再次执行") {
		t.Fatalf("the first batch leaves one behind and says so: %+v", first)
	}
	second := link(bigKey, target.Id)
	if !second.Success || second.Updated != 1 || second.Remaining != 0 {
		t.Fatalf("the second round finishes the group: %+v", second)
	}
}
