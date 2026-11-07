package model

import "testing"

func TestNormalizeSnookerFormat(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		targetWins   int
		bestOfFrames int
		wantFormat   string
		wantTarget   int
		wantOK       bool
	}{
		{name: "free", format: SnookerFormatFree, wantFormat: SnookerFormatFree, wantOK: true},
		{name: "race to one", format: SnookerFormatRaceTo, targetWins: 1, wantFormat: SnookerFormatRaceTo, wantTarget: 1, wantOK: true},
		{name: "race to twenty five", format: SnookerFormatRaceTo, targetWins: 25, wantFormat: SnookerFormatRaceTo, wantTarget: 25, wantOK: true},
		{name: "race target too large", format: SnookerFormatRaceTo, targetWins: 26, wantFormat: SnookerFormatRaceTo, wantTarget: 26, wantOK: false},
		{name: "legacy best of three", bestOfFrames: 3, wantFormat: SnookerFormatRaceTo, wantTarget: 2, wantOK: true},
		{name: "legacy missing format", wantFormat: SnookerFormatLegacy, wantOK: true},
		{name: "unknown", format: "round_robin", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format, target, ok := NormalizeSnookerFormat(tt.format, tt.targetWins, tt.bestOfFrames)
			if format != tt.wantFormat || target != tt.wantTarget || ok != tt.wantOK {
				t.Fatalf("got (%q, %d, %v), want (%q, %d, %v)", format, target, ok, tt.wantFormat, tt.wantTarget, tt.wantOK)
			}
		})
	}
}

func TestSnookerTargetReached(t *testing.T) {
	match := &Match{GameType: 1, SnookerFormat: SnookerFormatRaceTo, SnookerTargetWins: 10, MyScore: 10, OpponentScore: 9}
	if !SnookerTargetReached(match) {
		t.Fatal("expected target to be reached")
	}
	match.SnookerFormat = SnookerFormatFree
	if SnookerTargetReached(match) {
		t.Fatal("free format must not use a target win threshold")
	}
}
