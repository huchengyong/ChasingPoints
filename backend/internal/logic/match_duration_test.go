package logic

import (
	"testing"
	"time"

	"chasing_points/internal/model"
)

func TestMatchDurationSeconds(t *testing.T) {
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	endTime := now.Add(-time.Minute)
	cases := []struct {
		name  string
		match *model.Match
		want  int64
	}{
		{name: "nil match", match: nil, want: 0},
		{name: "ongoing uses now", match: &model.Match{Status: 1, CreatedAt: now.Add(-90 * time.Second)}, want: 90},
		{name: "completed uses end time", match: &model.Match{Status: 2, CreatedAt: endTime.Add(-2 * time.Minute), EndTime: &endTime}, want: 120},
		{name: "cancelled uses end time", match: &model.Match{Status: 3, CreatedAt: endTime.Add(-30 * time.Second), EndTime: &endTime}, want: 30},
		{name: "negative clamps to zero", match: &model.Match{Status: 2, CreatedAt: endTime.Add(time.Minute), EndTime: &endTime}, want: 0},
	}
	for _, testCase := range cases {
		if got := MatchDurationSeconds(testCase.match, now); got != testCase.want {
			t.Fatalf("%s: got %d, want %d", testCase.name, got, testCase.want)
		}
	}
}
