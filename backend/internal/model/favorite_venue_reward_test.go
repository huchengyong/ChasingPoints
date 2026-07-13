package model

import (
	"testing"
	"time"

	"chasing_points/internal/testsupport"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newFavoriteVenueRewardTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := testsupport.PrepareFavoriteVenueRewardSchema(db); err != nil {
		t.Fatalf("prepare favorite venue reward schema: %v", err)
	}

	return db
}

func TestFavoriteVenueRewardConfigModelReadsAndUpdatesConfig(t *testing.T) {
	db := newFavoriteVenueRewardTestDB(t)
	configModel := NewFavoriteVenueRewardConfigModel(db)

	config := &FavoriteVenueRewardConfig{
		ActivityKey:       FavoriteVenueRewardActivityKey,
		Enabled:           false,
		PopupEnabled:      false,
		RewardDays:        30,
		NewUserWindowDays: 7,
	}
	if err := configModel.Upsert(config); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	stored, err := configModel.FindByActivityKey(FavoriteVenueRewardActivityKey)
	if err != nil {
		t.Fatalf("find config: %v", err)
	}
	if stored == nil {
		t.Fatal("expected config, got nil")
	}
	if stored.RewardDays != 30 {
		t.Fatalf("expected reward days 30, got %d", stored.RewardDays)
	}

	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	config.Enabled = true
	config.PopupEnabled = true
	config.RewardDays = 45
	config.NewUserWindowDays = 10
	config.StartAt = &now
	if err := configModel.Upsert(config); err != nil {
		t.Fatalf("update config: %v", err)
	}

	updated, err := configModel.FindByActivityKey(FavoriteVenueRewardActivityKey)
	if err != nil {
		t.Fatalf("find updated config: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated config, got nil")
	}
	if !updated.Enabled {
		t.Fatal("expected enabled config")
	}
	if !updated.PopupEnabled {
		t.Fatal("expected popup enabled config")
	}
	if updated.RewardDays != 45 {
		t.Fatalf("expected reward days 45, got %d", updated.RewardDays)
	}
	if updated.NewUserWindowDays != 10 {
		t.Fatalf("expected new user window days 10, got %d", updated.NewUserWindowDays)
	}
	if updated.StartAt == nil || !updated.StartAt.Equal(now) {
		t.Fatalf("expected start_at %v, got %#v", now, updated.StartAt)
	}
}

func TestFavoriteVenueRewardRecordModelEnforcesUserAndVenueUniqueness(t *testing.T) {
	db := newFavoriteVenueRewardTestDB(t)
	userModel := NewUserModel(db)
	venueModel := NewVenueModel(db)
	recordModel := NewFavoriteVenueRewardRecordModel(db)

	if err := userModel.Create(&User{Id: 101, Nickname: "奖励用户"}); err != nil {
		t.Fatalf("create user 101: %v", err)
	}
	if err := userModel.Create(&User{Id: 102, Nickname: "另一位用户"}); err != nil {
		t.Fatalf("create user 102: %v", err)
	}
	if err := venueModel.Create(&Venue{
		Id:          202,
		Name:        "南山台球会",
		Address:     "科技园 1 号",
		City:        "深圳",
		District:    "南山区",
		FullAddress: "深圳南山区科技园1号",
		Status:      VenueStatusPublished,
		GeoStatus:   VenueGeoStatusSuccess,
	}); err != nil {
		t.Fatalf("create venue 202: %v", err)
	}
	if err := venueModel.Create(&Venue{
		Id:          203,
		Name:        "福田台球会",
		Address:     "深南大道 9 号",
		City:        "深圳",
		District:    "福田区",
		FullAddress: "深圳福田区深南大道9号",
		Status:      VenueStatusPublished,
		GeoStatus:   VenueGeoStatusSuccess,
	}); err != nil {
		t.Fatalf("create venue 203: %v", err)
	}

	before := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	after := before.Add(30 * 24 * time.Hour)
	record := &FavoriteVenueRewardRecord{
		ActivityKey:           FavoriteVenueRewardActivityKey,
		UserId:                101,
		VenueId:               202,
		RewardDays:            30,
		MemberExpiresAtBefore: &before,
		MemberExpiresAtAfter:  after,
		GrantedAt:             before,
	}
	if err := recordModel.Create(record); err != nil {
		t.Fatalf("create reward record: %v", err)
	}

	byUser, err := recordModel.FindByActivityAndUser(FavoriteVenueRewardActivityKey, 101)
	if err != nil {
		t.Fatalf("find reward by user: %v", err)
	}
	if byUser == nil || byUser.VenueId != 202 {
		t.Fatalf("expected reward record for user 101 and venue 202, got %#v", byUser)
	}

	byVenue, err := recordModel.FindByActivityAndVenue(FavoriteVenueRewardActivityKey, 202)
	if err != nil {
		t.Fatalf("find reward by venue: %v", err)
	}
	if byVenue == nil || byVenue.UserId != 101 {
		t.Fatalf("expected reward record for venue 202 and user 101, got %#v", byVenue)
	}

	duplicateUser := &FavoriteVenueRewardRecord{
		ActivityKey:          FavoriteVenueRewardActivityKey,
		UserId:               101,
		VenueId:              203,
		RewardDays:           30,
		MemberExpiresAtAfter: after,
		GrantedAt:            before,
	}
	if err := recordModel.Create(duplicateUser); err == nil {
		t.Fatal("expected duplicate user reward insert to fail")
	}

	duplicateVenue := &FavoriteVenueRewardRecord{
		ActivityKey:          FavoriteVenueRewardActivityKey,
		UserId:               102,
		VenueId:              202,
		RewardDays:           30,
		MemberExpiresAtAfter: after,
		GrantedAt:            before,
	}
	if err := recordModel.Create(duplicateVenue); err == nil {
		t.Fatal("expected duplicate venue reward insert to fail")
	}
}

func TestFavoriteVenueRewardRecordModelFindAdminListReturnsJoinedData(t *testing.T) {
	db := newFavoriteVenueRewardTestDB(t)
	userModel := NewUserModel(db)
	venueModel := NewVenueModel(db)
	recordModel := NewFavoriteVenueRewardRecordModel(db)

	phone := "13800000000"
	if err := userModel.Create(&User{Id: 201, Nickname: "会员用户", Phone: &phone}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := venueModel.Create(&Venue{
		Id:          301,
		Name:        "宝安桌球",
		Address:     "创业路 8 号",
		City:        "深圳",
		District:    "宝安区",
		FullAddress: "深圳宝安区创业路8号",
		Status:      VenueStatusPublished,
		GeoStatus:   VenueGeoStatusSuccess,
	}); err != nil {
		t.Fatalf("create venue: %v", err)
	}

	grantedAt := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	expiresAt := grantedAt.Add(30 * 24 * time.Hour)
	if err := recordModel.Create(&FavoriteVenueRewardRecord{
		ActivityKey:          FavoriteVenueRewardActivityKey,
		UserId:               201,
		VenueId:              301,
		RewardDays:           30,
		MemberExpiresAtAfter: expiresAt,
		GrantedAt:            grantedAt,
	}); err != nil {
		t.Fatalf("create reward record: %v", err)
	}

	list, total, err := recordModel.FindAdminList(FavoriteVenueRewardActivityKey, 1, 20)
	if err != nil {
		t.Fatalf("find admin list: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 joined reward record, got total=%d list=%d", total, len(list))
	}

	item := list[0]
	if item.UserNickname != "会员用户" || item.UserPhone != phone {
		t.Fatalf("unexpected joined user data: %#v", item)
	}
	if item.VenueName != "宝安桌球" || item.VenueId != 301 {
		t.Fatalf("unexpected joined venue data: %#v", item)
	}
	if item.MemberExpiresAtAfter.IsZero() || !item.GrantedAt.Equal(grantedAt) {
		t.Fatalf("unexpected reward timestamps: %#v", item)
	}
}
