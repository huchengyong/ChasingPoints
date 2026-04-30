package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMatchH2HTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(&User{}, &Match{}); err != nil {
		t.Fatalf("prepare h2h schema: %v", err)
	}

	return db
}

func TestMatchModelGetH2HStatsByOpponentUsesBidirectionalMatches(t *testing.T) {
	db := newMatchH2HTestDB(t)
	userModel := NewUserModel(db)
	matchModel := NewMatchModel(db)

	if err := userModel.Create(&User{Id: 1, Nickname: "我"}); err != nil {
		t.Fatalf("create user 1: %v", err)
	}
	if err := userModel.Create(&User{Id: 2, Nickname: "现在昵称"}); err != nil {
		t.Fatalf("create user 2: %v", err)
	}

	opponentID := int64(2)
	userID := int64(1)
	win := 1

	if err := matchModel.Create(&Match{
		Id:            101,
		UserId:        userID,
		OpponentId:    &opponentID,
		OpponentName:  "旧昵称",
		GameType:      3,
		MyScore:       5,
		OpponentScore: 3,
		Status:        2,
		Result:        &win,
		MatchTime:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("create match 101: %v", err)
	}

	if err := matchModel.Create(&Match{
		Id:            102,
		UserId:        opponentID,
		OpponentId:    &userID,
		OpponentName:  "我",
		GameType:      3,
		MyScore:       4,
		OpponentScore: 2,
		Status:        2,
		Result:        &win,
		MatchTime:     time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("create match 102: %v", err)
	}

	total, myWins, oppWins, avgDiff, maxWinStreak, err := matchModel.GetH2HStatsByOpponent(userID, opponentID, "现在昵称")
	if err != nil {
		t.Fatalf("get h2h stats: %v", err)
	}

	if total != 2 {
		t.Fatalf("expected 2 total matches, got %d", total)
	}
	if myWins != 1 {
		t.Fatalf("expected 1 win from user perspective, got %d", myWins)
	}
	if oppWins != 1 {
		t.Fatalf("expected 1 opponent win from user perspective, got %d", oppWins)
	}
	if avgDiff != 0 {
		t.Fatalf("expected bidirectional score diff average 0, got %v", avgDiff)
	}
	if maxWinStreak != 1 {
		t.Fatalf("expected max win streak 1, got %d", maxWinStreak)
	}
}

func TestMatchModelGetH2HStatsByOpponentFallsBackToNameForAnonymousOpponents(t *testing.T) {
	db := newMatchH2HTestDB(t)
	matchModel := NewMatchModel(db)

	win := 1
	lose := 2

	if err := matchModel.Create(&Match{
		Id:            201,
		UserId:        1,
		OpponentName:  "线下朋友",
		GameType:      2,
		MyScore:       21,
		OpponentScore: 18,
		Status:        2,
		Result:        &win,
		MatchTime:     time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("create match 201: %v", err)
	}

	if err := matchModel.Create(&Match{
		Id:            202,
		UserId:        1,
		OpponentName:  "线下朋友",
		GameType:      2,
		MyScore:       16,
		OpponentScore: 21,
		Status:        2,
		Result:        &lose,
		MatchTime:     time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("create match 202: %v", err)
	}

	total, myWins, oppWins, avgDiff, maxWinStreak, err := matchModel.GetH2HStatsByOpponent(1, 0, "线下朋友")
	if err != nil {
		t.Fatalf("get anonymous opponent stats: %v", err)
	}

	if total != 2 || myWins != 1 || oppWins != 1 {
		t.Fatalf("expected 2 matches split 1:1, got total=%d myWins=%d oppWins=%d", total, myWins, oppWins)
	}
	if avgDiff != -1 {
		t.Fatalf("expected average score diff -1, got %v", avgDiff)
	}
	if maxWinStreak != 1 {
		t.Fatalf("expected anonymous max win streak 1, got %d", maxWinStreak)
	}
}

func TestMatchModelGetH2HStatsByOpponentCalculatesLongestWinStreak(t *testing.T) {
	db := newMatchH2HTestDB(t)
	userModel := NewUserModel(db)
	matchModel := NewMatchModel(db)

	if err := userModel.Create(&User{Id: 1, Nickname: "我"}); err != nil {
		t.Fatalf("create user 1: %v", err)
	}
	if err := userModel.Create(&User{Id: 2, Nickname: "球友"}); err != nil {
		t.Fatalf("create user 2: %v", err)
	}

	userID := int64(1)
	opponentID := int64(2)
	win := 1
	lose := 2

	matches := []*Match{
		{
			Id:            301,
			UserId:        userID,
			OpponentId:    &opponentID,
			OpponentName:  "球友旧名",
			GameType:      3,
			MyScore:       7,
			OpponentScore: 4,
			Status:        2,
			Result:        &win,
			MatchTime:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			Id:            302,
			UserId:        opponentID,
			OpponentId:    &userID,
			OpponentName:  "我",
			GameType:      3,
			MyScore:       4,
			OpponentScore: 7,
			Status:        2,
			Result:        &lose,
			MatchTime:     time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
		},
		{
			Id:            303,
			UserId:        userID,
			OpponentId:    &opponentID,
			OpponentName:  "球友旧名",
			GameType:      3,
			MyScore:       3,
			OpponentScore: 7,
			Status:        2,
			Result:        &lose,
			MatchTime:     time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC),
		},
		{
			Id:            304,
			UserId:        opponentID,
			OpponentId:    &userID,
			OpponentName:  "我",
			GameType:      3,
			MyScore:       2,
			OpponentScore: 7,
			Status:        2,
			Result:        &lose,
			MatchTime:     time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC),
		},
	}

	for _, match := range matches {
		if err := matchModel.Create(match); err != nil {
			t.Fatalf("create match %d: %v", match.Id, err)
		}
	}

	total, myWins, oppWins, _, maxWinStreak, err := matchModel.GetH2HStatsByOpponent(userID, opponentID, "球友")
	if err != nil {
		t.Fatalf("get streak h2h stats: %v", err)
	}

	if total != 4 || myWins != 3 || oppWins != 1 {
		t.Fatalf("expected total=4 myWins=3 oppWins=1, got total=%d myWins=%d oppWins=%d", total, myWins, oppWins)
	}
	if maxWinStreak != 2 {
		t.Fatalf("expected longest h2h streak 2, got %d", maxWinStreak)
	}
}

func TestMatchModelOpponentListMergesRegisteredOpponentAcrossNicknameChanges(t *testing.T) {
	db := newMatchH2HTestDB(t)
	userModel := NewUserModel(db)
	matchModel := NewMatchModel(db)

	if err := userModel.Create(&User{Id: 1, Nickname: "我"}); err != nil {
		t.Fatalf("create user 1: %v", err)
	}
	if err := userModel.Create(&User{Id: 2, Nickname: "新昵称", Avatar: "https://img.example/new.png"}); err != nil {
		t.Fatalf("create user 2: %v", err)
	}

	userID := int64(1)
	opponentID := int64(2)
	win := 1

	if err := matchModel.Create(&Match{
		Id:            401,
		UserId:        userID,
		OpponentId:    &opponentID,
		OpponentName:  "旧昵称",
		GameType:      2,
		MyScore:       21,
		OpponentScore: 18,
		Status:        2,
		Result:        &win,
		MatchTime:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("create match 401: %v", err)
	}

	if err := matchModel.Create(&Match{
		Id:            402,
		UserId:        opponentID,
		OpponentId:    &userID,
		OpponentName:  "我",
		GameType:      2,
		MyScore:       21,
		OpponentScore: 19,
		Status:        2,
		Result:        &win,
		MatchTime:     time.Date(2026, 3, 2, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("create match 402: %v", err)
	}

	list, total, err := matchModel.ListOpponentsWithStats(userID, "", 0, 20)
	if err != nil {
		t.Fatalf("list opponents with stats: %v", err)
	}

	if total != 1 || len(list) != 1 {
		t.Fatalf("expected one merged opponent entry, got total=%d len=%d", total, len(list))
	}

	item := list[0]
	if item.OpponentId != opponentID {
		t.Fatalf("expected opponent id %d, got %d", opponentID, item.OpponentId)
	}
	if item.OpponentName != "新昵称" {
		t.Fatalf("expected latest nickname 新昵称, got %q", item.OpponentName)
	}
	if item.Avatar != "https://img.example/new.png" {
		t.Fatalf("expected latest avatar, got %q", item.Avatar)
	}
	if item.TotalMatches != 2 || item.Wins != 1 || item.Losses != 1 {
		t.Fatalf("expected merged stats 2 matches 1 win 1 loss, got %#v", item)
	}

	totalOpponents, totalWins, err := matchModel.GetOverallOpponentStats(userID)
	if err != nil {
		t.Fatalf("get overall opponent stats: %v", err)
	}
	if totalOpponents != 1 {
		t.Fatalf("expected total opponents 1 after merge, got %d", totalOpponents)
	}
	if totalWins != 1 {
		t.Fatalf("expected total wins 1, got %d", totalWins)
	}
}

func TestMatchModelListByOpponentNameReturnsAnonymousHistory(t *testing.T) {
	db := newMatchH2HTestDB(t)
	matchModel := NewMatchModel(db)

	win := 1
	lose := 2

	seed := []*Match{
		{
			Id:            501,
			UserId:        1,
			OpponentName:  "线下朋友",
			GameType:      2,
			MyScore:       21,
			OpponentScore: 19,
			Status:        2,
			Result:        &win,
			MatchTime:     time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
		},
		{
			Id:            502,
			UserId:        1,
			OpponentName:  "线下朋友",
			GameType:      2,
			MyScore:       17,
			OpponentScore: 21,
			Status:        2,
			Result:        &lose,
			MatchTime:     time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC),
		},
	}

	for _, match := range seed {
		if err := matchModel.Create(match); err != nil {
			t.Fatalf("create match %d: %v", match.Id, err)
		}
	}

	list, total, err := matchModel.ListByOpponentName(1, "线下朋友", 0, 0, 20, nil, nil)
	if err != nil {
		t.Fatalf("list by opponent name: %v", err)
	}

	if total != 2 || len(list) != 2 {
		t.Fatalf("expected 2 anonymous matches, got total=%d len=%d", total, len(list))
	}
	if list[0].Id != 502 || list[1].Id != 501 {
		t.Fatalf("expected matches ordered by time desc, got %#v", list)
	}
	if list[0].Result != 2 || list[1].Result != 1 {
		t.Fatalf("expected preserved anonymous results, got %#v", list)
	}
}

func TestMatchModelListByOpponentIdFiltersByDateRange(t *testing.T) {
	db := newMatchH2HTestDB(t)
	userModel := NewUserModel(db)
	matchModel := NewMatchModel(db)

	if err := userModel.Create(&User{Id: 1, Nickname: "我"}); err != nil {
		t.Fatalf("create user 1: %v", err)
	}
	if err := userModel.Create(&User{Id: 2, Nickname: "球友"}); err != nil {
		t.Fatalf("create user 2: %v", err)
	}

	userID := int64(1)
	opponentID := int64(2)
	win := 1
	for _, match := range []*Match{
		{Id: 601, UserId: userID, OpponentId: &opponentID, OpponentName: "球友", GameType: 3, MyScore: 5, OpponentScore: 3, Status: 2, Result: &win, MatchTime: time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)},
		{Id: 602, UserId: userID, OpponentId: &opponentID, OpponentName: "球友", GameType: 3, MyScore: 5, OpponentScore: 3, Status: 2, Result: &win, MatchTime: time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)},
		{Id: 603, UserId: userID, OpponentId: &opponentID, OpponentName: "球友", GameType: 3, MyScore: 5, OpponentScore: 3, Status: 2, Result: &win, MatchTime: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)},
	} {
		if err := matchModel.Create(match); err != nil {
			t.Fatalf("create match %d: %v", match.Id, err)
		}
	}

	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	list, total, err := matchModel.ListByOpponentId(userID, opponentID, 0, 0, 20, &start, &end)
	if err != nil {
		t.Fatalf("list by opponent id: %v", err)
	}

	if total != 1 || len(list) != 1 || list[0].Id != 602 {
		t.Fatalf("expected only April match, got total=%d list=%#v", total, list)
	}
}

func TestMatchModelListByOpponentNameFiltersByDateRange(t *testing.T) {
	db := newMatchH2HTestDB(t)
	matchModel := NewMatchModel(db)

	win := 1
	for _, match := range []*Match{
		{Id: 701, UserId: 1, OpponentName: "线下朋友", GameType: 2, MyScore: 21, OpponentScore: 18, Status: 2, Result: &win, MatchTime: time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)},
		{Id: 702, UserId: 1, OpponentName: "线下朋友", GameType: 2, MyScore: 21, OpponentScore: 18, Status: 2, Result: &win, MatchTime: time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)},
	} {
		if err := matchModel.Create(match); err != nil {
			t.Fatalf("create match %d: %v", match.Id, err)
		}
	}

	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	list, total, err := matchModel.ListByOpponentName(1, "线下朋友", 0, 0, 20, &start, &end)
	if err != nil {
		t.Fatalf("list by opponent name: %v", err)
	}

	if total != 1 || len(list) != 1 || list[0].Id != 702 {
		t.Fatalf("expected only April anonymous match, got total=%d list=%#v", total, list)
	}
}
