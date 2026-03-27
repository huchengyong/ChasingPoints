package admin

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newVenueRewardAdminTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareFavoriteVenueRewardSchema(db); err != nil {
		t.Fatalf("prepare favorite venue reward schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                             db,
		UserModel:                      model.NewUserModel(db),
		VenueModel:                     model.NewVenueModel(db),
		FavoriteVenueRewardConfigModel: model.NewFavoriteVenueRewardConfigModel(db),
		FavoriteVenueRewardRecordModel: model.NewFavoriteVenueRewardRecordModel(db),
	}
}

func TestAdminVenueRewardConfigRoundTrip(t *testing.T) {
	svcCtx := newVenueRewardAdminTestSvc(t)

	if err := svcCtx.FavoriteVenueRewardConfigModel.Upsert(&model.FavoriteVenueRewardConfig{
		ActivityKey:       model.FavoriteVenueRewardActivityKey,
		Enabled:           false,
		PopupEnabled:      false,
		RewardDays:        30,
		NewUserWindowDays: 7,
	}); err != nil {
		t.Fatalf("seed reward config: %v", err)
	}

	getLogic := NewAdminGetVenueRewardConfigLogic(context.Background(), svcCtx)
	getResp, err := getLogic.AdminGetVenueRewardConfig()
	if err != nil {
		t.Fatalf("get reward config: %v", err)
	}
	if !getResp.Success || getResp.RewardDays != 30 || getResp.NewUserWindowDays != 7 {
		t.Fatalf("unexpected get resp: %#v", getResp)
	}

	updateLogic := NewAdminUpdateVenueRewardConfigLogic(context.Background(), svcCtx)
	updateResp, err := updateLogic.AdminUpdateVenueRewardConfig(&types.AdminVenueRewardConfigUpdateReq{
		Enabled:           true,
		PopupEnabled:      true,
		RewardDays:        45,
		NewUserWindowDays: 10,
		StartAt:           "2026-03-26 12:00:00",
		EndAt:             "2026-04-26 12:00:00",
	})
	if err != nil {
		t.Fatalf("update reward config: %v", err)
	}
	if !updateResp.Success || updateResp.Code != 0 {
		t.Fatalf("unexpected update resp: %#v", updateResp)
	}

	stored, err := svcCtx.FavoriteVenueRewardConfigModel.FindByActivityKey(model.FavoriteVenueRewardActivityKey)
	if err != nil {
		t.Fatalf("find updated config: %v", err)
	}
	if stored == nil {
		t.Fatal("expected updated config, got nil")
	}
	if !stored.Enabled || !stored.PopupEnabled {
		t.Fatalf("expected enabled popup config, got %#v", stored)
	}
	if stored.RewardDays != 45 || stored.NewUserWindowDays != 10 {
		t.Fatalf("unexpected updated reward config: %#v", stored)
	}
	if stored.StartAt == nil || stored.EndAt == nil {
		t.Fatalf("expected start and end time, got %#v", stored)
	}
}

func TestAdminUpdateVenueRewardConfigRejectsInvalidValues(t *testing.T) {
	svcCtx := newVenueRewardAdminTestSvc(t)
	logic := NewAdminUpdateVenueRewardConfigLogic(context.Background(), svcCtx)

	resp, err := logic.AdminUpdateVenueRewardConfig(&types.AdminVenueRewardConfigUpdateReq{
		Enabled:           true,
		PopupEnabled:      true,
		RewardDays:        0,
		NewUserWindowDays: 0,
	})
	if err != nil {
		t.Fatalf("update reward config with invalid values: %v", err)
	}
	if resp.Success || resp.Code != 400 {
		t.Fatalf("expected validation failure, got %#v", resp)
	}
}

func TestAdminGetVenueRewardConfigReturnsDefaultDisabledWhenMissing(t *testing.T) {
	svcCtx := newVenueRewardAdminTestSvc(t)
	logic := NewAdminGetVenueRewardConfigLogic(context.Background(), svcCtx)

	resp, err := logic.AdminGetVenueRewardConfig()
	if err != nil {
		t.Fatalf("get reward config without row: %v", err)
	}
	if !resp.Success || resp.Enabled || resp.PopupEnabled {
		t.Fatalf("expected disabled default config, got %#v", resp)
	}
	if resp.RewardDays != 30 || resp.NewUserWindowDays != 7 {
		t.Fatalf("expected default reward values, got %#v", resp)
	}
}

func TestParseAdminRewardOptionalTimeUsesShanghaiClock(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.UTC
	t.Cleanup(func() {
		time.Local = originalLocal
	})

	parsed, err := parseAdminRewardOptionalTime("2026-03-26 12:00:00")
	if err != nil {
		t.Fatalf("parse reward time: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected parsed time, got nil")
	}
	_, offset := parsed.Zone()
	if offset != 8*60*60 {
		t.Fatalf("expected +08:00 reward time, got offset=%d parsed=%v", offset, parsed)
	}
}

func TestAdminGetVenueRewardRecordListReturnsJoinedRows(t *testing.T) {
	svcCtx := newVenueRewardAdminTestSvc(t)

	phone := "13900000000"
	if err := svcCtx.UserModel.Create(&model.User{Id: 301, Nickname: "记录用户", Phone: &phone}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := svcCtx.VenueModel.Create(&model.Venue{
		Id:          401,
		Name:        "记录球馆",
		Address:     "香蜜湖 18 号",
		City:        "深圳",
		District:    "福田区",
		FullAddress: "深圳福田区香蜜湖18号",
		Status:      model.VenueStatusPublished,
		GeoStatus:   model.VenueGeoStatusSuccess,
	}); err != nil {
		t.Fatalf("create venue: %v", err)
	}

	grantedAt := mustParseRewardTime(t, "2026-03-26 15:00:00")
	expiresAt := grantedAt.Add(30 * 24 * time.Hour)
	if err := svcCtx.FavoriteVenueRewardRecordModel.Create(&model.FavoriteVenueRewardRecord{
		ActivityKey:          model.FavoriteVenueRewardActivityKey,
		UserId:               301,
		VenueId:              401,
		RewardDays:           30,
		MemberExpiresAtAfter: expiresAt,
		GrantedAt:            grantedAt,
	}); err != nil {
		t.Fatalf("create reward record: %v", err)
	}

	logic := NewAdminGetVenueRewardRecordListLogic(context.Background(), svcCtx)
	resp, err := logic.AdminGetVenueRewardRecordList(&types.AdminVenueRewardRecordListReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("get reward record list: %v", err)
	}
	if !resp.Success || resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("unexpected reward record list resp: %#v", resp)
	}
	if resp.List[0].UserNickname != "记录用户" || resp.List[0].VenueName != "记录球馆" {
		t.Fatalf("unexpected reward record item: %#v", resp.List[0])
	}
}

func mustParseRewardTime(t *testing.T, raw string) time.Time {
	t.Helper()

	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local)
	if err != nil {
		t.Fatalf("parse time %q: %v", raw, err)
	}
	return parsed
}
