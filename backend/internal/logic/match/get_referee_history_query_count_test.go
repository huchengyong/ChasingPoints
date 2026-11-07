package match

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

func TestGetRefereeHistoryUsesJoinedProfilesWithConstantQueries(t *testing.T) {
	oneQueries := refereeHistoryQueryCount(t, 1)
	hundredQueries := refereeHistoryQueryCount(t, 100)
	if oneQueries != 2 || hundredQueries != 2 {
		t.Fatalf("referee history must use one COUNT and one joined page query: one=%d hundred=%d", oneQueries, hundredQueries)
	}
}

func refereeHistoryQueryCount(t *testing.T, matchCount int) int {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", matchCount)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}); err != nil {
		t.Fatalf("prepare referee schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "裁判"}, {Id: 2, Nickname: "甲"}, {Id: 3, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	opponentID := int64(3)
	refereeID := int64(1)
	result := 1
	now := time.Now()
	matches := make([]model.Match, 0, matchCount)
	for i := 0; i < matchCount; i++ {
		matches = append(matches, model.Match{Id: int64(i + 1), UserId: 2, OpponentId: &opponentID, OpponentName: "乙", GameType: 3, Status: 2, Result: &result, RefereeUserId: &refereeID, MatchTime: now.Add(-time.Duration(i+1) * time.Hour), EndTime: &now})
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed matches: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	svcCtx := &svc.ServiceContext{MatchModel: model.NewMatchModel(db.WithContext(ctx))}
	resp, err := NewGetRefereeHistoryLogic(ctx, svcCtx).GetRefereeHistory(&types.RefereeHistoryReq{Page: 1, PageSize: 100})
	if err != nil || !resp.Success || resp.Total != int64(matchCount) || len(resp.List) != matchCount {
		t.Fatalf("get referee history: resp=%#v err=%v", resp, err)
	}
	if resp.List[0].Player1Name != "甲" || resp.List[0].Player2Name != "乙" {
		t.Fatalf("joined player profiles missing: %#v", resp.List[0])
	}
	return int(metrics.Snapshot().SQLCount)
}
