package ranking

import (
	"math"
	"testing"
)

func repeatedSamples(opponent, raceLength, wins, losses int) []PerformanceSample {
	samples := make([]PerformanceSample, 0, wins+losses)
	for range wins {
		samples = append(samples, PerformanceSample{OpponentRating: opponent, RaceLength: raceLength, Won: true})
	}
	for range losses {
		samples = append(samples, PerformanceSample{OpponentRating: opponent, RaceLength: raceLength})
	}
	return samples
}

// 期望值由独立实现（Python，浮点二分）算出，不用被测代码自证。
func TestPerformanceRatingMatchesIndependentlyComputedValues(t *testing.T) {
	cases := []struct {
		name    string
		samples []PerformanceSample
		want    int
		bound   Bound
	}{
		{"race-to-5 vs five 1500, 3 wins 2 losses", repeatedSamples(1500, 5, 3, 2), 1678, BoundExact},
		{"all wins solves for 4.5 and is a lower bound", repeatedSamples(1500, 5, 5, 0), 2415, BoundAtLeast},
		{"all losses solves for 0.5 and is an upper bound", repeatedSamples(1500, 5, 0, 5), 585, BoundAtMost},
		{"one win in five", repeatedSamples(1500, 5, 1, 4), 904, BoundExact},
		{"mixed opponents and race lengths", []PerformanceSample{
			{1000, 5, true}, {2000, 3, false}, {1500, 7, true}, {1800, 5, false}, {1200, 9, true}, {1600, 1, false},
		}, 1470, BoundExact},
		{"three samples are enough for a month", []PerformanceSample{{1500, 5, true}, {1500, 5, true}, {1500, 5, false}}, 1804, BoundExact},
		// 上界要向外扩展：200 场抢 1 全胜，需要远高于对手分 + 3000。
		{"search bound expands for very long winning runs", repeatedSamples(1500, 1, 200, 0), 8002, BoundAtLeast},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, bound, ok := PerformanceRating(tc.samples, 3)
			if !ok || got != tc.want || bound != tc.bound {
				t.Fatalf("got rating=%d bound=%q ok=%v want rating=%d bound=%q", got, bound, ok, tc.want, tc.bound)
			}
		})
	}
}

func TestPerformanceRatingNeedsEnoughSamples(t *testing.T) {
	four := repeatedSamples(1500, 5, 2, 2)
	if _, _, ok := PerformanceRating(four, MinPerformanceSamples); ok {
		t.Fatal("four samples must not produce a 90-day performance rating")
	}
	if _, _, ok := PerformanceRating(four, MinMonthlyPerformanceSamples); !ok {
		t.Fatal("four samples are enough for a monthly rating")
	}
	if _, _, ok := PerformanceRating(nil, MinMonthlyPerformanceSamples); ok {
		t.Fatal("no samples must never produce a rating")
	}
	if _, _, ok := PerformanceRating(nil, 0); ok {
		t.Fatal("a zero minimum must still reject an empty sample set")
	}
}

func TestPerformanceRatingNeverGoesBelowZero(t *testing.T) {
	// 对手分很低又全负：独立计算的原始解是 −2285，展示时夹到 0，并保持「至多」。
	got, bound, ok := PerformanceRating(repeatedSamples(100, 1, 0, 5), MinPerformanceSamples)
	if !ok || got != 0 || bound != BoundAtMost {
		t.Fatalf("expected a floored upper bound, got rating=%d bound=%q ok=%v", got, bound, ok)
	}
	// 200 场只赢 1 场，原始解 −4247。
	got, bound, ok = PerformanceRating(append(repeatedSamples(1500, 1, 0, 199), PerformanceSample{OpponentRating: 1500, RaceLength: 1, Won: true}), MinPerformanceSamples)
	if !ok || got != 0 || bound != BoundExact {
		t.Fatalf("expected an exact value floored at zero, got rating=%d bound=%q ok=%v", got, bound, ok)
	}
}

func TestPerformanceRatingIsOrderIndependentAndMonotonicInWins(t *testing.T) {
	forward := []PerformanceSample{{1000, 5, true}, {2000, 3, false}, {1500, 7, true}, {1800, 5, false}, {1200, 9, true}, {1600, 1, false}}
	backward := make([]PerformanceSample, len(forward))
	for i, sample := range forward {
		backward[len(forward)-1-i] = sample
	}
	a, _, _ := PerformanceRating(forward, MinPerformanceSamples)
	b, _, _ := PerformanceRating(backward, MinPerformanceSamples)
	if a != b {
		t.Fatalf("sample order must not matter: %d vs %d", a, b)
	}

	previous := -1
	for wins := 0; wins <= 5; wins++ {
		got, _, ok := PerformanceRating(repeatedSamples(1500, 5, wins, 5-wins), MinPerformanceSamples)
		if !ok || got <= previous {
			t.Fatalf("rating must grow with wins: wins=%d got=%d previous=%d", wins, got, previous)
		}
		previous = got
	}
}

func TestPerformanceRatingExpectedWinsMatchTargetAtTheSolution(t *testing.T) {
	samples := []PerformanceSample{{1000, 5, true}, {2000, 3, false}, {1500, 7, true}, {1800, 5, false}, {1200, 9, true}, {1600, 1, false}}
	rating, _, ok := PerformanceRating(samples, MinPerformanceSamples)
	if !ok {
		t.Fatal("expected a rating")
	}
	total := 0.0
	for _, sample := range samples {
		total += RaceWinProbability(RackWinProbability(rating, sample.OpponentRating), sample.RaceLength)
	}
	// 三胜三负：解出的表现分对应的期望胜场必须落在 3 附近（取整带来的误差远小于 0.05）。
	if math.Abs(total-3) > 0.05 {
		t.Fatalf("expected wins at the solved rating should be about 3, got %.4f", total)
	}
}
