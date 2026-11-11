package matchtext

import (
	"testing"

	"chasing_points/internal/model"
)

func actionWithWinType(actionType string, actor, score int, winType string) *model.MatchAction {
	action := &model.MatchAction{ActionType: actionType, Actor: actor, ScoreChange: score}
	if winType != "" {
		extra := `{"win_type":"` + winType + `"}`
		action.ExtraData = &extra
	}
	return action
}

func TestLastActionDescriptionByViewer(t *testing.T) {
	cases := []struct {
		name     string
		viewer   string
		gameType int
		action   *model.MatchAction
		want     string
	}{
		{"player1 round start", "player1", 3, &model.MatchAction{ActionType: "round_start", RoundNo: 3}, "开始第3局"},
		{"player2 round start", "player2", 3, &model.MatchAction{ActionType: "round_start", RoundNo: 3}, "开始第3局"},
		{"referee round start", "referee", 1, &model.MatchAction{ActionType: "round_start", RoundNo: 2}, "开始第2局"},
		{"round start without round number", "player1", 3, &model.MatchAction{ActionType: "round_start"}, "开始新一局"},
		{"player1 match end", "player1", 3, &model.MatchAction{ActionType: "match_end"}, "对局结束"},
		{"player2 match end", "player2", 3, &model.MatchAction{ActionType: "match_end"}, "对局结束"},
		{"referee match end", "referee", 1, &model.MatchAction{ActionType: "match_end"}, "对局结束"},
		{"player1 pool normal win", "player1", 3, actionWithWinType("win", 1, 1, "normal"), "我方赢下本局"},
		{"player2 sees player1 pool win", "player2", 3, actionWithWinType("win", 1, 1, ""), "对手赢下本局"},
		{"player2 pool break clear", "player2", 3, actionWithWinType("win", 2, 1, "break_clear"), "我方炸清，赢下本局"},
		{"referee pool continue clear", "referee", 3, actionWithWinType("win", 2, 1, "continue_clear"), "选手2接清，赢下本局"},
		{"referee pool small gold", "referee", 4, actionWithWinType("win", 1, 1, "small_gold"), "选手1小金，赢下本局"},
		{"player1 pool big gold", "player1", 4, actionWithWinType("win", 1, 1, "big_gold"), "我方大金，赢下本局"},
		{"player1 legacy foul", "player1", 1, actionWithWinType("foul", 1, 4, ""), "我方犯规，对手得4分"},
		{"player2 sees player1 legacy foul", "player2", 3, actionWithWinType("foul", 1, 4, ""), "对手犯规，我方得4分"},
		{"referee sees player2 legacy foul", "referee", 3, actionWithWinType("foul", 2, 4, ""), "选手2犯规，选手1得4分"},
		{"nine ball win unchanged", "player1", 2, actionWithWinType("win", 1, 4, "small_gold"), "我方小金，得4分"},
		{"nine ball foul unchanged", "player2", 2, actionWithWinType("foul", 1, 1, ""), "对手犯规，我方得1分"},
		{"unknown action stays empty", "player1", 3, &model.MatchAction{ActionType: "undo"}, ""},
	}
	for _, testCase := range cases {
		if got := LastActionDescription(testCase.viewer, testCase.gameType, testCase.action); got != testCase.want {
			t.Fatalf("%s: got %q, want %q", testCase.name, got, testCase.want)
		}
	}
}

func TestFoulDescriptionByViewer(t *testing.T) {
	cases := []struct {
		name   string
		viewer string
		actor  int
		want   string
	}{
		{"player1 fouls", "player1", 1, "我方犯规，对手得4分"},
		{"player2 sees player1 foul", "player2", 1, "对手犯规，我方得4分"},
		{"referee sees player2 foul", "referee", 2, "选手2犯规，选手1得4分"},
		{"unknown viewer uses players", "", 1, "选手1犯规，选手2得4分"},
	}
	for _, testCase := range cases {
		action := &model.MatchAction{ActionType: model.MatchActionTypeSnookerStroke, Actor: testCase.actor, ScoreChange: 4}
		if got := FoulDescription(testCase.viewer, action); got != testCase.want {
			t.Fatalf("%s: got %q, want %q", testCase.name, got, testCase.want)
		}
	}
}
