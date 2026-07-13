package stats

import "testing"

func TestCalculateWinRatePercentReturnsDisplayPercent(t *testing.T) {
	tests := []struct {
		name    string
		wins    int
		matches int
		want    float64
	}{
		{name: "all wins", wins: 2, matches: 2, want: 100},
		{name: "partial wins", wins: 20, matches: 22, want: 90.9090909090909},
		{name: "no matches", wins: 0, matches: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateWinRatePercent(tt.wins, tt.matches)
			if got != tt.want {
				t.Fatalf("calculateWinRatePercent(%d, %d) = %v, want %v", tt.wins, tt.matches, got, tt.want)
			}
		})
	}
}
