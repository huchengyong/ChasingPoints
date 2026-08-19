package model

import "testing"

func TestNormalizePoolMatchFormat(t *testing.T) {
	tests := []struct {
		name       string
		gameType   int
		format     string
		targetWins int
		wantFormat string
		wantTarget int
		wantOK     bool
	}{
		{name: "legacy pool", gameType: 3, wantFormat: MatchFormatLegacy, wantOK: true},
		{name: "legacy rejects target", gameType: 3, targetWins: 1, wantFormat: MatchFormatLegacy, wantOK: false},
		{name: "free chinese eight", gameType: 3, format: MatchFormatFree, wantFormat: MatchFormatFree, wantOK: true},
		{name: "free american nine", gameType: 4, format: MatchFormatFree, wantFormat: MatchFormatFree, wantOK: true},
		{name: "race to one", gameType: 3, format: MatchFormatRaceTo, targetWins: 1, wantFormat: MatchFormatRaceTo, wantTarget: 1, wantOK: true},
		{name: "race to sixty five", gameType: 4, format: MatchFormatRaceTo, targetWins: 65, wantFormat: MatchFormatRaceTo, wantTarget: 65, wantOK: true},
		{name: "race target too large", gameType: 3, format: MatchFormatRaceTo, targetWins: 66, wantFormat: MatchFormatRaceTo, wantTarget: 66, wantOK: false},
		{name: "free rejects target", gameType: 3, format: MatchFormatFree, targetWins: 1, wantFormat: MatchFormatFree, wantTarget: 0, wantOK: false},
		{name: "unsupported snooker legacy", gameType: 1, wantFormat: MatchFormatLegacy, wantOK: true},
		{name: "unsupported snooker legacy rejects target", gameType: 1, targetWins: 1, wantFormat: MatchFormatLegacy, wantOK: false},
		{name: "unsupported nine ball score chasing", gameType: 2, wantFormat: MatchFormatLegacy, wantOK: true},
		{name: "unsupported game type rejects free", gameType: 1, format: MatchFormatFree, wantOK: false},
		{name: "unknown", gameType: 3, format: "best_of", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, target, ok := NormalizePoolMatchFormat(tt.gameType, tt.format, tt.targetWins)
			if format != tt.wantFormat || target != tt.wantTarget || ok != tt.wantOK {
				t.Fatalf("got (%q, %d, %v), want (%q, %d, %v)", format, target, ok, tt.wantFormat, tt.wantTarget, tt.wantOK)
			}
		})
	}
}

func TestPoolMatchTargetReached(t *testing.T) {
	match := &Match{GameType: 3, MatchFormat: MatchFormatRaceTo, TargetWins: 65, MyScore: 65, OpponentScore: 64}
	if !PoolMatchTargetReached(match) {
		t.Fatal("expected target to be reached")
	}

	match.MatchFormat = MatchFormatFree
	if PoolMatchTargetReached(match) {
		t.Fatal("free format must not use a target win threshold")
	}

	match.GameType = 2
	match.MatchFormat = MatchFormatRaceTo
	if PoolMatchTargetReached(match) {
		t.Fatal("unsupported game type must not use pool match target")
	}
}
