package gear

import (
	"strconv"
	"strings"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestGetGearMineUsesConstantQueries(t *testing.T) {
	count := func(items int) int {
		env := newGearEnv(t, strconv.Itoa(items))
		m := model.NewGearModel(env.db)
		for i := 0; i < items; i++ {
			item := model.UserGearItem{
				UserId: 1, Category: model.GearCategoryCue, ProductId: env.c.cuePool,
				TipProductId: env.c.tipAlpha, ChalkProductId: env.c.chalkAlpha, Status: model.GearItemStatusActive,
			}
			if i%3 == 1 {
				item = model.UserGearItem{UserId: 1, Category: model.GearCategoryCue, CustomBrandId: env.c.brandAlpha, CustomModelName: "x", Status: model.GearItemStatusActive}
			}
			if err := m.CreateItemWithTx(nil, &item); err != nil {
				t.Fatalf("seed item: %v", err)
			}
		}
		if err := m.UpsertDefaultWithTx(nil, &model.UserGearDefault{UserId: 1, GameType: 3, Slot: model.GearSlotCue, ItemId: 1}); err != nil {
			t.Fatalf("seed default: %v", err)
		}
		ctx, svcCtx, metrics := env.request(1)
		resp, err := NewGetGearMineLogic(ctx, svcCtx).GetGearMine()
		if err != nil || !resp.Success || len(resp.Items) != items {
			t.Fatalf("get mine: items=%d resp=%+v err=%v", items, resp, err)
		}
		return int(metrics.Snapshot().SQLCount)
	}
	one, thirty := count(1), count(30)
	if one != thirty || one != 4 {
		t.Fatalf("mine must use items, products, brands and defaults regardless of count: one=%d thirty=%d", one, thirty)
	}
}

func TestGetGearMineReturnsOwnItemsOnlyWithSpecsAndDefaults(t *testing.T) {
	env := newGearEnv(t, "")
	cueId := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool, TipProductId: env.c.tipAlpha, ChalkProductId: env.c.chalkAlpha})
	env.mustCreate(2, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cueSnooker})

	resp := env.mine(1)
	if !resp.Success || resp.MaxItems != model.GearMaxItemsPerUser || len(resp.Items) != 1 {
		t.Fatalf("mine should only list the caller's items: %+v", resp)
	}
	item := resp.Items[0]
	if item.Id != cueId || item.Product == nil || item.Product.WeightG != 540 || item.TipProduct == nil || item.ChalkProduct == nil {
		t.Fatalf("mine should carry official specs, tip and chalk: %+v", item)
	}
	if resp.Defaults == nil || resp.Items == nil {
		t.Fatal("empty collections must be non-nil so clients never receive null")
	}
}

func TestCreateGearItemFromCatalogAndUnlisted(t *testing.T) {
	env := newGearEnv(t, "")

	catalog := env.createItem(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	if !catalog.Success || catalog.Item.Unlisted || catalog.Item.Product == nil || catalog.Item.Status != model.GearItemStatusActive {
		t.Fatalf("catalog cue should be registered as an active listed item: %+v", catalog)
	}

	house := env.createItem(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.houseCue})
	if !house.Success || !house.Item.House || house.Item.ModelName != "球房公杆" || house.Item.BrandName != "" {
		t.Fatalf("house cue should register as the brandless house entry: %+v", house)
	}

	own := env.createItem(1, types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "  自家小厂 ", ModelName: " 手工杆 1 ", CueStyle: model.GearCueStyleSnooker})
	if !own.Success || !own.Item.Unlisted || own.Item.BrandName != "自家小厂" || own.Item.ModelName != "手工杆 1" || own.Item.CueStyle != model.GearCueStyleSnooker {
		t.Fatalf("unlisted cue with a typed brand: %+v", own)
	}
	var stored model.UserGearItem
	if err := env.db.First(&stored, own.Item.Id).Error; err != nil {
		t.Fatalf("load stored item: %v", err)
	}
	if stored.CustomKey != "cue|n自家小厂|手工杆 1" || stored.ProductId != 0 {
		t.Fatalf("unlisted registration must store the merge key and no product: %+v", stored)
	}

	withBrand := env.createItem(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, BrandId: env.c.brandAlpha, ModelName: "Alpha 新冲杆", Roles: []string{"jump", "break"}})
	if !withBrand.Success || withBrand.Item.BrandName != "Alpha" || len(withBrand.Item.Roles) != 2 || withBrand.Item.Roles[0] != "break" {
		t.Fatalf("unlisted break/jump with a catalog brand: %+v", withBrand)
	}

	// 两位用户登记同一型号的不同写法，归并键相同
	a := env.createItem(2, types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "Brand X", ModelName: "Brand X  Pro 1", CueStyle: model.GearCueStylePool})
	b := env.createItem(3, types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "brand x", ModelName: "brand x pro 1", CueStyle: model.GearCueStylePool})
	var keyA, keyB model.UserGearItem
	env.db.First(&keyA, a.Item.Id)
	env.db.First(&keyB, b.Item.Id)
	if keyA.CustomKey == "" || keyA.CustomKey != keyB.CustomKey {
		t.Fatalf("the two spellings must merge into one pending group: %q vs %q", keyA.CustomKey, keyB.CustomKey)
	}
}

func TestCreateGearItemRejectsInvalidRequests(t *testing.T) {
	env := newGearEnv(t, "")
	longName := strings.Repeat("长", 65)
	longBrand := strings.Repeat("品", 33)
	cases := []struct {
		name string
		req  types.CreateGearItemReq
	}{
		{"category", types.CreateGearItemReq{Category: "tip", ProductId: env.c.tipAlpha}},
		{"unknown product", types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: 99999}},
		{"product category mismatch", types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.breakCue}},
		{"disabled product", types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cueOff}},
		{"product of disabled brand", types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cueOfOffBrand}},
		{"missing model name", types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "品牌", CueStyle: model.GearCueStylePool}},
		{"model name too long", types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "品牌", ModelName: longName, CueStyle: model.GearCueStylePool}},
		{"missing brand", types.CreateGearItemReq{Category: model.GearCategoryCue, ModelName: "型号", CueStyle: model.GearCueStylePool}},
		{"brand too long", types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: longBrand, ModelName: "型号", CueStyle: model.GearCueStylePool}},
		{"disabled brand", types.CreateGearItemReq{Category: model.GearCategoryCue, BrandId: env.c.brandOff, ModelName: "型号", CueStyle: model.GearCueStylePool}},
		{"missing cue style", types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "品牌", ModelName: "型号"}},
		{"bad cue style", types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "品牌", ModelName: "型号", CueStyle: "carom"}},
		{"missing roles", types.CreateGearItemReq{Category: model.GearCategoryBreakJump, BrandName: "品牌", ModelName: "型号"}},
		{"bad roles", types.CreateGearItemReq{Category: model.GearCategoryBreakJump, BrandName: "品牌", ModelName: "型号", Roles: []string{"fly"}}},
		{"tip on break cue", types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.breakCue, TipProductId: env.c.tipAlpha}},
		{"chalk on jump cue", types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.jumpCue, ChalkProductId: env.c.chalkAlpha}},
		{"disabled tip", types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool, TipProductId: env.c.tipOff}},
		{"tip is not a tip", types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool, TipProductId: env.c.chalkAlpha}},
		{"chalk is not chalk", types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool, ChalkProductId: env.c.tipAlpha}},
	}
	for _, tc := range cases {
		resp := env.createItem(1, tc.req)
		if resp.Success || resp.Message == "" {
			t.Fatalf("%s: expected a rejection with a message, got %+v", tc.name, resp)
		}
	}
	var count int64
	env.db.Model(&model.UserGearItem{}).Count(&count)
	if count != 0 {
		t.Fatalf("rejected requests must not write anything, found %d rows", count)
	}
}

func TestCreateGearItemEnforcesItemLimit(t *testing.T) {
	env := newGearEnv(t, "")
	for i := 0; i < model.GearMaxItemsPerUser; i++ {
		env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	}
	over := env.createItem(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	if over.Success || !strings.Contains(over.Message, "30") {
		t.Fatalf("the 31st item must be rejected with the limit in the message, got %+v", over)
	}
	// 上限按用户计算，别人不受影响
	if resp := env.createItem(2, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool}); !resp.Success {
		t.Fatalf("another user is unaffected by the limit: %+v", resp)
	}

	// 删除后释放名额
	mine := env.mine(1)
	ctx, svcCtx, _ := env.request(1)
	if del, _ := NewDeleteGearItemLogic(ctx, svcCtx).DeleteGearItem(&types.DeleteGearItemReq{ItemId: mine.Items[0].Id}); !del.Success {
		t.Fatalf("delete: %+v", del)
	}
	if again := env.createItem(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool}); !again.Success {
		t.Fatalf("deleting an item must free a slot: %+v", again)
	}
}

func TestUpdateGearItemChangesTipChalkAndStatus(t *testing.T) {
	env := newGearEnv(t, "")
	cueId := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	update := func(userId int64, req types.UpdateGearItemReq) *types.GearItemResp {
		ctx, svcCtx, _ := env.request(userId)
		resp, err := NewUpdateGearItemLogic(ctx, svcCtx).UpdateGearItem(&req)
		if err != nil {
			t.Fatalf("update returned an error: %v", err)
		}
		return resp
	}

	resp := update(1, types.UpdateGearItemReq{ItemId: cueId, UpdateTip: true, TipProductId: env.c.tipAlpha, UpdateChalk: true, ChalkProductId: env.c.chalkAlpha})
	if !resp.Success || resp.Item.TipProduct == nil || resp.Item.ChalkProduct == nil {
		t.Fatalf("set tip and chalk: %+v", resp)
	}
	// 只改巧克时，皮头保持不变
	resp = update(1, types.UpdateGearItemReq{ItemId: cueId, UpdateChalk: true, ChalkProductId: 0})
	if !resp.Success || resp.Item.TipProduct == nil || resp.Item.ChalkProduct != nil {
		t.Fatalf("clearing chalk must leave the tip alone: %+v", resp)
	}
	// 不带更新标志时子属性保持不变
	resp = update(1, types.UpdateGearItemReq{ItemId: cueId, Status: model.GearItemStatusRetired})
	if !resp.Success || resp.Item.Status != model.GearItemStatusRetired || resp.Item.TipProduct == nil {
		t.Fatalf("status-only update keeps tip: %+v", resp)
	}
	resp = update(1, types.UpdateGearItemReq{ItemId: cueId, Status: model.GearItemStatusActive})
	if !resp.Success || resp.Item.Status != model.GearItemStatusActive {
		t.Fatalf("restore from retired: %+v", resp)
	}

	for name, req := range map[string]types.UpdateGearItemReq{
		"bad status":     {ItemId: cueId, Status: 3},
		"disabled tip":   {ItemId: cueId, UpdateTip: true, TipProductId: env.c.tipOff},
		"wrong category": {ItemId: cueId, UpdateChalk: true, ChalkProductId: env.c.tipAlpha},
		"missing item":   {ItemId: 99999, Status: 2},
	} {
		if resp := update(1, req); resp.Success {
			t.Fatalf("%s should be rejected: %+v", name, resp)
		}
	}
	if resp := update(2, types.UpdateGearItemReq{ItemId: cueId, Status: model.GearItemStatusRetired}); resp.Success {
		t.Fatalf("another user must not modify the item: %+v", resp)
	}

	breakId := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.breakCue})
	if resp := update(1, types.UpdateGearItemReq{ItemId: breakId, UpdateTip: true, TipProductId: env.c.tipAlpha}); resp.Success {
		t.Fatalf("a break cue has no tip: %+v", resp)
	}
}

func TestRetireAndDeleteClearDefaultsAndHideItem(t *testing.T) {
	env := newGearEnv(t, "")
	cueId := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	breakId := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.breakCue})
	setDefault := func(gameType int, slot string, itemId int64) *types.CommonResp {
		ctx, svcCtx, _ := env.request(1)
		resp, _ := NewSetGearDefaultLogic(ctx, svcCtx).SetGearDefault(&types.SetGearDefaultReq{GameType: gameType, Slot: slot, ItemId: itemId})
		return resp
	}
	for _, gameType := range []int{1, 3} {
		if resp := setDefault(gameType, model.GearSlotCue, cueId); !resp.Success {
			t.Fatalf("set cue default for %d: %+v", gameType, resp)
		}
	}
	if resp := setDefault(3, model.GearSlotBreak, breakId); !resp.Success {
		t.Fatalf("set break default: %+v", resp)
	}

	ctx, svcCtx, _ := env.request(1)
	if resp, _ := NewUpdateGearItemLogic(ctx, svcCtx).UpdateGearItem(&types.UpdateGearItemReq{ItemId: cueId, Status: model.GearItemStatusRetired}); !resp.Success {
		t.Fatalf("retire: %+v", resp)
	}
	after := env.mine(1)
	if len(after.Defaults) != 1 || after.Defaults[0].Slot != model.GearSlotBreak {
		t.Fatalf("retiring the main cue must clear its defaults only: %+v", after.Defaults)
	}
	if resp := setDefault(3, model.GearSlotCue, cueId); resp.Success {
		t.Fatalf("a retired cue cannot become the main cue: %+v", resp)
	}

	ctx, svcCtx, _ = env.request(1)
	if resp, _ := NewDeleteGearItemLogic(ctx, svcCtx).DeleteGearItem(&types.DeleteGearItemReq{ItemId: breakId}); !resp.Success {
		t.Fatalf("delete: %+v", resp)
	}
	after = env.mine(1)
	if len(after.Defaults) != 0 || findItem(after.Items, breakId) != nil {
		t.Fatalf("a deleted item vanishes from mine and loses its defaults: %+v", after)
	}
	// 软删除：记录仍在，历史对局能解析
	ids, err := model.NewGearModel(env.db).FindItemsByIds([]int64{breakId})
	if err != nil || ids[breakId].Status != model.GearItemStatusDeleted {
		t.Fatalf("deleted item must stay resolvable by id: %+v err=%v", ids, err)
	}
	ctx, svcCtx, _ = env.request(1)
	if again, _ := NewDeleteGearItemLogic(ctx, svcCtx).DeleteGearItem(&types.DeleteGearItemReq{ItemId: breakId}); again.Success {
		t.Fatalf("deleting twice should report the item as gone: %+v", again)
	}
	ctx, svcCtx, _ = env.request(2)
	if other, _ := NewDeleteGearItemLogic(ctx, svcCtx).DeleteGearItem(&types.DeleteGearItemReq{ItemId: cueId}); other.Success {
		t.Fatalf("another user must not delete the item: %+v", other)
	}
}

func TestSetGearDefaultValidatesSlotAndItem(t *testing.T) {
	env := newGearEnv(t, "")
	cue := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	snookerCue := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cueSnooker})
	breakCue := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.breakCue})
	jumpOnly := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.jumpCue})
	both := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.breakJumpBoth})
	unlistedBreak := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, BrandName: "x", ModelName: "y", Roles: []string{"break"}})
	unlistedJump := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, BrandName: "x", ModelName: "z", Roles: []string{"jump"}})
	other := env.mustCreate(2, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})

	set := func(gameType int, slot string, itemId int64) *types.CommonResp {
		ctx, svcCtx, _ := env.request(1)
		resp, err := NewSetGearDefaultLogic(ctx, svcCtx).SetGearDefault(&types.SetGearDefaultReq{GameType: gameType, Slot: slot, ItemId: itemId})
		if err != nil {
			t.Fatalf("set default returned an error: %v", err)
		}
		return resp
	}
	mustOK := map[string]*types.CommonResp{
		"cue for snooker":          set(1, model.GearSlotCue, cue),
		"replace cue for snooker":  set(1, model.GearSlotCue, snookerCue),
		"break for nine":           set(2, model.GearSlotBreak, breakCue),
		"dual use for pool":        set(4, model.GearSlotBreak, both),
		"unlisted break for eight": set(3, model.GearSlotBreak, unlistedBreak),
	}
	for name, resp := range mustOK {
		if !resp.Success {
			t.Fatalf("%s should succeed: %+v", name, resp)
		}
	}
	mustFail := map[string]*types.CommonResp{
		"bad game type":              set(5, model.GearSlotCue, cue),
		"bad slot":                   set(3, "tip", cue),
		"snooker has no break":       set(1, model.GearSlotBreak, breakCue),
		"jump-only as break":         set(3, model.GearSlotBreak, jumpOnly),
		"unlisted jump-only":         set(3, model.GearSlotBreak, unlistedJump),
		"break cue as main cue":      set(3, model.GearSlotCue, breakCue),
		"cue as break":               set(3, model.GearSlotBreak, cue),
		"someone else's item":        set(3, model.GearSlotCue, other),
		"missing item":               set(3, model.GearSlotCue, 99999),
		"clearing snooker break too": set(1, model.GearSlotBreak, 0),
	}
	for name, resp := range mustFail {
		if resp.Success {
			t.Fatalf("%s should be rejected: %+v", name, resp)
		}
	}

	defaults := env.mine(1).Defaults
	if len(defaults) != 4 {
		t.Fatalf("expected four defaults after the accepted calls, got %+v", defaults)
	}
	for _, d := range defaults {
		if d.GameType == 1 && d.Slot == model.GearSlotCue && d.ItemId != snookerCue {
			t.Fatalf("setting again must replace the earlier main cue: %+v", d)
		}
	}
	if resp := set(1, model.GearSlotCue, 0); !resp.Success {
		t.Fatalf("clearing a default should succeed: %+v", resp)
	}
	if resp := set(3, model.GearSlotCue, 0); !resp.Success {
		t.Fatalf("clearing an unset default is a no-op success: %+v", resp)
	}
	for _, d := range env.mine(1).Defaults {
		if d.GameType == 1 && d.Slot == model.GearSlotCue {
			t.Fatalf("the cleared default is still present: %+v", d)
		}
	}
}
