package match

import (
	"testing"

	"chasing_points/internal/model"
)

func nineBallTestAction(actionType string, actor, score int, winType string) *model.MatchAction {
	action := &model.MatchAction{ActionType: actionType, Actor: actor, ScoreChange: score}
	if winType != "" {
		extra := `{"win_type":"` + winType + `"}`
		action.ExtraData = &extra
	}
	return action
}

func TestBuildMatchLastActionFromActionNineBallDescriptions(t *testing.T) {
	opponentID := int64(2)
	refereeID := int64(3)
	match := &model.Match{Id: 1, UserId: 1, OpponentId: &opponentID, RefereeUserId: &refereeID, GameType: 2}

	cases := []struct {
		name   string
		viewer int64
		action *model.MatchAction
		want   string
	}{
		{"player1 normal win", 1, nineBallTestAction("win", 1, 4, "normal"), "我方普胜，得4分"},
		{"player2 sees player1 normal win", 2, nineBallTestAction("win", 1, 4, "normal"), "对手普胜，得4分"},
		{"referee sees player1 normal win", 3, nineBallTestAction("win", 1, 4, "normal"), "选手1普胜，得4分"},
		{"referee sees player2 small gold", 3, nineBallTestAction("win", 2, 7, "small_gold"), "选手2小金，得7分"},
		{"player1 big gold", 1, nineBallTestAction("win", 1, 10, "big_gold"), "我方大金，得10分"},
		{"legacy golden_break is small gold", 1, nineBallTestAction("win", 1, 7, "golden_break"), "我方小金，得7分"},
		{"legacy nine_on_break is big gold", 2, nineBallTestAction("win", 2, 10, "nine_on_break"), "我方大金，得10分"},
		{"unknown win type omits win method", 1, nineBallTestAction("win", 1, 5, "mystery"), "我方得5分"},
		{"missing win type omits win method", 1, nineBallTestAction("win", 1, 4, ""), "我方得4分"},
		{"player1 foul", 1, nineBallTestAction("foul", 1, 1, ""), "我方犯规，对手得1分"},
		{"player2 sees player1 foul", 2, nineBallTestAction("foul", 1, 1, ""), "对手犯规，我方得1分"},
		{"referee sees player1 foul", 3, nineBallTestAction("foul", 1, 1, ""), "选手1犯规，选手2得1分"},
		{"referee sees player2 foul", 3, nineBallTestAction("foul", 2, 1, ""), "选手2犯规，选手1得1分"},
	}
	for _, testCase := range cases {
		got := buildMatchLastActionFromAction(testCase.viewer, match, testCase.action)
		if got == nil || got.Description != testCase.want {
			t.Fatalf("%s: got %#v, want description %q", testCase.name, got, testCase.want)
		}
	}
}

func TestBuildMatchLastActionFromActionKeepsPoolWinDescription(t *testing.T) {
	opponentID := int64(2)
	match := &model.Match{Id: 2, UserId: 1, OpponentId: &opponentID, GameType: 3}

	got := buildMatchLastActionFromAction(1, match, nineBallTestAction("win", 1, 1, "normal"))
	if got == nil || got.Description != "我方结束本局" {
		t.Fatalf("pool win description = %#v, want 我方结束本局", got)
	}
}
