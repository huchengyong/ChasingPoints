package match

import (
	"reflect"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

// ratingOutcomeSnapshot 只收集与排位公平有关的结算结果，去掉主键与时间戳等每次都会不同的字段。
type ratingOutcomeSnapshot struct {
	Rankings []rankingFields
	Logs     []rankLogFields
}

type rankingFields struct {
	UserID                      int64
	RankScore, RankLevel        int
	TotalWins, TotalLosses      int
	CurrentStreak, MaxStreak    int
	RatingVersion, RatedMatches int
}

type rankLogFields struct {
	UserID                                   int64
	ChangeType, Result                       string
	BaseScore, AchievementScore, FinalChange int
	BeforeScore, AfterScore                  int
	BeforeLevel, AfterLevel                  int
	Remark                                   string
}

func collectRatingOutcome(t *testing.T, svcCtx *svc.ServiceContext) ratingOutcomeSnapshot {
	t.Helper()
	var snapshot ratingOutcomeSnapshot
	for _, userID := range []int64{1001, 2002} {
		row := findRanking(t, svcCtx, userID)
		snapshot.Rankings = append(snapshot.Rankings, rankingFields{
			UserID: userID, RankScore: row.RankScore, RankLevel: row.RankLevel,
			TotalWins: row.TotalWins, TotalLosses: row.TotalLosses,
			CurrentStreak: row.CurrentStreak, MaxStreak: row.MaxStreak,
			RatingVersion: row.RatingVersion, RatedMatches: row.RatedMatches,
		})
	}
	var logs []model.RankChangeLog
	if err := svcCtx.DB.Order("user_id ASC, change_type ASC").Find(&logs).Error; err != nil {
		t.Fatalf("load rank change logs: %v", err)
	}
	for _, item := range logs {
		snapshot.Logs = append(snapshot.Logs, rankLogFields{
			UserID: item.UserId, ChangeType: item.ChangeType, Result: item.Result,
			BaseScore: item.BaseScore, AchievementScore: item.AchievementScore, FinalChange: item.FinalChange,
			BeforeScore: item.BeforeScore, AfterScore: item.AfterScore,
			BeforeLevel: item.BeforeLevel, AfterLevel: item.AfterLevel, Remark: item.Remark,
		})
	}
	return snapshot
}

// 会员权益不得影响竞技公平：同一场 V2 排位，无论哪一方是不是会员，双方的排位分、段位、计数与结算记录都必须完全一致。
func TestRankedRatingV2OutcomeIsIdenticalRegardlessOfMembership(t *testing.T) {
	future := time.Now().Add(30 * 24 * time.Hour)
	scenarios := []struct {
		name             string
		player1, player2 *time.Time
	}{
		{name: "both free"},
		{name: "player1 member", player1: &future},
		{name: "player2 member", player2: &future},
		{name: "both members", player1: &future, player2: &future},
	}

	snapshots := make(map[string]ratingOutcomeSnapshot, len(scenarios))
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			svcCtx, matchID := newRatingV2FinishFixture(t, 0)
			for userID, expiresAt := range map[int64]*time.Time{1001: scenario.player1, 2002: scenario.player2} {
				if err := svcCtx.DB.Model(&model.User{}).Where("id = ?", userID).Update("member_expires_at", expiresAt).Error; err != nil {
					t.Fatalf("set membership for %d: %v", userID, err)
				}
			}

			resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: matchID, ClientActionId: "fairness-" + scenario.name})
			if err != nil || resp == nil || !resp.Success {
				t.Fatalf("finish match: resp=%#v err=%v", resp, err)
			}
			snapshot := collectRatingOutcome(t, svcCtx)
			if len(snapshot.Rankings) != 2 || len(snapshot.Logs) < 4 {
				t.Fatalf("fixture must settle both players with a reset and a result log each: %+v", snapshot)
			}
			for _, item := range snapshot.Logs {
				if item.AchievementScore != 0 {
					t.Fatalf("V2 must never add member highlight points: %+v", item)
				}
			}
			snapshots[scenario.name] = snapshot
		})
	}

	baseline, ok := snapshots["both free"]
	if !ok {
		t.Fatal("baseline scenario did not run")
	}
	for name, snapshot := range snapshots {
		if !reflect.DeepEqual(snapshot, baseline) {
			t.Fatalf("scenario %q changed the rating outcome:\n got=%+v\nwant=%+v", name, snapshot, baseline)
		}
	}
}
