package stats

import (
	"testing"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func TestSingleHighScoreReadsBoundedParticipantProjection(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	base := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	if err := svcCtx.DB.Create(&[]model.MatchParticipantResult{
		{MatchId: 1, UserId: 101, OpponentName: "甲", GameType: 3, MatchMode: model.MatchModeRanked, CompletedAt: base, MatchHighScore: 7},
		{MatchId: 2, UserId: 101, OpponentName: "乙", GameType: 3, MatchMode: model.MatchModeRanked, CompletedAt: base.Add(time.Hour), MatchHighScore: 9},
		{MatchId: 3, UserId: 101, OpponentName: "练习对手", GameType: 3, MatchMode: model.MatchModePractice, CompletedAt: base.Add(2 * time.Hour), MatchHighScore: 15},
	}).Error; err != nil {
		t.Fatalf("seed participant projections: %v", err)
	}

	resp, err := NewGetSingleHighScoreLogic(statsLogicCtx(101), svcCtx).GetSingleHighScore(&types.GetSingleHighScoreReq{GameType: 3, Limit: 1})
	if err != nil || !resp.Success || len(resp.List) != 1 {
		t.Fatalf("get single high score: resp=%#v err=%v", resp, err)
	}
	if record := resp.List[0]; record.MatchId != 2 || record.Score != 9 || record.OpponentName != "乙" || record.Date != "2026-08-11" {
		t.Fatalf("unexpected projected high score record: %#v", record)
	}
}

func TestOpponentStrengthKeepsSettlementBucketAfterOpponentPromotion(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	if err := svcCtx.DB.Create(&model.UserOpponentStrengthBucket{
		UserId: 101, GameType: 3, RankBucket: "score_0_1000", Matches: 2, Wins: 1,
	}).Error; err != nil {
		t.Fatalf("seed settlement strength bucket: %v", err)
	}
	if err := svcCtx.DB.Create(&model.UserRanking{UserId: 202, GameType: 3, RankScore: 2600, RankLevel: 6}).Error; err != nil {
		t.Fatalf("seed promoted current ranking: %v", err)
	}

	resp, err := NewGetOpponentStrengthLogic(statsLogicCtx(101), svcCtx).GetOpponentStrength(&types.GetOpponentStrengthReq{GameType: 3})
	if err != nil || !resp.Success || len(resp.List) != 1 {
		t.Fatalf("get opponent strength: resp=%#v err=%v", resp, err)
	}
	if item := resp.List[0]; item.RankRange != "初级(0-1000)" || item.Matches != 2 || item.Wins != 1 {
		t.Fatalf("current promotion must not rewrite historical bucket: %#v", item)
	}
}

func TestCurrentHighScoreUsesCompetitiveSnapshot(t *testing.T) {
	svcCtx := newStatsLogicTestSvc(t)
	if err := svcCtx.DB.Create(&[]model.UserCompetitiveStats{
		{UserId: 101, GameType: 3, HighestScore: 12},
		{UserId: 101, GameType: 1, HighestScore: 80, HighestBreak: 65},
	}).Error; err != nil {
		t.Fatalf("seed stats snapshots: %v", err)
	}

	poolScore, err := logicx.LoadUserMaxSingleScore(svcCtx, 101, 3)
	if err != nil || poolScore != 12 {
		t.Fatalf("pool current high score = %d, err=%v", poolScore, err)
	}
	snookerBreak, err := logicx.LoadUserMaxSingleScore(svcCtx, 101, 1)
	if err != nil || snookerBreak != 65 {
		t.Fatalf("snooker current high break = %d, err=%v", snookerBreak, err)
	}
}
