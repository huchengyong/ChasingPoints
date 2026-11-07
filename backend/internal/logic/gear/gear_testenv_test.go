package gear

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// gearEnv 是装备逻辑测试用的 SQLite 环境：users、好友关系与装备五张表。
type gearEnv struct {
	t  *testing.T
	db *gorm.DB
	c  catalogIDs
}

// catalogIDs 是 seedCatalog 预置目录的 ID，球房公杆与球房巧克来自迁移预置。
type catalogIDs struct {
	brandAlpha, brandOff                               int64
	cuePool, cueSnooker, cueOff, cueOfOffBrand         int64
	breakCue, jumpCue, breakJumpBoth                   int64
	tipAlpha, tipOff, chalkAlpha, houseCue, houseChalk int64
}

func newGearEnv(t *testing.T, suffix string) *gearEnv {
	t.Helper()
	name := t.Name()
	if suffix != "" {
		name += "-" + suffix
	}
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		Logger: observability.NewGormLogger(time.Hour),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Friend{}, &model.FriendBlacklist{}, &model.Match{}); err != nil {
		t.Fatalf("prepare user schema: %v", err)
	}
	if err := testsupport.PrepareGearSchema(db); err != nil {
		t.Fatalf("prepare gear schema: %v", err)
	}
	env := &gearEnv{t: t, db: db}
	for id := int64(1); id <= 6; id++ {
		if err := db.Create(&model.User{Id: id, Nickname: fmt.Sprintf("球友%d", id)}).Error; err != nil {
			t.Fatalf("seed user %d: %v", id, err)
		}
	}
	env.seedCatalog()
	return env
}

func (e *gearEnv) seedCatalog() {
	e.t.Helper()
	m := model.NewGearModel(e.db)
	alpha := model.GearBrand{Name: "Alpha", Aliases: "阿尔法", Sort: 1, Status: 1}
	off := model.GearBrand{Name: "OffBrand", Sort: 2, Status: 0}
	for _, b := range []*model.GearBrand{&alpha, &off} {
		if err := m.SaveBrand(b); err != nil {
			e.t.Fatalf("save brand: %v", err)
		}
	}
	save := func(p model.GearProduct) int64 {
		if err := m.SaveProduct(&p); err != nil {
			e.t.Fatalf("save product %s: %v", p.Name, err)
		}
		return p.Id
	}
	e.c = catalogIDs{
		brandAlpha: alpha.Id, brandOff: off.Id,
		cuePool:       save(model.GearProduct{Category: model.GearCategoryCue, BrandId: alpha.Id, Name: "A-100", CueStyle: model.GearCueStylePool, LengthMm: 1470, WeightG: 540, TipDiameterMm: 12.9, Sort: 1, Status: 1}),
		cueSnooker:    save(model.GearProduct{Category: model.GearCategoryCue, BrandId: alpha.Id, Name: "A-200", CueStyle: model.GearCueStyleSnooker, Sort: 2, Status: 1}),
		cueOff:        save(model.GearProduct{Category: model.GearCategoryCue, BrandId: alpha.Id, Name: "A-OFF", CueStyle: model.GearCueStylePool, Sort: 3, Status: 0}),
		cueOfOffBrand: save(model.GearProduct{Category: model.GearCategoryCue, BrandId: off.Id, Name: "G-1", CueStyle: model.GearCueStylePool, Sort: 1, Status: 1}),
		breakCue:      save(model.GearProduct{Category: model.GearCategoryBreakJump, BrandId: alpha.Id, Name: "A-Break", Roles: "break", Sort: 1, Status: 1}),
		jumpCue:       save(model.GearProduct{Category: model.GearCategoryBreakJump, BrandId: alpha.Id, Name: "A-Jump", Roles: "jump", Sort: 2, Status: 1}),
		breakJumpBoth: save(model.GearProduct{Category: model.GearCategoryBreakJump, BrandId: alpha.Id, Name: "A-BJ", Roles: "break,jump", Sort: 3, Status: 1}),
		tipAlpha:      save(model.GearProduct{Category: model.GearCategoryTip, BrandId: alpha.Id, Name: "A-Tip", Sort: 1, Status: 1}),
		tipOff:        save(model.GearProduct{Category: model.GearCategoryTip, BrandId: alpha.Id, Name: "A-Tip-Off", Sort: 2, Status: 0}),
		chalkAlpha:    save(model.GearProduct{Category: model.GearCategoryChalk, BrandId: alpha.Id, Name: "A-Chalk", Sort: 1, Status: 1}),
	}
	var houseCue, houseChalk model.GearProduct
	if err := e.db.Where("category = ? AND house = ?", model.GearCategoryCue, true).First(&houseCue).Error; err != nil {
		e.t.Fatalf("preset house cue missing: %v", err)
	}
	if err := e.db.Where("category = ? AND house = ?", model.GearCategoryChalk, true).First(&houseChalk).Error; err != nil {
		e.t.Fatalf("preset house chalk missing: %v", err)
	}
	e.c.houseCue, e.c.houseChalk = houseCue.Id, houseChalk.Id
}

// request 返回带用户身份与请求指标的 ctx，以及与它绑定的 ServiceContext，SQL 次数可从 metrics 读取。
func (e *gearEnv) request(userId int64) (context.Context, *svc.ServiceContext, *observability.RequestMetrics) {
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", userId), metrics)
	db := e.db.WithContext(ctx)
	return ctx, &svc.ServiceContext{
		GearModel:   model.NewGearModel(db),
		UserModel:   model.NewUserModel(db),
		FriendModel: model.NewFriendModel(db),
		MatchModel:  model.NewMatchModel(db),
	}, metrics
}

func (e *gearEnv) createItem(userId int64, req types.CreateGearItemReq) *types.GearItemResp {
	e.t.Helper()
	ctx, svcCtx, _ := e.request(userId)
	resp, err := NewCreateGearItemLogic(ctx, svcCtx).CreateGearItem(&req)
	if err != nil {
		e.t.Fatalf("create item returned an error: %v", err)
	}
	return resp
}

// mustCreate 登记一件装备并要求成功，返回装备 ID。
func (e *gearEnv) mustCreate(userId int64, req types.CreateGearItemReq) int64 {
	e.t.Helper()
	resp := e.createItem(userId, req)
	if !resp.Success || resp.Item == nil {
		e.t.Fatalf("expected create to succeed, got %+v", resp)
	}
	return resp.Item.Id
}

func (e *gearEnv) mine(userId int64) *types.GetGearMineResp {
	e.t.Helper()
	ctx, svcCtx, _ := e.request(userId)
	resp, err := NewGetGearMineLogic(ctx, svcCtx).GetGearMine()
	if err != nil {
		e.t.Fatalf("get mine returned an error: %v", err)
	}
	return resp
}

func (e *gearEnv) makeFriends(a, b int64) {
	e.t.Helper()
	if err := e.db.Create(&model.Friend{UserId: a, FriendId: b, Status: 1}).Error; err != nil {
		e.t.Fatalf("make friends: %v", err)
	}
}

func findItem(items []types.GearItemInfo, id int64) *types.GearItemInfo {
	for i := range items {
		if items[i].Id == id {
			return &items[i]
		}
	}
	return nil
}

// createMatch 创建一场对局；p2 为 0 表示对手不是注册用户。completedAt 为 nil 表示进行中。
func (e *gearEnv) createMatch(id int64, gameType int, p1, p2 int64, completedAt *time.Time) *model.Match {
	e.t.Helper()
	match := &model.Match{
		Id: id, UserId: p1, OpponentName: "对手", GameType: gameType, MatchMode: model.MatchModePractice,
		Visibility: model.MatchVisibilityPrivate, Status: 1, MatchTime: time.Now().Add(-time.Hour),
	}
	if p2 > 0 {
		match.OpponentId = &p2
	}
	if completedAt != nil {
		match.Status = 2
		match.CompletedAt = completedAt
		match.EndTime = completedAt
	}
	if err := e.db.Create(match).Error; err != nil {
		e.t.Fatalf("create match: %v", err)
	}
	return match
}

func timeAgo(d time.Duration) *time.Time {
	t := time.Now().Add(-d)
	return &t
}
