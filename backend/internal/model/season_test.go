package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeasonModelFindsAndCreatesDeterministicWindows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Season{}); err != nil {
		t.Fatalf("migrate season: %v", err)
	}
	model := NewSeasonModel(db)
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	item := &Season{Name: "S2", StartDate: start, EndDate: start.AddDate(0, 1, -1), Status: 0}
	created, err := model.CreateIfAbsentWithTx(nil, item)
	if err != nil || !created {
		t.Fatalf("create deterministic window: created=%t err=%v", created, err)
	}
	created, err = model.CreateIfAbsentWithTx(nil, &Season{Name: "S2", StartDate: start, EndDate: start.AddDate(0, 1, -1)})
	if err != nil || created {
		t.Fatalf("repeat deterministic window: created=%t err=%v", created, err)
	}

	found, err := model.FindByStartDate(start)
	if err != nil || found == nil || found.Id != item.Id {
		t.Fatalf("find by start date: season=%+v err=%v", found, err)
	}
	byTime, err := model.FindByEffectiveTime(start.Add(12 * time.Hour))
	if err != nil || byTime == nil || byTime.Id != item.Id {
		t.Fatalf("find by effective time: season=%+v err=%v", byTime, err)
	}
	byTime, err = model.FindByEffectiveTime(item.EndDate.Add(23 * time.Hour))
	if err != nil || byTime == nil || byTime.Id != item.Id {
		t.Fatalf("end date must remain in the effective season: season=%+v err=%v", byTime, err)
	}
	if changed, err := model.UpdateStatusToWithTx(nil, item.Id, 1); err != nil || !changed {
		t.Fatalf("activate season: changed=%t err=%v", changed, err)
	}
	if changed, err := model.UpdateStatusToWithTx(nil, item.Id, 1); err != nil || changed {
		t.Fatalf("same status must not write: changed=%t err=%v", changed, err)
	}
}

func TestSeasonModelUsesBusinessTimezoneForEffectiveDate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Season{}); err != nil {
		t.Fatalf("migrate seasons: %v", err)
	}
	if err := db.Create(&[]Season{
		{Name: "July", StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)},
		{Name: "August", StartDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)},
	}).Error; err != nil {
		t.Fatalf("seed seasons: %v", err)
	}
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load timezone: %v", err)
	}
	season, err := NewSeasonModel(db).FindByEffectiveTimeInLocationWithTx(nil, time.Date(2026, 7, 31, 16, 30, 0, 0, time.UTC), shanghai)
	if err != nil || season == nil || season.Name != "August" {
		t.Fatalf("Shanghai Aug 1 match must belong to August: season=%+v err=%v", season, err)
	}
}

func TestSeasonModelFindsDueUnsettledWindowsByBoundedPage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Season{}, &SeasonSettlement{}); err != nil {
		t.Fatalf("migrate seasons: %v", err)
	}
	utc := time.UTC
	july := Season{Name: "S1", StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, utc), EndDate: time.Date(2026, 7, 31, 0, 0, 0, 0, utc), Status: 1}
	august := Season{Name: "S2", StartDate: time.Date(2026, 8, 1, 0, 0, 0, 0, utc), EndDate: time.Date(2026, 8, 31, 0, 0, 0, 0, utc), Status: 0}
	if err := db.Create(&[]Season{july, august}).Error; err != nil {
		t.Fatalf("seed seasons: %v", err)
	}
	if err := db.Create(&SeasonSettlement{SeasonId: 2, Status: SeasonSettlementStatusCompleted}).Error; err != nil {
		t.Fatalf("seed completed settlement: %v", err)
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	model := NewSeasonModel(db)
	items, err := model.FindDueUnsettledBatch(
		time.Date(2026, 7, 1, 0, 0, 0, 0, location),
		time.Date(2026, 9, 1, 0, 0, 0, 0, location),
		time.Time{},
		0,
		1,
	)
	if err != nil || len(items) != 1 || items[0].Name != "S1" {
		t.Fatalf("find first due page: items=%+v err=%v", items, err)
	}
	next, err := model.FindDueUnsettledBatch(
		time.Date(2026, 7, 1, 0, 0, 0, 0, location),
		time.Date(2026, 9, 1, 0, 0, 0, 0, location),
		items[0].EndDate,
		items[0].Id,
		1,
	)
	if err != nil || len(next) != 0 {
		t.Fatalf("completed and consumed windows must not repeat: items=%+v err=%v", next, err)
	}
}

func TestSeasonRecordCompetitiveInsertLeavesRewardsNull(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&SeasonRecord{}); err != nil {
		t.Fatalf("migrate season records: %v", err)
	}
	seasonRecords := NewSeasonRecordModel(db)
	if err := seasonRecords.ApplyCompetitiveMatchWithTx(nil, SeasonRecordMatchDelta{
		SeasonId:       1,
		UserId:         7,
		GameType:       3,
		StartRankScore: 100,
		EndRankScore:   110,
		Won:            true,
	}); err != nil {
		t.Fatalf("apply competitive match: %v", err)
	}
	record, err := seasonRecords.FindBySeasonAndUserAndGameType(1, 7, 3)
	if err != nil || record == nil || record.Rewards != nil {
		t.Fatalf("new season record must leave JSON rewards null: record=%+v err=%v", record, err)
	}
}

func TestSeasonReportWinQueryUsesIncrementalSeasonRecords(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&SeasonRecord{}); err != nil {
		t.Fatalf("migrate season records: %v", err)
	}
	if err := db.Create(&[]SeasonRecord{
		{SeasonId: 7, UserId: 1, GameType: 2, Wins: 2},
		{SeasonId: 7, UserId: 1, GameType: 3, Wins: 3},
		{SeasonId: 7, UserId: 2, GameType: 3, Wins: 9},
		{SeasonId: 8, UserId: 1, GameType: 3, Wins: 10},
	}).Error; err != nil {
		t.Fatalf("seed season records: %v", err)
	}
	wins, err := NewSeasonRecordModel(db).FindUserWinsBySeason(7, 1)
	if err != nil || wins["2"] != 2 || wins["3"] != 3 || len(wins) != 2 {
		t.Fatalf("season report must read incremental wins only: wins=%+v err=%v", wins, err)
	}
}
