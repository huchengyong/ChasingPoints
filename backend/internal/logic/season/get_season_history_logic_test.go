package season

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeasonHistoryRejectsMissingAuthentication(t *testing.T) {
	resp, err := NewGetSeasonHistoryLogic(context.Background(), &svc.ServiceContext{}).GetSeasonHistory(&types.GetSeasonHistoryReq{})
	if err != nil || resp.Success || len(resp.List) != 0 {
		t.Fatalf("unauthenticated season history must not return data: resp=%#v err=%v", resp, err)
	}
}

func TestSeasonHistoryListsOnlySettledSeasonsWithViewerRecords(t *testing.T) {
	db := newSeasonHistoryTestDB(t, t.Name())
	seedSeasonHistory(t, db, 3)
	month := func(offset int) time.Time { return time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC).AddDate(0, offset, 0) }
	if err := db.Create(&[]model.Season{
		{Id: 4, Name: "S4", StartDate: month(3), EndDate: month(4).AddDate(0, 0, -1), Status: 1},
		{Id: 5, Name: "S5", StartDate: month(4), EndDate: month(5).AddDate(0, 0, -1), Status: 0},
	}).Error; err != nil {
		t.Fatalf("seed unsettled seasons: %v", err)
	}
	if err := db.Create(&[]model.SeasonRecord{
		{SeasonId: 1, UserId: 1, GameType: 3, EndRankScore: 1100, MatchesPlayed: 4, Wins: 3, FinalRank: 2},
		{SeasonId: 3, UserId: 1, GameType: 3, EndRankScore: 1200, MatchesPlayed: 9, Wins: 6, FinalRank: 1},
		{SeasonId: 3, UserId: 1, GameType: 1, EndRankScore: 900, MatchesPlayed: 1, Wins: 0, FinalRank: 4},
		{SeasonId: 4, UserId: 1, GameType: 3, EndRankScore: 1300, MatchesPlayed: 2, Wins: 2},
		{SeasonId: 2, UserId: 2, GameType: 3, EndRankScore: 1000, MatchesPlayed: 1, Wins: 1, FinalRank: 1},
	}).Error; err != nil {
		t.Fatalf("seed season records: %v", err)
	}
	svcCtx := &svc.ServiceContext{SeasonModel: model.NewSeasonModel(db), SeasonRecordModel: model.NewSeasonRecordModel(db)}
	ctx := context.WithValue(context.Background(), "user_id", int64(1))

	resp, err := NewGetSeasonHistoryLogic(ctx, svcCtx).GetSeasonHistory(&types.GetSeasonHistoryReq{GameType: 3, Page: 1, PageSize: 10})
	if err != nil || !resp.Success || resp.Total != 3 || len(resp.List) != 3 {
		t.Fatalf("history must list settled seasons only: resp=%#v err=%v", resp, err)
	}
	for index, wantID := range []int64{3, 2, 1} {
		if resp.List[index].Season == nil || resp.List[index].Season.Id != wantID || resp.List[index].Season.Status != 2 {
			t.Fatalf("history item %d must be settled season %d newest first: %+v", index, wantID, resp.List[index].Season)
		}
	}
	if resp.List[0].Record == nil || resp.List[0].Record.FinalRank != 1 || resp.List[0].Record.SeasonName != "S3" {
		t.Fatalf("latest season must carry viewer record for requested game type: %+v", resp.List[0].Record)
	}
	if resp.List[1].Record != nil {
		t.Fatalf("season without viewer record must return null record: %+v", resp.List[1].Record)
	}
	if resp.List[2].Record == nil || resp.List[2].Record.FinalRank != 2 {
		t.Fatalf("older season must carry viewer record: %+v", resp.List[2].Record)
	}

	snooker, err := NewGetSeasonHistoryLogic(ctx, svcCtx).GetSeasonHistory(&types.GetSeasonHistoryReq{GameType: 1, Page: 1, PageSize: 10})
	if err != nil || !snooker.Success || len(snooker.List) != 3 || snooker.List[0].Record == nil || snooker.List[0].Record.FinalRank != 4 || snooker.List[2].Record != nil {
		t.Fatalf("history records must follow requested game type: resp=%#v err=%v", snooker, err)
	}
}

func TestSeasonHistoryNormalizesOutOfRangePaging(t *testing.T) {
	db := newSeasonHistoryTestDB(t, t.Name())
	seedSeasonHistory(t, db, 25)
	svcCtx := &svc.ServiceContext{SeasonModel: model.NewSeasonModel(db), SeasonRecordModel: model.NewSeasonRecordModel(db)}
	ctx := context.WithValue(context.Background(), "user_id", int64(1))

	defaults, err := NewGetSeasonHistoryLogic(ctx, svcCtx).GetSeasonHistory(&types.GetSeasonHistoryReq{Page: 0, PageSize: 0})
	if err != nil || !defaults.Success || defaults.Total != 25 || len(defaults.List) != 10 || defaults.List[0].Season.Id != 25 {
		t.Fatalf("non-positive paging must normalize to first default page: resp=%#v err=%v", defaults, err)
	}
	capped, err := NewGetSeasonHistoryLogic(ctx, svcCtx).GetSeasonHistory(&types.GetSeasonHistoryReq{Page: 1, PageSize: 100})
	if err != nil || !capped.Success || len(capped.List) != 20 {
		t.Fatalf("page size must be capped at 20: len=%d err=%v", len(capped.List), err)
	}
	beyond, err := NewGetSeasonHistoryLogic(ctx, svcCtx).GetSeasonHistory(&types.GetSeasonHistoryReq{Page: 9, PageSize: 10})
	if err != nil || !beyond.Success || beyond.Total != 25 || len(beyond.List) != 0 {
		t.Fatalf("page beyond total must be an empty success: resp=%#v err=%v", beyond, err)
	}
}

func TestSeasonHistoryUsesFixedQueryCountForOneAndTwentySeasons(t *testing.T) {
	one := seasonHistoryQueryCount(t, 1)
	twenty := seasonHistoryQueryCount(t, 20)
	if one != 3 || twenty != 3 {
		t.Fatalf("season history must use three fixed queries: one=%d twenty=%d", one, twenty)
	}
}

func seasonHistoryQueryCount(t *testing.T, seasons int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", seasons)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonRecord{}); err != nil {
		t.Fatalf("prepare season history schema: %v", err)
	}
	seedSeasonHistory(t, db, seasons)
	records := make([]model.SeasonRecord, 0, seasons)
	for index := 1; index <= seasons; index++ {
		records = append(records, model.SeasonRecord{SeasonId: int64(index), UserId: 1, GameType: 3, MatchesPlayed: 1, Wins: 1, FinalRank: 1})
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("seed season records: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	requestDB := db.WithContext(ctx)
	resp, err := NewGetSeasonHistoryLogic(ctx, &svc.ServiceContext{SeasonModel: model.NewSeasonModel(requestDB), SeasonRecordModel: model.NewSeasonRecordModel(requestDB)}).GetSeasonHistory(&types.GetSeasonHistoryReq{GameType: 3, Page: 1, PageSize: 20})
	if err != nil || !resp.Success || len(resp.List) != seasons {
		t.Fatalf("get season history: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}

func newSeasonHistoryTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonRecord{}); err != nil {
		t.Fatalf("prepare season history schema: %v", err)
	}
	return db
}

func seedSeasonHistory(t *testing.T, db *gorm.DB, count int) {
	t.Helper()
	seasons := make([]model.Season, 0, count)
	anchor := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for index := 1; index <= count; index++ {
		start := anchor.AddDate(0, index-1, 0)
		seasons = append(seasons, model.Season{Id: int64(index), Name: fmt.Sprintf("S%d", index), StartDate: start, EndDate: start.AddDate(0, 1, -1), Status: 2})
	}
	if err := db.Create(&seasons).Error; err != nil {
		t.Fatalf("seed settled seasons: %v", err)
	}
}
