package public

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPublicMatchListUsesJoinedRoundCountsWithConstantQueries(t *testing.T) {
	oneQueries := publicMatchListQueryCount(t, 1)
	hundredQueries := publicMatchListQueryCount(t, 100)
	if oneQueries != 2 || hundredQueries != 2 {
		t.Fatalf("public match list must use one COUNT and one joined page query: one=%d hundred=%d", oneQueries, hundredQueries)
	}
}

func publicMatchListQueryCount(t *testing.T, matchCount int) int {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", matchCount)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}); err != nil {
		t.Fatalf("prepare public match schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	opponentID := int64(2)
	result := 1
	winner := 1
	now := time.Now()
	matches := make([]model.Match, 0, matchCount)
	rounds := make([]model.MatchRound, 0, matchCount)
	for i := 1; i <= matchCount; i++ {
		matchID := int64(i)
		matches = append(matches, model.Match{Id: matchID, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3, MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, Status: 2, Result: &result, MatchTime: now.Add(-time.Duration(i) * time.Minute)})
		rounds = append(rounds, model.MatchRound{MatchId: matchID, RoundNo: 1, Winner: &winner})
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed matches: %v", err)
	}
	if err := db.Create(&rounds).Error; err != nil {
		t.Fatalf("seed rounds: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.Background(), metrics)
	svcCtx := &svc.ServiceContext{DB: db.WithContext(ctx), MatchModel: model.NewMatchModel(db.WithContext(ctx))}
	resp, err := NewGetPublicMatchesLogic(ctx, svcCtx).GetPublicMatches(&types.PublicMatchListReq{Scope: "hall", Status: 2, Page: 1, PageSize: 50})
	expectedLen := matchCount
	if expectedLen > 50 {
		expectedLen = 50
	}
	if err != nil || !resp.Success || resp.Total != int64(matchCount) || len(resp.List) != expectedLen {
		t.Fatalf("get public matches: resp=%#v err=%v", resp, err)
	}
	if resp.List[0].Player1Name != "甲" || resp.List[0].Player2Avatar != "" || resp.List[0].CurrentRound != 1 {
		t.Fatalf("joined public match fields missing: %#v", resp.List[0])
	}
	return int(metrics.Snapshot().SQLCount)
}
