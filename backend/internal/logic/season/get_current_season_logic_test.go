package season

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCurrentSeasonLogicTestSvc(t *testing.T, lifecycle config.SeasonLifecycleConfig) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.User{}, &model.SeasonRecord{}); err != nil {
		t.Fatalf("migrate seasons: %v", err)
	}
	return &svc.ServiceContext{
		Config:            config.Config{SeasonLifecycle: lifecycle},
		DB:                db,
		SeasonModel:       model.NewSeasonModel(db),
		UserModel:         model.NewUserModel(db),
		SeasonRecordModel: model.NewSeasonRecordModel(db),
	}
}

func TestDefaultSeasonReadsUseCurrentLifecycleWindow(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	now := time.Now()
	anchor := time.Date(now.In(location).Year(), now.In(location).Month(), 1, 0, 0, 0, 0, location)
	lifecycle := config.SeasonLifecycleConfig{Enabled: true, AnchorDate: anchor.Format(time.DateOnly)}
	svcCtx := newCurrentSeasonLogicTestSvc(t, lifecycle)
	policy, err := seasonx.NewPolicy(lifecycle)
	if err != nil {
		t.Fatalf("new policy: %v", err)
	}
	window, ok := policy.WindowAt(now)
	if !ok {
		t.Fatal("expected current window")
	}
	current := seasonx.WindowSeason(window)
	current.Status = 0
	if err := svcCtx.SeasonModel.Create(&current); err != nil {
		t.Fatalf("seed current window: %v", err)
	}
	if err := svcCtx.UserModel.Create(&model.User{Id: 9, Nickname: "球友", Status: 1}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := svcCtx.SeasonRecordModel.Create(&model.SeasonRecord{SeasonId: current.Id, UserId: 9, GameType: 3, MatchesPlayed: 2, Wins: 1}); err != nil {
		t.Fatalf("seed season record: %v", err)
	}

	leaderboard, err := NewGetSeasonLeaderboardLogic(context.Background(), svcCtx).GetSeasonLeaderboard(&types.GetSeasonLeaderboardReq{})
	if err != nil || !leaderboard.Success || leaderboard.Season == nil || leaderboard.Season.Id != current.Id || leaderboard.Season.Status != 1 {
		t.Fatalf("default leaderboard must resolve the active lifecycle window: %+v err=%v", leaderboard, err)
	}
	userCtx := context.WithValue(context.Background(), "user_id", int64(9))
	myRecord, err := NewGetMySeasonRecordLogic(userCtx, svcCtx).GetMySeasonRecord(&types.GetMySeasonRecordReq{})
	if err != nil || !myRecord.Success || myRecord.Record == nil || myRecord.Record.SeasonId != current.Id {
		t.Fatalf("default personal record must resolve the active lifecycle window: %+v err=%v", myRecord, err)
	}
}

func TestGetCurrentSeasonLifecycleStates(t *testing.T) {
	now := time.Now()
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	currentMonth := time.Date(now.In(location).Year(), now.In(location).Month(), 1, 0, 0, 0, 0, location)

	for _, tc := range []struct {
		name           string
		lifecycle      config.SeasonLifecycleConfig
		seedCurrent    bool
		wantState      string
		wantSeason     bool
		wantWireStatus int
		wantBounds     bool
	}{
		{name: "not started", lifecycle: config.SeasonLifecycleConfig{Enabled: true, AnchorDate: currentMonth.AddDate(0, 1, 0).Format(time.DateOnly)}, wantState: seasonx.StateNotStarted},
		{name: "unavailable schedule", lifecycle: config.SeasonLifecycleConfig{Enabled: true, AnchorDate: currentMonth.Format(time.DateOnly)}, wantState: seasonx.StateUnavailable},
		{name: "active despite delayed status", lifecycle: config.SeasonLifecycleConfig{Enabled: true, AnchorDate: currentMonth.Format(time.DateOnly)}, seedCurrent: true, wantState: seasonx.StateActive, wantSeason: true, wantWireStatus: 1, wantBounds: true},
		{name: "legacy empty schedule", lifecycle: config.SeasonLifecycleConfig{}, wantState: seasonx.StateNotStarted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx := newCurrentSeasonLogicTestSvc(t, tc.lifecycle)
			if tc.seedCurrent {
				policy, policyErr := seasonx.NewPolicy(tc.lifecycle)
				if policyErr != nil {
					t.Fatalf("new policy: %v", policyErr)
				}
				window, ok := policy.WindowAt(now)
				if !ok {
					t.Fatal("expected current window")
				}
				item := seasonx.WindowSeason(window)
				item.Status = 0
				if err := svcCtx.SeasonModel.Create(&item); err != nil {
					t.Fatalf("seed delayed status season: %v", err)
				}
			}
			resp, logicErr := NewGetCurrentSeasonLogic(context.Background(), svcCtx).GetCurrentSeason()
			if logicErr != nil {
				t.Fatalf("get current season: %v", logicErr)
			}
			if !resp.Success || resp.SeasonState != tc.wantState || (resp.Season != nil) != tc.wantSeason {
				t.Fatalf("unexpected current season response: %+v", resp)
			}
			if tc.wantSeason && resp.Season.Status != tc.wantWireStatus {
				t.Fatalf("wire season must be active: %+v", resp.Season)
			}
			if tc.wantBounds && (resp.Season.StartAt == "" || resp.Season.EndAtExclusive == "") {
				t.Fatalf("current season must include authoritative half-open bounds: %+v", resp.Season)
			}
		})
	}
}

func TestBuildSeasonInfoUsesConfiguredBusinessBounds(t *testing.T) {
	season := &model.Season{
		Id:        1,
		Name:      "S1",
		StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
	}
	info := buildSeasonInfo(&svc.ServiceContext{Config: config.Config{
		SeasonLifecycle: config.SeasonLifecycleConfig{Timezone: "America/New_York"},
	}}, season)
	if info.StartAt != "2026-07-01T00:00:00-04:00" || info.EndAtExclusive != "2026-08-01T00:00:00-04:00" {
		t.Fatalf("season bounds must preserve configured business timezone: %+v", info)
	}
}
