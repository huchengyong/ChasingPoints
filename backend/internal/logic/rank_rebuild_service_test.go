package logic

import (
	"testing"
	"time"

	"chasing_points/internal/model"
)

func TestUpdateReplaySettlementStateCountsDrawForSameOpponentDailyLimit(t *testing.T) {
	dayStart := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	match := &model.Match{
		Id:         11,
		UserId:     8,
		OpponentId: int64Ptr(18),
		GameType:   3,
	}

	dailyPositiveGains := make(map[rankDailyGainKey]int)
	sameOpponentDailyCounts := make(map[rankPairDailyKey]int)

	updateReplaySettlementState(dailyPositiveGains, sameOpponentDailyCounts, match, dayStart, 0, 0)

	key := buildRankPairDailyKey(8, 18, 3, dayStart)
	if sameOpponentDailyCounts[key] != 1 {
		t.Fatalf("expected draw to count toward same-opponent daily limit, got %d", sameOpponentDailyCounts[key])
	}
}

func TestUpdateReplaySettlementStateOnlyAccumulatesPositiveDailyGain(t *testing.T) {
	dayStart := time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC)
	match := &model.Match{
		Id:         12,
		UserId:     8,
		OpponentId: int64Ptr(18),
		GameType:   3,
	}

	dailyPositiveGains := make(map[rankDailyGainKey]int)
	sameOpponentDailyCounts := make(map[rankPairDailyKey]int)

	updateReplaySettlementState(dailyPositiveGains, sameOpponentDailyCounts, match, dayStart, 28, -2)

	player1Key := rankDailyGainKey{UserId: 8, GameType: 3, DayStart: dayStart}
	player2Key := rankDailyGainKey{UserId: 18, GameType: 3, DayStart: dayStart}
	if dailyPositiveGains[player1Key] != 28 {
		t.Fatalf("expected player1 positive gain 28, got %d", dailyPositiveGains[player1Key])
	}
	if dailyPositiveGains[player2Key] != 0 {
		t.Fatalf("expected player2 non-positive gain to be ignored, got %d", dailyPositiveGains[player2Key])
	}
}
