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

func TestSeasonReportWinQueryUsesCompletedRankedMatchesForBothPlayers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Match{}, &MatchRound{}); err != nil {
		t.Fatalf("migrate report matches: %v", err)
	}

	userID := int64(1)
	opponentID := int64(2)
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	resultWin := 1
	resultLoss := 2
	matches := []Match{
		{Id: 1, UserId: userID, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: MatchModeRanked, Status: 2, Result: &resultWin, MatchTime: start.Add(time.Hour)},
		{Id: 2, UserId: opponentID, OpponentId: &userID, OpponentName: "本人", GameType: 3, MatchMode: MatchModeRanked, Status: 2, Result: &resultLoss, MatchTime: start.Add(2 * time.Hour)},
		{Id: 3, UserId: userID, OpponentId: &opponentID, OpponentName: "练习对手", GameType: 3, MatchMode: MatchModePractice, Status: 2, Result: &resultWin, MatchTime: start.Add(3 * time.Hour)},
		{Id: 4, UserId: userID, OpponentId: &opponentID, OpponentName: "删除对手", GameType: 3, MatchMode: MatchModeRanked, Status: 2, Result: &resultWin, MatchTime: start.Add(4 * time.Hour)},
		{Id: 5, UserId: userID, OpponentId: &opponentID, OpponentName: "未完成对手", GameType: 3, MatchMode: MatchModeRanked, Status: 2, Result: &resultWin, MatchTime: start.Add(5 * time.Hour)},
		{Id: 6, UserId: userID, OpponentId: &opponentID, OpponentName: "历史对手", GameType: 3, MatchMode: MatchModeRanked, Status: 2, Result: &resultWin, MatchTime: start.Add(6 * time.Hour)},
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed report matches: %v", err)
	}
	winner := 1
	rounds := []MatchRound{
		{MatchId: 1, RoundNo: 1, Winner: &winner, WinType: "normal"},
		{MatchId: 2, RoundNo: 1, Winner: &winner, WinType: "normal"},
		{MatchId: 3, RoundNo: 1, Winner: &winner, WinType: "normal"},
		{MatchId: 4, RoundNo: 1, Winner: &winner, WinType: "normal"},
		{MatchId: 5, RoundNo: 1, Winner: &winner, WinType: "start"},
		{MatchId: 6, RoundNo: 1, Winner: &winner, WinType: "normal"},
	}
	if err := db.Create(&rounds).Error; err != nil {
		t.Fatalf("seed report rounds: %v", err)
	}
	if err := db.Model(&MatchRound{}).Where("match_id = ?", 6).Update("win_type", nil).Error; err != nil {
		t.Fatalf("seed nullable historical round: %v", err)
	}
	if err := db.Delete(&Match{}, 4).Error; err != nil {
		t.Fatalf("soft delete report match: %v", err)
	}

	wins, err := NewSeasonRecordModel(db).FindUserWinByTypeInSeasonHalfOpen(userID, start, start.AddDate(0, 1, 0))
	if err != nil {
		t.Fatalf("query report wins: %v", err)
	}
	if wins["3"] != 3 {
		t.Fatalf("expected creator, opponent, and nullable-history wins only, got %+v", wins)
	}
}
