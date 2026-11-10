package match

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newScoreBreakdownDetailSvc(t *testing.T) (*svc.ServiceContext, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}); err != nil {
		t.Fatalf("prepare detail schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}, db
}

func seedScoreBreakdownUsers(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
}

// seedChasingScoreBreakdownMatch：选手1 普胜 2、小金 1、犯规 3 次，选手2 大金 1、犯规 3 次（含最后一次胜局后的 2 次），最终 18:13。
func seedChasingScoreBreakdownMatch(t *testing.T, db *gorm.DB, status int) int64 {
	t.Helper()
	const matchID = int64(501)
	opponentID := int64(2)
	if err := db.Create(&model.Match{
		Id: matchID, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 2,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
		Status: status, MyScore: 18, OpponentScore: 13, MatchTime: time.Date(2026, 10, 27, 20, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create match: %v", err)
	}

	base := time.Date(2026, 10, 27, 20, 0, 0, 0, time.UTC)
	win := func(round, actor, points int, winType string, offset time.Duration) model.MatchAction {
		extra := `{"win_type":"` + winType + `"}`
		return model.MatchAction{MatchId: matchID, RoundNo: round, ActionType: "win", Actor: actor, ScoreChange: points, ExtraData: &extra, CreatedAt: base.Add(offset)}
	}
	foul := func(round, actor int, offset time.Duration) model.MatchAction {
		return model.MatchAction{MatchId: matchID, RoundNo: round, ActionType: "foul", Actor: actor, ScoreChange: 1, CreatedAt: base.Add(offset)}
	}
	actions := []model.MatchAction{
		win(1, 1, 4, "normal", 0),
		foul(2, 2, time.Minute),
		win(2, 1, 7, "small_gold", 2*time.Minute),
		foul(3, 1, 3*time.Minute),
		foul(3, 1, 4*time.Minute),
		win(3, 2, 10, "big_gold", 5*time.Minute),
		foul(4, 1, 6*time.Minute),
		win(4, 1, 4, "normal", 7*time.Minute),
		foul(5, 2, 8*time.Minute),
		foul(5, 2, 9*time.Minute),
	}
	if err := db.Create(&actions).Error; err != nil {
		t.Fatalf("seed actions: %v", err)
	}

	round := func(no, player1, player2, winner int, winType string) model.MatchRound {
		w := winner
		return model.MatchRound{MatchId: matchID, RoundNo: no, MyScore: player1, OpponentScore: player2, Winner: &w, WinType: winType}
	}
	rounds := []model.MatchRound{
		round(1, 4, 0, 1, "normal"),
		round(2, 12, 0, 1, "small_gold"),
		round(3, 12, 12, 2, "big_gold"),
		round(4, 16, 13, 1, "normal"),
	}
	if err := db.Create(&rounds).Error; err != nil {
		t.Fatalf("seed rounds: %v", err)
	}
	return matchID
}

func loadScoreBreakdownDetail(t *testing.T, svcCtx *svc.ServiceContext, matchID, viewerID int64) types.MatchDetailData {
	t.Helper()
	resp, err := NewGetMatchDetailLogic(context.WithValue(context.Background(), "user_id", viewerID), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: matchID})
	if err != nil || !resp.Success {
		t.Fatalf("get match detail: resp=%#v err=%v", resp, err)
	}
	return resp.Match
}

func TestGetMatchDetailReturnsChasingScoreBreakdown(t *testing.T) {
	svcCtx, db := newScoreBreakdownDetailSvc(t)
	seedScoreBreakdownUsers(t, db)
	matchID := seedChasingScoreBreakdownMatch(t, db, 2)

	detail := loadScoreBreakdownDetail(t, svcCtx, matchID, 1)
	breakdown := detail.ScoreBreakdown
	if breakdown == nil || !breakdown.Available || breakdown.EventsTruncated {
		t.Fatalf("unexpected breakdown: %+v", breakdown)
	}
	if breakdown.Player1 != (types.ChasingScoreSide{Score: 18, NormalWins: 2, SmallGolds: 1, FoulCount: 3, FoulPointsGiven: 3}) {
		t.Fatalf("player1: %+v", breakdown.Player1)
	}
	// 两名选手都在最后一次胜局后犯规，尾部犯规必须计入。
	if breakdown.Player2 != (types.ChasingScoreSide{Score: 13, BigGolds: 1, FoulCount: 3, FoulPointsGiven: 3}) {
		t.Fatalf("player2: %+v", breakdown.Player2)
	}
	if len(breakdown.Events) != 10 || breakdown.Events[0].Kind != "normal" || breakdown.Events[9].Kind != "foul" {
		t.Fatalf("events: %+v", breakdown.Events)
	}
}

func TestGetMatchDetailKeepsChasingScoreBreakdownInAbsolutePositions(t *testing.T) {
	svcCtx, db := newScoreBreakdownDetailSvc(t)
	seedScoreBreakdownUsers(t, db)
	matchID := seedChasingScoreBreakdownMatch(t, db, 2)

	detail := loadScoreBreakdownDetail(t, svcCtx, matchID, 2)
	if detail.IsPlayer1 || detail.MyScore != 13 || detail.OpponentScore != 18 {
		t.Fatalf("opponent view must be swapped: %+v", detail)
	}
	breakdown := detail.ScoreBreakdown
	if breakdown == nil {
		t.Fatal("breakdown missing")
	}
	// 得分构成是绝对位置：选手1 仍是 18 分，不随查看者交换。
	if breakdown.Player1.Score != 18 || breakdown.Player1.NormalWins != 2 || breakdown.Player1.SmallGolds != 1 {
		t.Fatalf("player1 must stay absolute: %+v", breakdown.Player1)
	}
	if breakdown.Player2.Score != 13 || breakdown.Player2.BigGolds != 1 {
		t.Fatalf("player2 must stay absolute: %+v", breakdown.Player2)
	}
}

func TestGetMatchDetailOmitsChasingScoreBreakdownForOtherGameTypes(t *testing.T) {
	svcCtx, db := newScoreBreakdownDetailSvc(t)
	seedScoreBreakdownUsers(t, db)
	opponentID := int64(2)
	if err := db.Create(&model.Match{
		Id: 502, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3,
		MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
		Status: 2, MyScore: 3, OpponentScore: 1, MatchTime: time.Date(2026, 10, 27, 20, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create match: %v", err)
	}

	detail := loadScoreBreakdownDetail(t, svcCtx, 502, 1)
	if detail.ScoreBreakdown != nil {
		t.Fatalf("other game types must not carry a breakdown: %+v", detail.ScoreBreakdown)
	}
}
