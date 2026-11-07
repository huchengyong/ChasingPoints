package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
	adminlogic "chasing_points/internal/logic/admin"
	gearlogic "chasing_points/internal/logic/gear"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 装备功能在真实 MySQL 上的端到端验收：SQLite 单测覆盖逻辑，这里专门验证迁移与 MySQL 方言
// （ON DUPLICATE KEY、LIKE ... ESCAPE、行锁事务、分组计数子查询、批量更新）。
func TestMySQLGearEndToEnd(t *testing.T) {
	if os.Getenv("RUN_MYSQL_GEAR_TESTS") != "1" {
		t.Skip("set RUN_MYSQL_GEAR_TESTS=1 to run the gear MySQL acceptance")
	}
	admin, err := sql.Open("mysql", mysqlTestDSN(t, ""))
	if err != nil {
		t.Fatalf("open MySQL admin connection: %v", err)
	}
	defer admin.Close()
	databaseName := fmt.Sprintf("chasing_points_gear_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + databaseName + "`"); err != nil {
		t.Fatalf("create gear database: %v", err)
	}
	defer func() { _, _ = admin.Exec("DROP DATABASE IF EXISTS `" + databaseName + "`") }()
	dsn := mysqlTestDSN(t, databaseName)
	if output, err := exec.Command("goose", "-dir", ".", "mysql", dsn, "up").CombinedOutput(); err != nil {
		t.Fatalf("apply migrations: %v: %s", err, output)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open Gorm database: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:                     db,
		GearModel:              model.NewGearModel(db),
		UserModel:              model.NewUserModel(db),
		FriendModel:            model.NewFriendModel(db),
		MatchModel:             model.NewMatchModel(db),
		UserDataLifecycleModel: model.NewUserDataLifecycleModel(db),
	}
	userCtx := func(id int64) context.Context { return context.WithValue(context.Background(), "user_id", id) }
	adminCtx := userCtx(-1)

	// 迁移预置的球房公杆与球房巧克
	var house int64
	if err := db.Model(&model.GearProduct{}).Where("house = ?", true).Count(&house).Error; err != nil || house != 2 {
		t.Fatalf("the migration must preset the two house entries: count=%d err=%v", house, err)
	}
	var houseCue model.GearProduct
	if err := db.Where("category = ? AND house = ?", model.GearCategoryCue, true).First(&houseCue).Error; err != nil {
		t.Fatalf("preset house cue: %v", err)
	}

	for id := int64(1); id <= 3; id++ {
		if err := svcCtx.UserModel.Create(&model.User{Id: id, Nickname: fmt.Sprintf("球友%d", id), Status: 1}); err != nil {
			t.Fatalf("seed user %d: %v", id, err)
		}
	}
	if err := db.Create(&model.Friend{UserId: 1, FriendId: 2, Status: 1}).Error; err != nil {
		t.Fatalf("seed friendship: %v", err)
	}

	// ---- 后台维护目录 ----
	saveBrand := func(req types.AdminGearBrandSaveReq) int64 {
		resp, _ := adminlogic.NewAdminSaveGearBrandLogic(adminCtx, svcCtx).AdminSaveGearBrand(&req)
		if !resp.Success {
			t.Fatalf("save brand %+v: %+v", req, resp)
		}
		return resp.Id
	}
	saveProduct := func(req types.AdminGearProductSaveReq) int64 {
		resp, _ := adminlogic.NewAdminSaveGearProductLogic(adminCtx, svcCtx).AdminSaveGearProduct(&req)
		if !resp.Success {
			t.Fatalf("save product %+v: %+v", req, resp)
		}
		return resp.Id
	}
	alpha := saveBrand(types.AdminGearBrandSaveReq{Name: "Alpha", Aliases: "阿尔法,alp", Sort: 1, Status: 1})
	if dup, _ := adminlogic.NewAdminSaveGearBrandLogic(adminCtx, svcCtx).AdminSaveGearBrand(&types.AdminGearBrandSaveReq{Name: "Alpha", Status: 1}); dup.Success || dup.Code != 409 {
		t.Fatalf("a duplicate brand must be refused: %+v", dup)
	}
	cuePool := saveProduct(types.AdminGearProductSaveReq{Category: model.GearCategoryCue, BrandId: alpha, Name: "A-100 100%", CueStyle: model.GearCueStylePool, LengthMm: 1470, WeightG: 540, TipDiameterMm: 12.9, Status: 1})
	cueSnooker := saveProduct(types.AdminGearProductSaveReq{Category: model.GearCategoryCue, BrandId: alpha, Name: "A-200", CueStyle: model.GearCueStyleSnooker, Status: 1})
	breakCue := saveProduct(types.AdminGearProductSaveReq{Category: model.GearCategoryBreakJump, BrandId: alpha, Name: "A-Break", Roles: []string{"break"}, Status: 1})
	tipProduct := saveProduct(types.AdminGearProductSaveReq{Category: model.GearCategoryTip, BrandId: alpha, Name: "A-Tip", Status: 1})
	chalkProduct := saveProduct(types.AdminGearProductSaveReq{Category: model.GearCategoryChalk, BrandId: alpha, Name: "A-Chalk", Status: 1})

	// ---- 用户端目录读取 ----
	brands, _ := gearlogic.NewGetGearBrandsLogic(userCtx(1), svcCtx).GetGearBrands(&types.GetGearBrandsReq{Category: model.GearCategoryCue})
	if !brands.Success || len(brands.List) != 1 || brands.List[0].Id != alpha {
		t.Fatalf("brand list: %+v", brands)
	}
	byAlias, _ := gearlogic.NewGetGearProductsLogic(userCtx(1), svcCtx).GetGearProducts(&types.GetGearProductsReq{Category: model.GearCategoryCue, Keyword: "阿尔法"})
	if !byAlias.Success || byAlias.Total != 2 {
		t.Fatalf("alias search on MySQL: %+v", byAlias)
	}
	literal, _ := gearlogic.NewGetGearProductsLogic(userCtx(1), svcCtx).GetGearProducts(&types.GetGearProductsReq{Category: model.GearCategoryCue, Keyword: "100%"})
	if !literal.Success || literal.Total != 1 || literal.List[0].Id != cuePool {
		t.Fatalf("a percent sign must match literally on MySQL: %+v", literal)
	}
	wildcard, _ := gearlogic.NewGetGearProductsLogic(userCtx(1), svcCtx).GetGearProducts(&types.GetGearProductsReq{Category: model.GearCategoryCue, Keyword: "%"})
	if !wildcard.Success || wildcard.Total != 1 {
		t.Fatalf("a lone percent sign must not match everything: %+v", wildcard)
	}

	// ---- 登记、数量上限（行锁事务）、主力 ----
	create := func(userId int64, req types.CreateGearItemReq) *types.GearItemResp {
		resp, _ := gearlogic.NewCreateGearItemLogic(userCtx(userId), svcCtx).CreateGearItem(&req)
		return resp
	}
	cue := create(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: cuePool, TipProductId: tipProduct, ChalkProductId: chalkProduct})
	brk := create(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: breakCue})
	if !cue.Success || !brk.Success {
		t.Fatalf("register gear: %+v %+v", cue, brk)
	}
	for i := 0; i < model.GearMaxItemsPerUser-2; i++ {
		if resp := create(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: houseCue.Id}); !resp.Success {
			t.Fatalf("filling up to the limit failed at %d: %+v", i, resp)
		}
	}
	if over := create(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: houseCue.Id}); over.Success || !strings.Contains(over.Message, "30") {
		t.Fatalf("the limit must hold on MySQL: %+v", over)
	}
	setDefault := func(userId int64, gameType int, slot string, itemId int64) {
		resp, _ := gearlogic.NewSetGearDefaultLogic(userCtx(userId), svcCtx).SetGearDefault(&types.SetGearDefaultReq{GameType: gameType, Slot: slot, ItemId: itemId})
		if !resp.Success {
			t.Fatalf("set default: %+v", resp)
		}
	}
	setDefault(1, 3, model.GearSlotCue, cue.Item.Id)
	setDefault(1, 3, model.GearSlotBreak, brk.Item.Id)
	setDefault(1, 3, model.GearSlotCue, cue.Item.Id) // 重复设置走 ON DUPLICATE KEY 更新
	mine, _ := gearlogic.NewGetGearMineLogic(userCtx(1), svcCtx).GetGearMine()
	if !mine.Success || len(mine.Items) != model.GearMaxItemsPerUser || len(mine.Defaults) != 2 {
		t.Fatalf("mine: items=%d defaults=%d resp=%+v", len(mine.Items), len(mine.Defaults), mine.Message)
	}

	// 用户 2 登记同一个未收录型号的两种写法
	u1 := create(2, types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "Brand X", ModelName: "Brand X  Pro 1", CueStyle: model.GearCueStylePool})
	u2 := create(2, types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "brand x", ModelName: "brand x pro 1", CueStyle: model.GearCueStylePool})
	if !u1.Success || !u2.Success {
		t.Fatalf("register unlisted: %+v %+v", u1, u2)
	}
	setDefault(2, 3, model.GearSlotCue, u1.Item.Id)

	// ---- 好友视图 ----
	friend, _ := gearlogic.NewGetGearUserLogic(userCtx(1), svcCtx).GetGearUser(&types.GetGearUserReq{UserId: 2})
	if !friend.Success || len(friend.Items) != 2 || friend.Items[0].ModelName != "未收录型号" {
		t.Fatalf("friend view: %+v", friend)
	}
	if stranger, _ := gearlogic.NewGetGearUserLogic(userCtx(3), svcCtx).GetGearUser(&types.GetGearUserReq{UserId: 2}); stranger.Success {
		t.Fatalf("a stranger must be refused: %+v", stranger)
	}

	// ---- 对局完成后记录用杆、读取、修改 ----
	completedAt := time.Now().Add(-time.Hour)
	opponent := int64(2)
	match := &model.Match{
		Id: 7001, UserId: 1, OpponentId: &opponent, OpponentName: "球友2", GameType: 3, MatchMode: model.MatchModePractice,
		Visibility: model.MatchVisibilityPrivate, FinishState: model.FinishStateNone, Status: 2, MatchTime: completedAt.Add(-time.Hour),
		EndTime: &completedAt, CompletedAt: &completedAt,
	}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("create match: %v", err)
	}
	if err := logicx.RecordMatchGearSnapshots(svcCtx, match); err != nil {
		t.Fatalf("record snapshots: %v", err)
	}
	if err := logicx.RecordMatchGearSnapshots(svcCtx, match); err != nil {
		t.Fatalf("a repeated record must be a no-op on MySQL: %v", err)
	}
	var recorded int64
	db.Model(&model.MatchGearSnapshot{}).Where("match_id = ?", match.Id).Count(&recorded)
	if recorded != 2 {
		t.Fatalf("expected one record per player, got %d", recorded)
	}
	view, _ := gearlogic.NewGetGearMatchLogic(userCtx(1), svcCtx).GetGearMatch(&types.GetGearMatchReq{MatchId: match.Id})
	if !view.Success || view.Me.Cue == nil || view.Me.TipProduct == nil || view.Me.BreakCue == nil || view.Opponent.Cue == nil || view.Opponent.Cue.ModelName != "未收录型号" {
		t.Fatalf("match view: %+v", view)
	}
	updated, _ := gearlogic.NewUpdateGearMatchLogic(userCtx(1), svcCtx).UpdateGearMatch(&types.UpdateGearMatchReq{MatchId: match.Id, CueItemId: cue.Item.Id, ChalkProductId: chalkProduct})
	if !updated.Success || updated.Me.Source != model.GearSnapshotSourceManual || updated.Me.TipProduct != nil || updated.Me.ChalkProduct == nil {
		t.Fatalf("manual update on MySQL: %+v", updated)
	}

	// ---- 后台归并关联：双方的历史对局随之显示为正式型号 ----
	pending, _ := adminlogic.NewAdminGetGearPendingListLogic(adminCtx, svcCtx).AdminGetGearPendingList(&types.AdminGearPendingListReq{})
	if !pending.Success || pending.Total != 1 || pending.List[0].Count != 2 {
		t.Fatalf("pending groups on MySQL: %+v", pending)
	}
	linked, _ := adminlogic.NewAdminLinkGearPendingLogic(adminCtx, svcCtx).AdminLinkGearPending(&types.AdminGearPendingLinkReq{CustomKey: pending.List[0].CustomKey, ProductId: cueSnooker})
	if !linked.Success || linked.Updated != 2 || linked.Remaining != 0 {
		t.Fatalf("link pending on MySQL: %+v", linked)
	}
	after, _ := gearlogic.NewGetGearMatchLogic(userCtx(1), svcCtx).GetGearMatch(&types.GetGearMatchReq{MatchId: match.Id})
	if !after.Success || after.Opponent.Cue == nil || after.Opponent.Cue.ModelName != "A-200" || after.Opponent.Cue.Unlisted {
		t.Fatalf("history must show the linked model: %+v", after.Opponent.Cue)
	}

	// ---- 个人数据导出与账号注销覆盖新表 ----
	for category, want := range map[string]int{"gear_items": 2, "gear_defaults": 1, "match_gear_snapshots": 1} {
		records, _, err := svcCtx.UserDataLifecycleModel.ListExportRecords(2, category, 0, time.Now().Add(time.Hour), 100)
		if err != nil || len(records) != want {
			t.Fatalf("export %s on MySQL: want %d got %d err=%v", category, want, len(records), err)
		}
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return svcCtx.UserDataLifecycleModel.DeleteAccountRelationsWithTx(tx, 2, "")
	}); err != nil {
		t.Fatalf("delete account relations on MySQL: %v", err)
	}
	for _, table := range []string{"user_gear_items", "user_gear_defaults", "match_gear_snapshots"} {
		var left int64
		db.Table(table).Where("user_id = ?", 2).Count(&left)
		if left != 0 {
			t.Fatalf("%s must be cleared by the account deletion, %d left", table, left)
		}
	}
	var otherPlayer int64
	db.Model(&model.MatchGearSnapshot{}).Where("match_id = ? AND user_id = ?", match.Id, 1).Count(&otherPlayer)
	if otherPlayer != 1 {
		t.Fatalf("the other player's record must survive, found %d", otherPlayer)
	}
}
