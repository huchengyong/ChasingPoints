package match

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestGetMatchDetailRoundsUseFixedPlayerPerspective(t *testing.T) {
	svcCtx, db := newScoreBreakdownDetailSvc(t)
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}, {Id: 3, Nickname: "裁判"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	opponentID, refereeID := int64(2), int64(3)
	result := 1
	if err := db.Create(&model.Match{
		Id: 601, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", RefereeUserId: &refereeID,
		GameType: 1, Status: 2, Result: &result, MyScore: 2, OpponentScore: 1,
		MatchTime: time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create match: %v", err)
	}
	// current_frame_started 列默认 true，已结束对局显式落为 false。
	if err := db.Model(&model.Match{}).Where("id = ?", 601).Update("current_frame_started", false).Error; err != nil {
		t.Fatalf("settle match frame state: %v", err)
	}
	winners := []int{1, 2, 1}
	rounds := make([]model.MatchRound, 0, len(winners))
	for index, winner := range winners {
		w := winner
		rounds = append(rounds, model.MatchRound{
			MatchId: 601, RoundNo: index + 1, MyScore: 60 + index, OpponentScore: 20 + index,
			Winner: &w, WinType: "normal",
		})
	}
	if err := db.Create(&rounds).Error; err != nil {
		t.Fatalf("seed rounds: %v", err)
	}
	if err := db.Create(&[]model.MatchAchievement{
		{MatchId: 601, Actor: 1, AchievementType: "break_clear", Count: 2},
		{MatchId: 601, Actor: 2, AchievementType: "break_clear", Count: 1},
		{MatchId: 601, Actor: 2, AchievementType: "continue_clear", Count: 1},
	}).Error; err != nil {
		t.Fatalf("seed achievements: %v", err)
	}

	player1Achievements := types.MatchAchievement{BreakClear: 2}
	player2Achievements := types.MatchAchievement{BreakClear: 1, ContinueClear: 1}
	cases := []struct {
		name     string
		viewer   int64
		mine     types.MatchAchievement
		opponent types.MatchAchievement
	}{
		{"player1", 1, player1Achievements, player2Achievements},
		{"player2", 2, player2Achievements, player1Achievements},
		{"referee", 3, player1Achievements, player2Achievements},
	}
	for _, testCase := range cases {
		detail := loadScoreBreakdownDetail(t, svcCtx, 601, testCase.viewer)
		if detail.Achievements != testCase.mine || detail.OpponentAchievements != testCase.opponent {
			t.Fatalf("%s achievements: mine=%+v opponent=%+v", testCase.name, detail.Achievements, detail.OpponentAchievements)
		}
		if len(detail.Rounds) != 3 || detail.TotalRounds != 3 {
			t.Fatalf("%s rounds: rounds=%+v total=%d current=%d", testCase.name, detail.Rounds, detail.TotalRounds, detail.CurrentRound)
		}
		wantFirst := types.RoundRecord{RoundNumber: 1, Player1Score: 60, Player2Score: 20, Winner: 1}
		wantSecond := types.RoundRecord{RoundNumber: 2, Player1Score: 61, Player2Score: 21, Winner: 2}
		if detail.Rounds[0] != wantFirst || detail.Rounds[1] != wantSecond {
			t.Fatalf("%s rounds must stay in player1/player2 positions: %+v", testCase.name, detail.Rounds)
		}
	}
}

func TestGetMatchDetailRoundProgressForOngoingPoolAndNineBall(t *testing.T) {
	svcCtx, db := newScoreBreakdownDetailSvc(t)
	seedScoreBreakdownUsers(t, db)
	opponentID := int64(2)
	if err := db.Create(&model.Match{
		Id: 602, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3,
		Status: 1, MyScore: 1, OpponentScore: 0,
		MatchTime: time.Date(2026, 11, 2, 20, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create pool match: %v", err)
	}
	winner := 1
	if err := db.Create(&model.MatchRound{MatchId: 602, RoundNo: 1, MyScore: 8, OpponentScore: 3, Winner: &winner, WinType: "normal"}).Error; err != nil {
		t.Fatalf("seed pool round: %v", err)
	}

	pool := loadScoreBreakdownDetail(t, svcCtx, 602, 1)
	if len(pool.Rounds) != 1 || pool.TotalRounds != 1 || pool.CurrentRound != 2 {
		t.Fatalf("ongoing pool rounds: rounds=%+v total=%d current=%d", pool.Rounds, pool.TotalRounds, pool.CurrentRound)
	}

	if err := db.Create(&model.Match{
		Id: 603, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 2,
		Status: 1, MyScore: 7, OpponentScore: 4,
		MatchTime: time.Date(2026, 11, 3, 20, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create nine-ball match: %v", err)
	}
	chasing := loadScoreBreakdownDetail(t, svcCtx, 603, 1)
	if len(chasing.Rounds) != 0 || chasing.TotalRounds != 0 || chasing.CurrentRound != 0 {
		t.Fatalf("nine-ball must not expose round records: rounds=%+v total=%d current=%d", chasing.Rounds, chasing.TotalRounds, chasing.CurrentRound)
	}
}
