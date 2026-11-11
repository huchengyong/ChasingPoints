package ws

import (
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildMatchSyncLastActionNineBallDescriptions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Match{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare last action schema: %v", err)
	}
	svcCtx := &svc.ServiceContext{DB: db, MatchModel: model.NewMatchModel(db)}

	opponentID := int64(2)
	refereeID := int64(3)
	matches := map[int]*model.Match{
		2: {Id: 701, UserId: 1, OpponentId: &opponentID, RefereeUserId: &refereeID, GameType: 2, MatchMode: model.MatchModePractice, Status: 1},
		3: {Id: 702, UserId: 1, OpponentId: &opponentID, RefereeUserId: &refereeID, GameType: 3, MatchMode: model.MatchModePractice, Status: 1},
	}
	for _, match := range matches {
		if err := db.Create(match).Error; err != nil {
			t.Fatalf("create match: %v", err)
		}
	}

	cases := []struct {
		name     string
		gameType int
		viewer   int64
		action   model.MatchAction
		want     string
	}{
		{"player1 normal win", 2, 1, hubNineBallAction("win", 1, 4, "normal"), "我方普胜，得4分"},
		{"player2 sees player1 normal win", 2, 2, hubNineBallAction("win", 1, 4, "normal"), "对手普胜，得4分"},
		{"referee sees player1 normal win", 2, 3, hubNineBallAction("win", 1, 4, "normal"), "选手1普胜，得4分"},
		{"referee sees player2 small gold", 2, 3, hubNineBallAction("win", 2, 7, "small_gold"), "选手2小金，得7分"},
		{"player1 big gold", 2, 1, hubNineBallAction("win", 1, 10, "big_gold"), "我方大金，得10分"},
		{"legacy golden_break is small gold", 2, 1, hubNineBallAction("win", 1, 7, "golden_break"), "我方小金，得7分"},
		{"legacy nine_on_break is big gold", 2, 2, hubNineBallAction("win", 2, 10, "nine_on_break"), "我方大金，得10分"},
		{"unknown win type omits win method", 2, 1, hubNineBallAction("win", 1, 5, "mystery"), "我方得5分"},
		{"missing win type omits win method", 2, 1, hubNineBallAction("win", 1, 4, ""), "我方得4分"},
		{"player1 foul", 2, 1, hubNineBallAction("foul", 1, 1, ""), "我方犯规，对手得1分"},
		{"player2 sees player1 foul", 2, 2, hubNineBallAction("foul", 1, 1, ""), "对手犯规，我方得1分"},
		{"referee sees player1 foul", 2, 3, hubNineBallAction("foul", 1, 1, ""), "选手1犯规，选手2得1分"},
		{"referee sees player2 foul", 2, 3, hubNineBallAction("foul", 2, 1, ""), "选手2犯规，选手1得1分"},
		{"pool win uses round winner text", 3, 1, hubNineBallAction("win", 1, 1, "normal"), "我方赢下本局"},
		{"pool break clear shows win method", 3, 1, hubNineBallAction("win", 1, 1, "break_clear"), "我方炸清，赢下本局"},
		{"pool round start shows round number", 3, 1, hubNineBallAction("round_start", 0, 0, ""), "开始新一局"},
	}
	for _, testCase := range cases {
		match := matches[testCase.gameType]
		action := testCase.action
		action.MatchId = match.Id
		if err := db.Create(&action).Error; err != nil {
			t.Fatalf("%s: create action: %v", testCase.name, err)
		}
		got := buildMatchSyncLastAction(svcCtx, match, testCase.viewer)
		if got == nil || got.Description != testCase.want {
			t.Fatalf("%s: got %#v, want description %q", testCase.name, got, testCase.want)
		}
	}
}

func hubNineBallAction(actionType string, actor, score int, winType string) model.MatchAction {
	action := model.MatchAction{ActionType: actionType, Actor: actor, ScoreChange: score}
	if winType != "" {
		extra := `{"win_type":"` + winType + `"}`
		action.ExtraData = &extra
	}
	return action
}
