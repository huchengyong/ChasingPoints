package ws

import (
	"testing"

	"chasing_points/internal/model"
)

// 约球比赛的 WS 同步快照必须与 HTTP 权威快照一致：
// 任一参赛方在合法结束点可单方结束（裁判绑定只放宽结束），并携带来源约球 ID。
func TestChallengeMatchSyncKeepsParticipantFinishAuthority(t *testing.T) {
	opponent, referee, challengeID := int64(2002), int64(3003), int64(77)
	match := &model.Match{
		Id: 88, UserId: 1001, OpponentId: &opponent, RefereeUserId: &referee,
		ChallengeId: &challengeID, GameType: 3, MatchFormat: "free", MyScore: 1,
		Status: 1, MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
		FinishState: model.FinishStateNone,
	}
	for _, userID := range []int64{1001, 2002} {
		data := buildMatchSyncDataForViewer(match, userID, 1, model.SnookerRoundState{}, nil)
		if !data.CanFinish {
			t.Fatalf("参赛者 %d 的 WS 同步快照必须保留单方结束权限: %+v", userID, data)
		}
		if data.CanScore || data.CanUndo {
			t.Fatalf("裁判绑定仍应限制记分/撤销: user=%d data=%+v", userID, data)
		}
		if data.ChallengeId != challengeID {
			t.Fatalf("WS 同步快照必须携带来源约球 ID: got=%d want=%d", data.ChallengeId, challengeID)
		}
	}

	// 非约球比赛不放宽：裁判绑定时参赛者仍不可结束，也不标记为约球比赛。
	noChallenge := *match
	noChallenge.ChallengeId = nil
	for _, userID := range []int64{1001, 2002} {
		data := buildMatchSyncDataForViewer(&noChallenge, userID, 1, model.SnookerRoundState{}, nil)
		if data.CanFinish {
			t.Fatalf("非约球裁判比赛不得放宽结束权限: user=%d", userID)
		}
		if data.ChallengeId != 0 {
			t.Fatalf("非约球比赛不得返回来源约球 ID: user=%d got=%d", userID, data.ChallengeId)
		}
	}
}

// WS 同步快照的合法结束权限必须与 HTTP 一致：灵活赛制未到结束点时不得放行（含裁判）。
func TestPoolMatchSyncFinishEligibilityMatchesHTTP(t *testing.T) {
	opponent, referee, challengeID := int64(2002), int64(3003), int64(77)
	for _, tc := range []struct {
		name    string
		format  string
		target  int
		myScore int
		userID  int64
		want    bool
	}{
		{name: "free_zero_player1", format: "free", userID: 1001, want: false},
		{name: "free_zero_player2", format: "free", userID: 2002, want: false},
		{name: "free_zero_referee", format: "free", userID: 3003, want: false},
		{name: "race_before_target", format: "race_to", target: 3, myScore: 1, userID: 1001, want: false},
		{name: "free_one_round", format: "free", myScore: 1, userID: 1001, want: true},
		{name: "race_target_reached", format: "race_to", target: 1, myScore: 1, userID: 2002, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			match := &model.Match{
				Id: 88, UserId: 1001, OpponentId: &opponent, RefereeUserId: &referee, ChallengeId: &challengeID,
				GameType: 3, MatchFormat: tc.format, TargetWins: tc.target, MyScore: tc.myScore,
				Status: 1, MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
				FinishState: model.FinishStateNone,
			}
			data := buildMatchSyncDataForViewer(match, tc.userID, int64(tc.myScore), model.SnookerRoundState{}, nil)
			if data.CanFinish != tc.want {
				t.Fatalf("viewer=%d can_finish=%v want=%v", tc.userID, data.CanFinish, tc.want)
			}
		})
	}
}
