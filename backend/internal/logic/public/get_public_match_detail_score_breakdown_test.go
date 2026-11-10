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

func newPublicScoreBreakdownSvc(t *testing.T) (*svc.ServiceContext, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}); err != nil {
		t.Fatalf("prepare public detail schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	return &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}, db
}

func seedPublicMatch(t *testing.T, db *gorm.DB, matchID int64, gameType, status, myScore, opponentScore int) {
	t.Helper()
	opponentID := int64(2)
	if err := db.Create(&model.Match{
		Id: matchID, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: gameType,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic,
		Status: status, MyScore: myScore, OpponentScore: opponentScore,
		MatchTime: time.Date(2026, 10, 27, 20, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create match: %v", err)
	}
}

// seedPublicChasingActions：选手2 犯规，选手1 普胜，选手2 大金，累计 5:10。
func seedPublicChasingActions(t *testing.T, db *gorm.DB, matchID int64) {
	t.Helper()
	base := time.Date(2026, 10, 27, 20, 0, 0, 0, time.UTC)
	normal := `{"win_type":"normal"}`
	bigGold := `{"win_type":"big_gold"}`
	actions := []model.MatchAction{
		{MatchId: matchID, RoundNo: 1, ActionType: "foul", Actor: 2, ScoreChange: 1, CreatedAt: base},
		{MatchId: matchID, RoundNo: 2, ActionType: "win", Actor: 1, ScoreChange: 4, ExtraData: &normal, CreatedAt: base.Add(time.Minute)},
		{MatchId: matchID, RoundNo: 3, ActionType: "win", Actor: 2, ScoreChange: 10, ExtraData: &bigGold, CreatedAt: base.Add(2 * time.Minute)},
	}
	if err := db.Create(&actions).Error; err != nil {
		t.Fatalf("seed actions: %v", err)
	}
}

func loadPublicDetail(t *testing.T, svcCtx *svc.ServiceContext, matchID int64) types.PublicMatchDetailData {
	t.Helper()
	resp, err := NewGetPublicMatchDetailLogic(context.Background(), svcCtx).GetPublicMatchDetail(&types.GetPublicMatchDetailReq{MatchId: matchID})
	if err != nil || !resp.Success || resp.Match == nil {
		t.Fatalf("get public match detail: resp=%#v err=%v", resp, err)
	}
	return *resp.Match
}

func TestGetPublicMatchDetailReturnsChasingScoreBreakdownInProgress(t *testing.T) {
	svcCtx, db := newPublicScoreBreakdownSvc(t)
	seedPublicMatch(t, db, 601, 2, 1, 5, 10)
	seedPublicChasingActions(t, db, 601)

	detail := loadPublicDetail(t, svcCtx, 601)
	breakdown := detail.ScoreBreakdown
	if breakdown == nil || !breakdown.Available {
		t.Fatalf("in-progress breakdown missing: %+v", breakdown)
	}
	if breakdown.Player1 != (types.ChasingScoreSide{Score: 5, NormalWins: 1}) ||
		breakdown.Player2 != (types.ChasingScoreSide{Score: 10, BigGolds: 1, FoulCount: 1, FoulPointsGiven: 1}) {
		t.Fatalf("in-progress breakdown: %+v / %+v", breakdown.Player1, breakdown.Player2)
	}
	if len(breakdown.Events) != 3 || breakdown.Events[0].Kind != "foul" || breakdown.Events[2].Kind != "big_gold" {
		t.Fatalf("events: %+v", breakdown.Events)
	}
}

func TestGetPublicMatchDetailReturnsChasingScoreBreakdownForCompletedMatch(t *testing.T) {
	svcCtx, db := newPublicScoreBreakdownSvc(t)
	seedPublicMatch(t, db, 602, 2, 2, 5, 10)
	seedPublicChasingActions(t, db, 602)

	detail := loadPublicDetail(t, svcCtx, 602)
	if detail.ScoreBreakdown == nil || !detail.ScoreBreakdown.Available {
		t.Fatalf("completed breakdown missing: %+v", detail.ScoreBreakdown)
	}
	if detail.ScoreBreakdown.Player1.Score != 5 || detail.ScoreBreakdown.Player2.Score != 10 {
		t.Fatalf("completed breakdown: %+v", detail.ScoreBreakdown)
	}
}

func TestGetPublicMatchDetailOmitsChasingScoreBreakdownForOtherGameTypes(t *testing.T) {
	svcCtx, db := newPublicScoreBreakdownSvc(t)
	seedPublicMatch(t, db, 603, 3, 1, 2, 1)

	detail := loadPublicDetail(t, svcCtx, 603)
	if detail.ScoreBreakdown != nil {
		t.Fatalf("other game types must not carry a breakdown: %+v", detail.ScoreBreakdown)
	}
}

func TestPublicMatchDetailChasingAddsTwoQueries(t *testing.T) {
	nineBallQueries := publicDetailQueryCount(t, 2)
	eightBallQueries := publicDetailQueryCount(t, 3)
	if nineBallQueries != eightBallQueries+2 {
		t.Fatalf("chasing detail must add exactly one COUNT and one action query: nine-ball=%d eight-ball=%d", nineBallQueries, eightBallQueries)
	}
}

func publicDetailQueryCount(t *testing.T, gameType int) int {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", gameType)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare public detail schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	seedPublicMatch(t, db, 604, gameType, 1, 5, 10)
	seedPublicChasingActions(t, db, 604)

	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.Background(), metrics)
	svcCtx := &svc.ServiceContext{
		DB:         db.WithContext(ctx),
		UserModel:  model.NewUserModel(db.WithContext(ctx)),
		MatchModel: model.NewMatchModel(db.WithContext(ctx)),
	}
	resp, err := NewGetPublicMatchDetailLogic(ctx, svcCtx).GetPublicMatchDetail(&types.GetPublicMatchDetailReq{MatchId: 604})
	if err != nil || !resp.Success || resp.Match == nil {
		t.Fatalf("get public match detail: resp=%#v err=%v", resp, err)
	}
	return int(metrics.Snapshot().SQLCount)
}
