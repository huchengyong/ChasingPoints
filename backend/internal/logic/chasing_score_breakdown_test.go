package logic

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

var chasingTestStart = time.Date(2026, 10, 27, 20, 0, 0, 0, UTC8Location)

func chasingWin(round, actor, points int, winType string, offset time.Duration) model.MatchAction {
	extra := `{"win_type":"` + winType + `"}`
	return model.MatchAction{RoundNo: round, ActionType: "win", Actor: actor, ScoreChange: points, ExtraData: &extra, CreatedAt: chasingTestStart.Add(offset)}
}

func chasingFoul(round, actor, points int, offset time.Duration) model.MatchAction {
	return model.MatchAction{RoundNo: round, ActionType: "foul", Actor: actor, ScoreChange: points, CreatedAt: chasingTestStart.Add(offset)}
}

func chasingScoreAction(round, actor, points int, offset time.Duration) model.MatchAction {
	return model.MatchAction{RoundNo: round, ActionType: "score", Actor: actor, ScoreChange: points, CreatedAt: chasingTestStart.Add(offset)}
}

func chasingRound(no, player1, player2, winner int, winType string) model.MatchRound {
	w := winner
	return model.MatchRound{RoundNo: no, MyScore: player1, OpponentScore: player2, Winner: &w, WinType: winType}
}

func chasingMatch(myScore, opponentScore int) *model.Match {
	return &model.Match{Id: 1, UserId: 1, GameType: 2, Status: 2, MyScore: myScore, OpponentScore: opponentScore}
}

func chasingEvent(seq int, kind string, scorer, points, player1, player2 int, offset time.Duration) ChasingScoreEvent {
	return ChasingScoreEvent{Seq: seq, Kind: kind, Scorer: scorer, Points: points, Player1Score: player1, Player2Score: player2, CreatedAt: chasingTestStart.Add(offset)}
}

func assertChasingSide(t *testing.T, name string, got, want ChasingScoreSide) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: got %+v, want %+v", name, got, want)
	}
}

func assertChasingEvents(t *testing.T, got, want []ChasingScoreEvent) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("events: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Seq != want[i].Seq || got[i].Kind != want[i].Kind || got[i].Scorer != want[i].Scorer ||
			got[i].Points != want[i].Points || got[i].Player1Score != want[i].Player1Score ||
			got[i].Player2Score != want[i].Player2Score || !got[i].CreatedAt.Equal(want[i].CreatedAt) {
			t.Fatalf("event %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// chasingCompleteFixture：选手1 普胜 2、小金 1 且犯规 3 次，选手2 大金 1 且犯规 1 次，最终 16:13。
func chasingCompleteFixture() (*model.Match, []model.MatchRound, []model.MatchAction) {
	actions := []model.MatchAction{
		chasingWin(1, 1, 4, "normal", 0),
		chasingFoul(2, 2, 1, time.Minute),
		chasingWin(2, 1, 7, "small_gold", 2*time.Minute),
		chasingFoul(3, 1, 1, 3*time.Minute),
		chasingFoul(3, 1, 1, 4*time.Minute),
		chasingWin(3, 2, 10, "big_gold", 5*time.Minute),
		chasingFoul(4, 1, 1, 6*time.Minute),
		chasingWin(4, 1, 4, "normal", 7*time.Minute),
	}
	rounds := []model.MatchRound{
		chasingRound(1, 4, 0, 1, "normal"),
		chasingRound(2, 12, 0, 1, "small_gold"),
		chasingRound(3, 12, 12, 2, "big_gold"),
		chasingRound(4, 16, 13, 1, "normal"),
	}
	return chasingMatch(16, 13), rounds, actions
}

func TestChasingScoreBreakdownCompleteMatch(t *testing.T) {
	match, rounds, actions := chasingCompleteFixture()
	got := BuildChasingScoreBreakdown(match, rounds, actions)
	if got == nil || !got.Available || got.EventsTruncated {
		t.Fatalf("unexpected shape: %+v", got)
	}
	assertChasingSide(t, "player1", got.Player1, ChasingScoreSide{Score: 16, NormalWins: 2, SmallGolds: 1, FoulCount: 3, FoulPointsGiven: 3})
	assertChasingSide(t, "player2", got.Player2, ChasingScoreSide{Score: 13, BigGolds: 1, FoulCount: 1, FoulPointsGiven: 1})
	assertChasingEvents(t, got.Events, []ChasingScoreEvent{
		chasingEvent(1, "normal", 1, 4, 4, 0, 0),
		chasingEvent(2, "foul", 1, 1, 5, 0, time.Minute),
		chasingEvent(3, "small_gold", 1, 7, 12, 0, 2*time.Minute),
		chasingEvent(4, "foul", 2, 1, 12, 1, 3*time.Minute),
		chasingEvent(5, "foul", 2, 1, 12, 2, 4*time.Minute),
		chasingEvent(6, "big_gold", 2, 10, 12, 12, 5*time.Minute),
		chasingEvent(7, "foul", 2, 1, 12, 13, 6*time.Minute),
		chasingEvent(8, "normal", 1, 4, 16, 13, 7*time.Minute),
	})
}

func TestChasingScoreBreakdownIncludesFoulsAfterLastWin(t *testing.T) {
	match, rounds, actions := chasingCompleteFixture()
	actions = append(actions,
		chasingFoul(5, 2, 1, 8*time.Minute),
		chasingFoul(5, 2, 1, 9*time.Minute),
	)
	match.MyScore = 18
	got := BuildChasingScoreBreakdown(match, rounds, actions)
	if !got.Available {
		t.Fatalf("fouls after the last win must reconcile: %+v", got)
	}
	assertChasingSide(t, "player1", got.Player1, ChasingScoreSide{Score: 18, NormalWins: 2, SmallGolds: 1, FoulCount: 3, FoulPointsGiven: 3})
	assertChasingSide(t, "player2", got.Player2, ChasingScoreSide{Score: 13, BigGolds: 1, FoulCount: 3, FoulPointsGiven: 3})
	if len(got.Events) != 10 || got.Events[8].Kind != "foul" || got.Events[9].Kind != "foul" || got.Events[9].Player1Score != 18 {
		t.Fatalf("tail fouls must appear in the stream: %+v", got.Events)
	}
}

func TestChasingScoreBreakdownEventOrderAndRunningTotals(t *testing.T) {
	match := chasingMatch(5, 10)
	got := BuildChasingScoreBreakdown(match, nil, []model.MatchAction{
		chasingFoul(1, 2, 1, 0),
		chasingWin(2, 1, 4, "normal", time.Minute),
		chasingWin(3, 2, 10, "big_gold", 2*time.Minute),
	})
	assertChasingEvents(t, got.Events, []ChasingScoreEvent{
		chasingEvent(1, "foul", 1, 1, 1, 0, 0),
		chasingEvent(2, "normal", 1, 4, 5, 0, time.Minute),
		chasingEvent(3, "big_gold", 2, 10, 5, 10, 2*time.Minute),
	})
}

func TestChasingScoreBreakdownTable(t *testing.T) {
	undoneSmallGold := chasingWin(1, 1, 7, "small_gold", time.Minute)
	undoneSmallGold.IsUndone = 1

	tests := []struct {
		name      string
		match     *model.Match
		rounds    []model.MatchRound
		actions   []model.MatchAction
		available bool
		want1     ChasingScoreSide
		want2     ChasingScoreSide
		wantKinds []string
	}{
		{
			name:      "旧版小金代码",
			match:     chasingMatch(7, 0),
			actions:   []model.MatchAction{chasingWin(1, 1, 7, "golden_break", 0)},
			available: true,
			want1:     ChasingScoreSide{Score: 7, SmallGolds: 1},
			wantKinds: []string{"small_gold"},
		},
		{
			name:      "旧版大金代码",
			match:     chasingMatch(0, 10),
			actions:   []model.MatchAction{chasingWin(1, 2, 10, "nine_on_break", 0)},
			available: true,
			want2:     ChasingScoreSide{Score: 10, BigGolds: 1},
			wantKinds: []string{"big_gold"},
		},
		{
			name:      "已撤销的小金",
			match:     chasingMatch(4, 0),
			actions:   []model.MatchAction{undoneSmallGold, chasingWin(1, 1, 4, "normal", 2*time.Minute)},
			available: true,
			want1:     ChasingScoreSide{Score: 4, NormalWins: 1},
			wantKinds: []string{"normal"},
		},
		{
			name:      "早期 score 动作计入其他得分",
			match:     chasingMatch(4, 0),
			actions:   []model.MatchAction{chasingScoreAction(1, 1, 4, 0)},
			available: true,
			want1:     ChasingScoreSide{Score: 4, OtherPoints: 4},
			wantKinds: []string{"other"},
		},
		{
			name:      "未知胜法计入其他得分",
			match:     chasingMatch(0, 3),
			actions:   []model.MatchAction{chasingWin(1, 2, 3, "clearance", 0)},
			available: true,
			want2:     ChasingScoreSide{Score: 3, OtherPoints: 3},
			wantKinds: []string{"other"},
		},
		{
			name:  "ExtraData 缺失回退到局记录",
			match: chasingMatch(0, 10),
			rounds: []model.MatchRound{
				chasingRound(1, 0, 10, 2, "big_gold"),
			},
			actions:   []model.MatchAction{{RoundNo: 1, ActionType: "win", Actor: 2, ScoreChange: 10, CreatedAt: chasingTestStart}},
			available: true,
			want2:     ChasingScoreSide{Score: 10, BigGolds: 1},
			wantKinds: []string{"big_gold"},
		},
		{
			name:      "分值为 0 的犯规只计次数",
			match:     chasingMatch(0, 0),
			actions:   []model.MatchAction{chasingFoul(1, 2, 0, 0)},
			available: true,
			want2:     ChasingScoreSide{FoulCount: 1},
		},
		{
			name:  "其他动作类型忽略",
			match: chasingMatch(0, 0),
			actions: []model.MatchAction{
				{RoundNo: 1, ActionType: "round_start", CreatedAt: chasingTestStart},
				{RoundNo: 1, ActionType: "match_end", CreatedAt: chasingTestStart.Add(time.Minute)},
			},
			available: true,
		},
		{
			name:    "对账失败只保留总分",
			match:   chasingMatch(5, 0),
			actions: []model.MatchAction{chasingWin(1, 1, 4, "normal", 0)},
			want1:   ChasingScoreSide{Score: 5},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := BuildChasingScoreBreakdown(test.match, test.rounds, test.actions)
			if got == nil {
				t.Fatal("nine-ball breakdown must not be nil")
			}
			if got.Available != test.available {
				t.Fatalf("available: got %v, want %v", got.Available, test.available)
			}
			assertChasingSide(t, "player1", got.Player1, test.want1)
			assertChasingSide(t, "player2", got.Player2, test.want2)
			if len(got.Events) != len(test.wantKinds) {
				t.Fatalf("events: got %d, want %d", len(got.Events), len(test.wantKinds))
			}
			for i, kind := range test.wantKinds {
				if got.Events[i].Kind != kind {
					t.Fatalf("event %d kind: got %q, want %q", i, got.Events[i].Kind, kind)
				}
			}
		})
	}
}

func TestChasingScoreBreakdownTruncatesToRecentEvents(t *testing.T) {
	actions := make([]model.MatchAction, 0, 260)
	for i := 0; i < 260; i++ {
		actions = append(actions, chasingWin(i/10+1, 1, 4, "normal", time.Duration(i)*time.Second))
	}
	got := BuildChasingScoreBreakdown(chasingMatch(1040, 0), nil, actions)
	if !got.Available || !got.EventsTruncated {
		t.Fatalf("truncation must be flagged: %+v", got)
	}
	if len(got.Events) != MaxChasingScoreEvents {
		t.Fatalf("events: got %d, want %d", len(got.Events), MaxChasingScoreEvents)
	}
	if first, last := got.Events[0], got.Events[len(got.Events)-1]; first.Seq != 61 || last.Seq != 260 || last.Player1Score != 1040 {
		t.Fatalf("must keep the most recent events with their original seq: first=%+v last=%+v", first, last)
	}
	assertChasingSide(t, "player1", got.Player1, ChasingScoreSide{Score: 1040, NormalWins: 260})
}

func TestChasingScoreBreakdownPayloadOmitsEvents(t *testing.T) {
	built := &ChasingScoreBreakdown{
		Available:       true,
		Player1:         ChasingScoreSide{Score: 16, NormalWins: 2, SmallGolds: 1, FoulCount: 3, FoulPointsGiven: 3},
		Player2:         ChasingScoreSide{Score: 13, BigGolds: 1, FoulCount: 1, FoulPointsGiven: 1},
		Events:          []ChasingScoreEvent{chasingEvent(1, "normal", 1, 4, 4, 0, 0)},
		EventsTruncated: true,
	}
	got := ChasingScoreBreakdownPayload(built, false)
	if got == nil || !got.Available {
		t.Fatalf("unexpected payload: %+v", got)
	}
	if got.Player1 != (types.ChasingScoreSide{Score: 16, NormalWins: 2, SmallGolds: 1, FoulCount: 3, FoulPointsGiven: 3}) ||
		got.Player2 != (types.ChasingScoreSide{Score: 13, BigGolds: 1, FoulCount: 1, FoulPointsGiven: 1}) {
		t.Fatalf("sides: %+v / %+v", got.Player1, got.Player2)
	}
	if got.Events == nil || len(got.Events) != 0 || got.EventsTruncated {
		t.Fatalf("without events: %+v", got)
	}
	if ChasingScoreBreakdownPayload(nil, true) != nil {
		t.Fatal("nil breakdown must stay nil")
	}
}

func TestChasingScoreBreakdownPayloadFormatsEvents(t *testing.T) {
	built := &ChasingScoreBreakdown{
		Available: true,
		Events: []ChasingScoreEvent{
			chasingEvent(1, "foul", 1, 1, 1, 0, 0),
			{Seq: 2, Kind: "normal", Scorer: 1, Points: 4, Player1Score: 5, Player2Score: 0},
		},
	}
	got := ChasingScoreBreakdownPayload(built, true)
	if len(got.Events) != 2 {
		t.Fatalf("events: %+v", got.Events)
	}
	if got.Events[0] != (types.ChasingScoreEvent{Seq: 1, Kind: "foul", Scorer: 1, Points: 1, Player1Score: 1, Player2Score: 0, CreatedAt: "2026-10-27 20:00:00"}) {
		t.Fatalf("event 0: %+v", got.Events[0])
	}
	if got.Events[1].CreatedAt != "" {
		t.Fatalf("zero time must be empty: %+v", got.Events[1])
	}
}

func TestUnavailableChasingScoreBreakdown(t *testing.T) {
	if got := UnavailableChasingScoreBreakdown(nil); got != nil {
		t.Fatalf("nil match: %+v", got)
	}
	if got := UnavailableChasingScoreBreakdown(&model.Match{GameType: 3}); got != nil {
		t.Fatalf("other game type: %+v", got)
	}

	got := UnavailableChasingScoreBreakdown(chasingMatch(21, 15))
	if got == nil || got.Available {
		t.Fatalf("must be unavailable: %+v", got)
	}
	assertChasingSide(t, "player1", got.Player1, ChasingScoreSide{Score: 21})
	assertChasingSide(t, "player2", got.Player2, ChasingScoreSide{Score: 15})
	if got.Events == nil || len(got.Events) != 0 {
		t.Fatalf("events must be an empty non-nil slice: %+v", got.Events)
	}

	// 对账失败分支复用同一结构。
	want := UnavailableChasingScoreBreakdown(chasingMatch(5, 0))
	reconciled := BuildChasingScoreBreakdown(chasingMatch(5, 0), nil, []model.MatchAction{chasingWin(1, 1, 4, "normal", 0)})
	if reconciled.Available != want.Available || reconciled.Player1 != want.Player1 ||
		reconciled.Player2 != want.Player2 || reconciled.Events == nil || len(reconciled.Events) != 0 {
		t.Fatalf("reconcile failure must reuse unavailable breakdown: got=%+v want=%+v", reconciled, want)
	}
}

func TestChasingScoreBreakdownSkipsOtherGameTypes(t *testing.T) {
	if got := BuildChasingScoreBreakdown(nil, nil, nil); got != nil {
		t.Fatalf("nil match: %+v", got)
	}
	for _, gameType := range []int{1, 3, 4} {
		match := &model.Match{GameType: gameType}
		if got := BuildChasingScoreBreakdown(match, nil, nil); got != nil {
			t.Fatalf("game type %d: %+v", gameType, got)
		}
	}
}
