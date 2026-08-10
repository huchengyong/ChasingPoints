package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMatchCoreFlowTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&User{}, &Friend{}, &Match{}); err != nil {
		t.Fatalf("prepare match schema: %v", err)
	}
	return db
}

func TestMatchDefaultsUseRankedPublicForLegacyRecords(t *testing.T) {
	db := newMatchCoreFlowTestDB(t)
	match := Match{Id: 1, UserId: 10, OpponentName: "对手", GameType: 3, Status: 2}
	if err := db.Create(&match).Error; err != nil {
		t.Fatalf("create legacy match: %v", err)
	}

	var stored Match
	if err := db.First(&stored, match.Id).Error; err != nil {
		t.Fatalf("load legacy match: %v", err)
	}
	gotMode := NormalizeMatchMode(stored.MatchMode)
	if gotMode != MatchModeRanked {
		t.Fatalf("expected ranked default, got %q", gotMode)
	}
	gotVisibility := NormalizeMatchVisibility(stored.Visibility, gotMode)
	if gotVisibility != MatchVisibilityPublic {
		t.Fatalf("expected public default, got %q", gotVisibility)
	}
	if stored.FinishConfirmationRequired {
		t.Fatal("legacy match should default to immediate finish compatibility")
	}
	if stored.SnookerRulesVersion != SnookerRulesVersionLegacy || stored.BestOfFrames != 0 || stored.StartingActor != 0 {
		t.Fatalf("unexpected legacy snooker defaults: version=%d bestOf=%d starter=%d", stored.SnookerRulesVersion, stored.BestOfFrames, stored.StartingActor)
	}
}

func TestMatchPersistsSnookerRulesV2Format(t *testing.T) {
	db := newMatchCoreFlowTestDB(t)
	match := Match{
		Id:                  2,
		UserId:              10,
		OpponentName:        "对手",
		GameType:            1,
		SnookerRulesVersion: SnookerRulesVersionWPBSA,
		BestOfFrames:        7,
		StartingActor:       2,
		Status:              1,
	}
	if err := db.Create(&match).Error; err != nil {
		t.Fatalf("create snooker v2 match: %v", err)
	}

	var stored Match
	if err := db.First(&stored, match.Id).Error; err != nil {
		t.Fatalf("load snooker v2 match: %v", err)
	}
	if stored.SnookerRulesVersion != SnookerRulesVersionWPBSA || stored.BestOfFrames != 7 || stored.StartingActor != 2 {
		t.Fatalf("unexpected snooker v2 format: version=%d bestOf=%d starter=%d", stored.SnookerRulesVersion, stored.BestOfFrames, stored.StartingActor)
	}
}

func TestListPublicMatchesHidesPrivateMatchesExceptParticipantFriendView(t *testing.T) {
	db := newMatchCoreFlowTestDB(t)
	for _, user := range []User{{Id: 1, Nickname: "我"}, {Id: 2, Nickname: "好友"}, {Id: 3, Nickname: "对手"}} {
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	if err := db.Create(&Friend{UserId: 1, FriendId: 2, Status: 1}).Error; err != nil {
		t.Fatalf("create friend: %v", err)
	}
	opponentID := int64(3)
	rows := []Match{
		{Id: 11, UserId: 2, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, Status: 1, MatchMode: MatchModePractice, Visibility: MatchVisibilityPrivate},
		{Id: 12, UserId: 2, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, Status: 1, MatchMode: MatchModePractice, Visibility: MatchVisibilityPublic},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create match %d: %v", row.Id, err)
		}
	}
	model := NewMatchModel(db)
	hall, total, err := model.ListPublicMatches(PublicMatchListOptions{Scope: "hall", Status: 1, Limit: 20})
	if err != nil {
		t.Fatalf("list hall: %v", err)
	}
	if total != 1 || len(hall) != 1 || hall[0].Id != 12 {
		t.Fatalf("expected only public match in hall, total=%d rows=%#v", total, hall)
	}

	friends, total, err := model.ListPublicMatches(PublicMatchListOptions{Scope: "friends", ViewerUserId: 2, Status: 1, Limit: 20})
	if err != nil {
		t.Fatalf("list participant friends: %v", err)
	}
	if total != 2 || len(friends) != 2 {
		t.Fatalf("expected participant to see both private and public matches, total=%d rows=%#v", total, friends)
	}

	friendView, total, err := model.ListPublicMatches(PublicMatchListOptions{Scope: "friends", ViewerUserId: 1, Status: 1, Limit: 20})
	if err != nil {
		t.Fatalf("list unrelated friend view: %v", err)
	}
	if total != 1 || len(friendView) != 1 || friendView[0].Id != 12 {
		t.Fatalf("private practice match should stay hidden from unrelated friend, total=%d rows=%#v", total, friendView)
	}
}

func TestListOngoingMatchesHidesPrivateMatchesFromPublicHall(t *testing.T) {
	db := newMatchCoreFlowTestDB(t)
	opponentID := int64(2)
	rows := []Match{
		{Id: 21, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, Status: 1, MatchMode: MatchModePractice, Visibility: MatchVisibilityPrivate},
		{Id: 22, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, Status: 1, MatchMode: MatchModePractice, Visibility: MatchVisibilityPublic},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create match %d: %v", row.Id, err)
		}
	}

	list, total, err := NewMatchModel(db).ListOngoingMatches(0, 20)
	if err != nil {
		t.Fatalf("list ongoing matches: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].Id != 22 {
		t.Fatalf("expected only public ongoing match, total=%d rows=%#v", total, list)
	}
}

func TestCompetitiveUserStatsExcludePracticeMatches(t *testing.T) {
	db := newMatchCoreFlowTestDB(t)
	opponentID := int64(2)
	win := 1
	matchTime := time.Now()
	for _, match := range []Match{
		{Id: 31, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: MatchModeRanked, Visibility: MatchVisibilityPublic, Status: 2, Result: &win, MatchTime: matchTime},
		{Id: 32, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: MatchModePractice, Visibility: MatchVisibilityPrivate, Status: 2, Result: &win, MatchTime: matchTime.Add(time.Minute)},
	} {
		if err := db.Create(&match).Error; err != nil {
			t.Fatalf("create match %d: %v", match.Id, err)
		}
	}

	stats, err := NewMatchModel(db).GetUserStats(1)
	if err != nil {
		t.Fatalf("get user stats: %v", err)
	}
	if stats.TotalMatches != 1 || stats.Wins != 1 || stats.Losses != 0 {
		t.Fatalf("practice match leaked into competitive stats: %+v", stats)
	}
}

func TestRankingReplayExcludesPracticeMatches(t *testing.T) {
	db := newMatchCoreFlowTestDB(t)
	if err := db.AutoMigrate(&MatchRound{}); err != nil {
		t.Fatalf("prepare match round schema: %v", err)
	}
	opponentID := int64(2)
	win := 1
	for _, match := range []Match{
		{Id: 41, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: MatchModeRanked, Visibility: MatchVisibilityPublic, Status: 2, Result: &win, MatchTime: time.Now()},
		{Id: 42, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: MatchModePractice, Visibility: MatchVisibilityPrivate, Status: 2, Result: &win, MatchTime: time.Now()},
	} {
		if err := db.Create(&match).Error; err != nil {
			t.Fatalf("create match %d: %v", match.Id, err)
		}
	}
	winner := 1
	if err := db.Create(&MatchRound{MatchId: 41, RoundNo: 1, Winner: &winner, WinType: "normal"}).Error; err != nil {
		t.Fatalf("create completed round: %v", err)
	}

	matches, err := NewMatchModel(db).ListCompletedForRankingReplay()
	if err != nil {
		t.Fatalf("list ranking replay matches: %v", err)
	}
	if len(matches) != 1 || matches[0].Id != 41 {
		t.Fatalf("practice match leaked into ranking replay: %#v", matches)
	}
}
