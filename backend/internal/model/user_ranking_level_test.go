package model

import "testing"

func TestRankingModelCalculateLevelUsesSixTierBoundaries(t *testing.T) {
	model := NewRankingModel(nil)
	tests := []struct {
		score int
		level int
	}{
		{score: 1999, level: 4},
		{score: 2000, level: 5},
		{score: 2499, level: 5},
		{score: 2500, level: 6},
	}

	for _, tt := range tests {
		if got := model.CalculateLevel(tt.score); got != tt.level {
			t.Fatalf("score %d: expected level %d, got %d", tt.score, tt.level, got)
		}
	}
}
