package model

import (
	"fmt"
	"reflect"
	"testing"

	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newGearTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := testsupport.PrepareGearSchema(db); err != nil {
		t.Fatalf("prepare gear schema: %v", err)
	}
	return db
}

func TestNormalizeGearTextMergesCaseWidthAndWhitespace(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Brand X  Pro 1", "brand x pro 1"},
		{"  brand x pro 1 ", "brand x pro 1"},
		{"ＢＲＡＮＤ　Ｘ　Ｐｒｏ１", "brand x pro1"},
		{"Ｐｒｏ　　１\t", "pro 1"},
		{"美洲豹  Ｊ１", "美洲豹 j1"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := NormalizeGearText(tc.in); got != tc.want {
			t.Fatalf("NormalizeGearText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBuildGearCustomKeyUsesBrandIDOrNormalizedName(t *testing.T) {
	a := BuildGearCustomKey(GearCategoryCue, 0, "Brand X", "Brand X  Pro 1")
	b := BuildGearCustomKey(GearCategoryCue, 0, " brand x ", "brand x pro 1")
	if a != b {
		t.Fatalf("expected normalized keys to match: %q vs %q", a, b)
	}
	if got := BuildGearCustomKey(GearCategoryCue, 12, "ignored", "Pro 1"); got != "cue|b12|pro 1" {
		t.Fatalf("expected catalog brand key to use the brand id, got %q", got)
	}
	if BuildGearCustomKey(GearCategoryCue, 0, "x", "m") == BuildGearCustomKey(GearCategoryBreakJump, 0, "x", "m") {
		t.Fatal("keys of different categories must differ")
	}
	if got := GearCustomKeyCategory(a); got != GearCategoryCue {
		t.Fatalf("GearCustomKeyCategory(%q) = %q", a, got)
	}
}

func TestNormalizeGearRolesDedupesAndRejectsUnknown(t *testing.T) {
	joined, ok := NormalizeGearRoles([]string{"jump", "break", "jump", ""})
	if !ok || joined != "break,jump" {
		t.Fatalf("expected canonical order break,jump, got %q ok=%v", joined, ok)
	}
	if _, ok := NormalizeGearRoles([]string{"break", "fly"}); ok {
		t.Fatal("unknown roles must be rejected")
	}
	if joined, ok := NormalizeGearRoles(nil); !ok || joined != "" {
		t.Fatalf("empty roles should be valid and empty, got %q ok=%v", joined, ok)
	}
	if got := ParseGearRoles("break, jump,fly"); !reflect.DeepEqual(got, []string{"break", "jump"}) {
		t.Fatalf("ParseGearRoles dropped or kept the wrong roles: %v", got)
	}
	if !GearRolesContain("break,jump", GearRoleJump) || GearRolesContain("break", GearRoleJump) {
		t.Fatal("GearRolesContain mismatch")
	}
}

func seedGearCatalog(t *testing.T, db *gorm.DB) (brandA, brandB GearBrand) {
	t.Helper()
	m := NewGearModel(db)
	brandA = GearBrand{Name: "Alpha", Aliases: "阿尔法,alp", Sort: 1, Status: 1}
	brandB = GearBrand{Name: "Beta", Aliases: "", Sort: 2, Status: 1}
	disabled := GearBrand{Name: "Gamma", Sort: 3, Status: 0}
	for _, b := range []*GearBrand{&brandA, &brandB, &disabled} {
		if err := m.SaveBrand(b); err != nil {
			t.Fatalf("save brand %s: %v", b.Name, err)
		}
	}
	products := []GearProduct{
		{Category: GearCategoryCue, BrandId: brandA.Id, Name: "A-100", CueStyle: GearCueStylePool, LengthMm: 1470, WeightG: 540, TipDiameterMm: 12.9, Sort: 1, Status: 1},
		{Category: GearCategoryCue, BrandId: brandA.Id, Name: "A-200", CueStyle: GearCueStyleSnooker, Sort: 2, Status: 1},
		{Category: GearCategoryCue, BrandId: brandA.Id, Name: "A-OFF", CueStyle: GearCueStylePool, Sort: 3, Status: 0},
		{Category: GearCategoryTip, BrandId: brandB.Id, Name: "B_tip%1", Sort: 1, Status: 1},
		{Category: GearCategoryCue, BrandId: disabled.Id, Name: "G-1", CueStyle: GearCueStylePool, Sort: 1, Status: 1},
	}
	for i := range products {
		if err := m.SaveProduct(&products[i]); err != nil {
			t.Fatalf("save product %s: %v", products[i].Name, err)
		}
	}
	return brandA, brandB
}

func TestGearModelListBrandsWithProductsOnlyReturnsUsableBrands(t *testing.T) {
	db := newGearTestDB(t)
	brandA, brandB := seedGearCatalog(t, db)
	m := NewGearModel(db)

	cueBrands, err := m.ListBrandsWithProducts(GearCategoryCue)
	if err != nil {
		t.Fatalf("list cue brands: %v", err)
	}
	if len(cueBrands) != 1 || cueBrands[0].Id != brandA.Id {
		t.Fatalf("expected only the enabled brand with enabled cue products, got %+v", cueBrands)
	}
	tipBrands, err := m.ListBrandsWithProducts(GearCategoryTip)
	if err != nil {
		t.Fatalf("list tip brands: %v", err)
	}
	if len(tipBrands) != 1 || tipBrands[0].Id != brandB.Id {
		t.Fatalf("expected the tip brand, got %+v", tipBrands)
	}
	chalkBrands, err := m.ListBrandsWithProducts(GearCategoryChalk)
	if err != nil || len(chalkBrands) != 0 {
		t.Fatalf("house chalk has no brand and must not create a brand entry, got %+v err=%v", chalkBrands, err)
	}
}

func TestGearModelListProductsPageFiltersAndPagesStably(t *testing.T) {
	db := newGearTestDB(t)
	brandA, _ := seedGearCatalog(t, db)
	m := NewGearModel(db)

	products, total, err := m.ListProductsPage(GearProductFilter{Category: GearCategoryCue, Status: 1, OnlyEnabledBrand: true, Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("list products: %v", err)
	}
	// 启用的 A-100、A-200 与球房公杆；停用型号和停用品牌下的型号都不出现。
	if total != 3 || len(products) != 2 {
		t.Fatalf("expected total 3 with a page of 2, got total=%d len=%d", total, len(products))
	}
	page2, _, err := m.ListProductsPage(GearProductFilter{Category: GearCategoryCue, Status: 1, OnlyEnabledBrand: true, Page: 2, PageSize: 2})
	if err != nil || len(page2) != 1 {
		t.Fatalf("expected one product on page 2, got %+v err=%v", page2, err)
	}
	seen := map[int64]bool{}
	for _, p := range append(products, page2...) {
		if seen[p.Id] {
			t.Fatalf("product %d repeated across pages", p.Id)
		}
		seen[p.Id] = true
		if p.Status != 1 || p.Name == "A-OFF" || p.Name == "G-1" {
			t.Fatalf("unexpected product in user-facing list: %+v", p)
		}
	}

	byAlias, _, err := m.ListProductsPage(GearProductFilter{Category: GearCategoryCue, Keyword: "阿尔法", Status: 1, OnlyEnabledBrand: true, Page: 1, PageSize: 10})
	if err != nil || len(byAlias) != 2 {
		t.Fatalf("expected the brand alias to match both enabled Alpha cues, got %+v err=%v", byAlias, err)
	}
	for _, p := range byAlias {
		if p.BrandId != brandA.Id {
			t.Fatalf("alias search returned a product of another brand: %+v", p)
		}
	}

	house, total, err := m.ListProductsPage(GearProductFilter{Category: GearCategoryCue, HouseOnly: true, Status: 1, Page: 1, PageSize: 10})
	if err != nil || total != 1 || !house[0].House || house[0].BrandId != 0 {
		t.Fatalf("expected the preset house cue, got %+v total=%d err=%v", house, total, err)
	}

	// LIKE 通配符必须按字面量匹配
	literal, _, err := m.ListProductsPage(GearProductFilter{Category: GearCategoryTip, Keyword: "tip%1", Status: 1, Page: 1, PageSize: 10})
	if err != nil || len(literal) != 1 {
		t.Fatalf("a percent sign in the keyword must match literally, got %+v err=%v", literal, err)
	}
	none, _, err := m.ListProductsPage(GearProductFilter{Category: GearCategoryTip, Keyword: "%", Status: 1, Page: 1, PageSize: 10})
	if err != nil || len(none) != 1 {
		t.Fatalf("a lone percent sign must only match names containing a percent sign, got %+v err=%v", none, err)
	}
	empty, _, err := m.ListProductsPage(GearProductFilter{Category: GearCategoryTip, Keyword: "_", Status: 1, Page: 1, PageSize: 10})
	if err != nil || len(empty) != 1 {
		t.Fatalf("underscore in the keyword must match literally, got %+v err=%v", empty, err)
	}
}

func TestGearModelSaveRejectsDuplicates(t *testing.T) {
	db := newGearTestDB(t)
	brandA, _ := seedGearCatalog(t, db)
	m := NewGearModel(db)

	if err := m.SaveBrand(&GearBrand{Name: "Alpha", Status: 1}); err != ErrGearDuplicate {
		t.Fatalf("expected duplicate brand error, got %v", err)
	}
	if err := m.SaveProduct(&GearProduct{Category: GearCategoryCue, BrandId: brandA.Id, Name: "A-100", CueStyle: GearCueStylePool, Status: 1}); err != ErrGearDuplicate {
		t.Fatalf("expected duplicate product error, got %v", err)
	}
	// 同名但类别不同的型号允许存在
	if err := m.SaveProduct(&GearProduct{Category: GearCategoryTip, BrandId: brandA.Id, Name: "A-100", Status: 1}); err != nil {
		t.Fatalf("same name in another category must be allowed: %v", err)
	}
}

func TestGearModelPendingGroupsAndLink(t *testing.T) {
	db := newGearTestDB(t)
	m := NewGearModel(db)

	addItem := func(userId int64, brandName, modelName string, status int) UserGearItem {
		item := UserGearItem{
			UserId: userId, Category: GearCategoryCue, CustomBrandName: brandName, CustomModelName: modelName,
			CustomKey: BuildGearCustomKey(GearCategoryCue, 0, brandName, modelName), CueStyle: GearCueStylePool, Status: status,
		}
		if err := m.CreateItemWithTx(nil, &item); err != nil {
			t.Fatalf("create item: %v", err)
		}
		return item
	}
	addItem(1, "Brand X", "Brand X  Pro 1", GearItemStatusActive)
	second := addItem(2, "brand x", "brand x pro 1", GearItemStatusRetired)
	addItem(3, "Brand X", "Pro 1", GearItemStatusActive) // 不同型号名
	deleted := addItem(4, "Brand X", "Brand X Pro 1", GearItemStatusDeleted)
	addItem(5, "Solo", "Solo 9", GearItemStatusActive)

	groups, total, err := m.ListPendingGroups("", 1, 10)
	if err != nil {
		t.Fatalf("list pending groups: %v", err)
	}
	if total != 3 || len(groups) != 3 {
		t.Fatalf("expected three groups, got total=%d groups=%+v", total, groups)
	}
	if groups[0].CustomKey != second.CustomKey || groups[0].Count != 2 {
		t.Fatalf("the two spellings must merge into the biggest group, deleted items excluded: %+v", groups[0])
	}

	paged, total, err := m.ListPendingGroups("", 2, 2)
	if err != nil || total != 3 || len(paged) != 1 {
		t.Fatalf("second page of two should hold the last group, got %+v total=%d err=%v", paged, total, err)
	}

	first, err := m.FindFirstByCustomKey(second.CustomKey)
	if err != nil || first == nil || first.CustomKey != second.CustomKey {
		t.Fatalf("find first by key: %+v err=%v", first, err)
	}

	updated, remaining, err := m.LinkPending(second.CustomKey, 99, 1)
	if err != nil || updated != 1 || remaining != 2 {
		t.Fatalf("batch of one should leave two behind, got updated=%d remaining=%d err=%v", updated, remaining, err)
	}
	updated, remaining, err = m.LinkPending(second.CustomKey, 99, GearPendingLinkBatchSize)
	if err != nil || updated != 2 || remaining != 0 {
		t.Fatalf("second round should link the rest, got updated=%d remaining=%d err=%v", updated, remaining, err)
	}

	var linkedDeleted UserGearItem
	if err := db.First(&linkedDeleted, deleted.Id).Error; err != nil || linkedDeleted.ProductId != 99 {
		t.Fatalf("deleted items must be linked too so history shows the real model: %+v err=%v", linkedDeleted, err)
	}
	if linkedDeleted.CustomKey != second.CustomKey || linkedDeleted.CustomModelName != "Brand X Pro 1" {
		t.Fatalf("linking must keep the user's own fields and key: %+v", linkedDeleted)
	}
	groups, total, err = m.ListPendingGroups("", 1, 10)
	if err != nil || total != 2 {
		t.Fatalf("linked group must leave the pending list, got total=%d groups=%+v err=%v", total, groups, err)
	}
}

func TestGearModelDefaultsAndSnapshotsAreIdempotent(t *testing.T) {
	db := newGearTestDB(t)
	m := NewGearModel(db)

	if err := m.UpsertDefaultWithTx(nil, &UserGearDefault{UserId: 1, GameType: 3, Slot: GearSlotCue, ItemId: 10}); err != nil {
		t.Fatalf("set default: %v", err)
	}
	if err := m.UpsertDefaultWithTx(nil, &UserGearDefault{UserId: 1, GameType: 3, Slot: GearSlotCue, ItemId: 11}); err != nil {
		t.Fatalf("replace default: %v", err)
	}
	defaults, err := m.ListDefaults(1)
	if err != nil || len(defaults) != 1 || defaults[0].ItemId != 11 {
		t.Fatalf("expected a single replaced default, got %+v err=%v", defaults, err)
	}
	if err := m.DeleteDefaultsByItemWithTx(nil, 1, 11); err != nil {
		t.Fatalf("delete defaults by item: %v", err)
	}
	if defaults, _ := m.ListDefaults(1); len(defaults) != 0 {
		t.Fatalf("defaults must be cleared with the item, got %+v", defaults)
	}

	auto := MatchGearSnapshot{MatchId: 7, UserId: 1, CueItemId: 10, TipProductId: 3, Source: GearSnapshotSourceAuto}
	if err := m.InsertSnapshotsIgnore([]MatchGearSnapshot{auto}); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}
	overwrite := MatchGearSnapshot{MatchId: 7, UserId: 1, CueItemId: 99, Source: GearSnapshotSourceAuto}
	if err := m.InsertSnapshotsIgnore([]MatchGearSnapshot{overwrite}); err != nil {
		t.Fatalf("repeat insert must not fail: %v", err)
	}
	rows, err := m.FindSnapshotsByMatch(7)
	if err != nil || len(rows) != 1 || rows[0].CueItemId != 10 || rows[0].TipProductId != 3 {
		t.Fatalf("auto insert must not overwrite an existing row, got %+v err=%v", rows, err)
	}

	manual := MatchGearSnapshot{MatchId: 7, UserId: 1, CueItemId: 12, ChalkProductId: 5}
	if err := m.UpsertManualSnapshot(&manual); err != nil {
		t.Fatalf("manual upsert: %v", err)
	}
	rows, _ = m.FindSnapshotsByMatch(7)
	if len(rows) != 1 || rows[0].CueItemId != 12 || rows[0].TipProductId != 0 || rows[0].ChalkProductId != 5 || rows[0].Source != GearSnapshotSourceManual {
		t.Fatalf("manual upsert must replace the combination and mark the source, got %+v", rows)
	}
	if err := m.InsertSnapshotsIgnore([]MatchGearSnapshot{auto}); err != nil {
		t.Fatalf("late auto write must not fail: %v", err)
	}
	rows, _ = m.FindSnapshotsByMatch(7)
	if rows[0].Source != GearSnapshotSourceManual || rows[0].CueItemId != 12 {
		t.Fatalf("a late automatic write must not overwrite a manual record, got %+v", rows)
	}
}

func TestGearModelListUserItemsIsBoundedAndOrdered(t *testing.T) {
	db := newGearTestDB(t)
	m := NewGearModel(db)
	for i := range GearMaxItemsPerUser*2 + 5 {
		item := UserGearItem{UserId: 1, Category: GearCategoryCue, CustomModelName: fmt.Sprintf("cue %d", i), Status: GearItemStatusActive}
		if err := m.CreateItemWithTx(nil, &item); err != nil {
			t.Fatalf("create item %d: %v", i, err)
		}
	}
	items, err := m.ListUserItems(1, []int{GearItemStatusActive, GearItemStatusRetired})
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != GearMaxItemsPerUser*2 {
		t.Fatalf("list must be capped as a safety net, got %d", len(items))
	}
	for i := 1; i < len(items); i++ {
		if items[i-1].Id >= items[i].Id {
			t.Fatal("items must be ordered by id ascending")
		}
	}
}
