package ranking

import (
	"math"
	"testing"
)

func placed(rating int) PlayerState {
	return PlayerState{Rating: rating, RatedMatches: PlacementMatches}
}

func TestRaceWinProbabilityMatchesClosedForms(t *testing.T) {
	if got := RaceWinProbability(0.7, 1); math.Abs(got-0.7) > 1e-12 {
		t.Fatalf("race to 1 must equal rack probability, got %v", got)
	}
	// 抢 2：p² + 2p²(1−p)
	p := 0.6
	if got, want := RaceWinProbability(p, 2), p*p+2*p*p*(1-p); math.Abs(got-want) > 1e-12 {
		t.Fatalf("race to 2 = %v, want %v", got, want)
	}
	for _, n := range []int{1, 3, 5, 9} {
		if got := RaceWinProbability(0.37, n) + RaceWinProbability(0.63, n); math.Abs(got-1) > 1e-12 {
			t.Fatalf("race to %d must be complementary, sum=%v", n, got)
		}
	}
}

func TestRaceLength(t *testing.T) {
	cases := []struct {
		name           string
		target, r1, r2 int
		want           int
	}{
		{name: "抢5取目标", target: 5, r1: 5, r2: 0, want: 5},
		{name: "抢5认输时仍取目标", target: 5, r1: 2, r2: 1, want: 5},
		{name: "自由局取较大赢局数", target: 0, r1: 6, r2: 3, want: 6},
		{name: "追分负方赢局更多", target: 0, r1: 3, r2: 5, want: 5},
		{name: "没有完成局按1", target: 0, r1: 0, r2: 0, want: 1},
		{name: "长自由局封顶9", target: 0, r1: 15, r2: 12, want: 9},
		{name: "长赛制封顶9", target: 11, r1: 11, r2: 4, want: 9},
	}
	for _, tc := range cases {
		if got := RaceLength(tc.target, tc.r1, tc.r2); got != tc.want {
			t.Fatalf("%s: got %d want %d", tc.name, got, tc.want)
		}
	}
}

func TestSettleUsesExpectationAndRaceLength(t *testing.T) {
	cases := []struct {
		name         string
		gap          int // 对手分 − 本人分
		n            int
		won          bool
		wantMine     int
		wantOpponent int
	}{
		{name: "同分抢5胜", gap: 0, n: 5, won: true, wantMine: 45, wantOpponent: -45},
		{name: "同分抢5负", gap: 0, n: 5, won: false, wantMine: -45, wantOpponent: 45},
		{name: "以下克上抢5", gap: 500, n: 5, won: true, wantMine: 68, wantOpponent: -68},
		{name: "输给强者抢5少扣", gap: 500, n: 5, won: false, wantMine: -21, wantOpponent: 21},
		{name: "赢弱者抢5少加", gap: -500, n: 5, won: true, wantMine: 21, wantOpponent: -21},
		{name: "同分抢1", gap: 0, n: 1, won: true, wantMine: 20, wantOpponent: -20},
		{name: "同分抢9", gap: 0, n: 9, won: true, wantMine: 60, wantOpponent: -60},
		{name: "悬殊对局胜方至少加1", gap: -5000, n: 9, won: true, wantMine: 1, wantOpponent: -1},
	}
	for _, tc := range cases {
		mine, opponent := Settle(Input{
			Player1:    placed(6000),
			Player2:    placed(6000 + tc.gap),
			Player1Won: tc.won,
			RaceLength: tc.n,
		})
		if mine.Change != tc.wantMine || opponent.Change != tc.wantOpponent {
			t.Fatalf("%s: got %+d/%+d want %+d/%+d", tc.name, mine.Change, opponent.Change, tc.wantMine, tc.wantOpponent)
		}
		if mine.Standard != mine.Change || opponent.Standard != opponent.Change {
			t.Fatalf("%s: unadjusted match must only contain the standard result: %+v %+v", tc.name, mine, opponent)
		}
	}
}

func TestSettleCloseScoreDoesNotChangeValue(t *testing.T) {
	sweep, _ := Settle(Input{Player1: placed(1300), Player2: placed(1200), Player1Won: true, RaceLength: RaceLength(5, 5, 0)})
	closeWin, _ := Settle(Input{Player1: placed(1300), Player2: placed(1200), Player1Won: true, RaceLength: RaceLength(5, 5, 4)})
	if sweep.Change != closeWin.Change {
		t.Fatalf("5:0 and 5:4 must score the same, got %+d and %+d", sweep.Change, closeWin.Change)
	}
}

func TestSettlePlacementAndOpponentDamping(t *testing.T) {
	newcomer, veteran := Settle(Input{
		Player1:    PlayerState{Rating: 1250, RatedMatches: 3},
		Player2:    placed(1250),
		Player1Won: true,
		RaceLength: 5,
	})
	if newcomer.Change != 90 || newcomer.Placement != 45 {
		t.Fatalf("placement must double the newcomer's change: %+v", newcomer)
	}
	if veteran.Change != -23 || veteran.OpponentDamping != 22 {
		t.Fatalf("placed opponent must lose half: %+v", veteran)
	}

	bothNew1, bothNew2 := Settle(Input{
		Player1:    PlayerState{Rating: 1250, RatedMatches: 0},
		Player2:    PlayerState{Rating: 1250, RatedMatches: 9},
		Player1Won: false,
		RaceLength: 5,
	})
	if bothNew1.Change != -90 || bothNew2.Change != 90 {
		t.Fatalf("two placement players must stay zero-sum: %+d/%+d", bothNew1.Change, bothNew2.Change)
	}
}

func TestSettleSameOpponentDecayIsSymmetric(t *testing.T) {
	cases := []struct {
		prior int
		want  int
		rated bool
	}{
		{prior: 0, want: 45, rated: true},
		{prior: 1, want: 45, rated: true},
		{prior: 2, want: 36, rated: true},
		{prior: 3, want: 14, rated: true},
		{prior: 5, want: 14, rated: true},
		{prior: 6, want: 0, rated: false},
		{prior: 9, want: 0, rated: false},
	}
	for _, tc := range cases {
		winner, loser := Settle(Input{Player1: placed(1500), Player2: placed(1500), Player1Won: true, RaceLength: 5, PriorPairMatchesToday: tc.prior})
		if winner.Change != tc.want || loser.Change != -tc.want || winner.Rated != tc.rated || loser.Rated != tc.rated {
			t.Fatalf("prior=%d: got winner=%+v loser=%+v", tc.prior, winner, loser)
		}
		if tc.rated && tc.want != 45 && winner.SameOpponent != tc.want-45 {
			t.Fatalf("prior=%d: decay detail must explain the reduction: %+v", tc.prior, winner)
		}
	}
}

func TestSettleDailyCapAndFloor(t *testing.T) {
	winner, loser := Settle(Input{
		Player1:          PlayerState{Rating: 1500, RatedMatches: PlacementMatches, TodayPositiveGain: 480},
		Player2:          placed(1500),
		Player1Won:       true,
		RaceLength:       5,
		DailyPositiveCap: 500,
	})
	if winner.Change != 20 || winner.DailyCap != -25 {
		t.Fatalf("daily cap must trim the gain to the remaining 20: %+v", winner)
	}
	if loser.Change != -45 {
		t.Fatalf("daily cap must not change the loser: %+v", loser)
	}

	_, nearZero := Settle(Input{Player1: placed(30), Player2: placed(12), Player1Won: true, RaceLength: 5})
	if nearZero.After != 0 || nearZero.Floor <= 0 || nearZero.Change != -12 {
		t.Fatalf("rating must not drop below zero: %+v", nearZero)
	}
}

func TestSettleDetailsSumToChange(t *testing.T) {
	for prior := 0; prior <= 6; prior++ {
		for _, gain := range []int{0, 470, 500} {
			for _, rated := range []int{0, 5, 12} {
				p1, p2 := Settle(Input{
					Player1:               PlayerState{Rating: 40, RatedMatches: rated, TodayPositiveGain: gain},
					Player2:               PlayerState{Rating: 900, RatedMatches: 12 - rated},
					Player1Won:            prior%2 == 0,
					RaceLength:            prior + 1,
					PriorPairMatchesToday: prior,
					DailyPositiveCap:      500,
				})
				for _, out := range []Outcome{p1, p2} {
					sum := out.Standard + out.Placement + out.OpponentDamping + out.SameOpponent + out.DailyCap + out.Floor
					if sum != out.Change || out.After != out.Before+out.Change || out.After < 0 {
						t.Fatalf("details must add up: %+v", out)
					}
				}
			}
		}
	}
}

func TestRemarkRoundTrip(t *testing.T) {
	out, _ := Settle(Input{Player1: PlayerState{Rating: 1250, RatedMatches: 2}, Player2: placed(1500), Player1Won: true, RaceLength: 3, PriorPairMatchesToday: 3})
	remark, ok := ParseRemark(RemarkFor(out))
	if !ok || !remark.Rated || remark.Standard != out.Standard || remark.Placement != out.Placement || remark.SameOpponent != out.SameOpponent || remark.RaceLength != 3 {
		t.Fatalf("remark must keep V2 details: remark=%+v out=%+v", remark, out)
	}
	if len(RemarkFor(out)) > 255 {
		t.Fatalf("remark must fit rank_change_logs.remark")
	}
	if _, ok := ParseRemark(`{"loss_floor_adjustment":-7}`); ok {
		t.Fatal("legacy remark must not parse as V2")
	}
	reset, ok := ParseRemark(ResetRemark(14))
	if !ok || reset.SeedMatches != 14 {
		t.Fatalf("reset remark must keep seed matches: %+v", reset)
	}
}

func TestStrengthBucket(t *testing.T) {
	if StrengthBucket(1000, 1300) != StrengthBucketStronger || StrengthBucket(1000, 1250) != StrengthBucketEven ||
		StrengthBucket(1000, 750) != StrengthBucketEven || StrengthBucket(1000, 700) != StrengthBucketWeaker {
		t.Fatal("strength buckets must use a ±250 gap")
	}
}
