package opponent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOpponentCandidatesRejectMissingAuthentication(t *testing.T) {
	resp, err := NewGetOpponentCandidatesLogic(context.Background(), &svc.ServiceContext{}).GetOpponentCandidates(&types.GetOpponentCandidatesReq{})
	if err != nil || resp.Success || len(resp.List) != 0 {
		t.Fatalf("unauthenticated candidates must not return data: resp=%#v err=%v", resp, err)
	}
}

func TestOpponentCandidatesKeepsRecentAvailableWhenFriendServiceIsMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRanking{}, &model.RankConfig{}, &model.UserOpponentStats{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare candidate schema: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, Revision: 2}).Error; err != nil {
		t.Fatalf("seed revision: %v", err)
	}
	resp, err := NewGetOpponentCandidatesLogic(context.WithValue(context.Background(), "user_id", int64(1)), &svc.ServiceContext{CompetitiveReadModel: model.NewCompetitiveReadModel(db), Config: config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}}}).GetOpponentCandidates(&types.GetOpponentCandidatesReq{})
	if err != nil || !resp.Success || resp.Availability["friends"] || !resp.Availability["recent"] || !resp.Availability["competitive_revision"] {
		t.Fatalf("missing friend service must remain a partial failure: resp=%#v err=%v", resp, err)
	}
}

func TestOpponentCandidatesMergeFriendsAndRecentByStableUserID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Friend{}, &model.UserRanking{}, &model.RankConfig{}, &model.UserOpponentStats{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare candidate schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "我"}, {Id: 2, Nickname: "好友"}, {Id: 3, Nickname: "近期对手"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&[]model.RankConfig{{Level: 1, Name: "新手"}, {Level: 2, Name: "进阶"}}).Error; err != nil {
		t.Fatalf("seed rank configs: %v", err)
	}
	if err := db.Create(&[]model.UserRanking{{UserId: 2, GameType: 3, RankLevel: 2}, {UserId: 3, GameType: 3, RankLevel: 1}}).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	if err := db.Create(&model.Friend{UserId: 1, FriendId: 2, Status: 1}).Error; err != nil {
		t.Fatalf("seed friend: %v", err)
	}
	now := time.Now()
	if err := db.Create(&[]model.UserOpponentStats{
		{UserId: 1, OpponentUserId: 2, OpponentNameKey: "user:2", OpponentName: "旧昵称", GameType: 0, LastMatchAt: &now, LastMatchId: 2},
		{UserId: 1, OpponentUserId: 3, OpponentNameKey: "user:3", OpponentName: "近期对手", GameType: 0, LastMatchAt: &now, LastMatchId: 3},
	}).Error; err != nil {
		t.Fatalf("seed opponent stats: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, Revision: 4}).Error; err != nil {
		t.Fatalf("seed revision: %v", err)
	}
	svcCtx := &svc.ServiceContext{FriendModel: model.NewFriendModel(db), CompetitiveReadModel: model.NewCompetitiveReadModel(db), Config: config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}}}
	resp, err := NewGetOpponentCandidatesLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetOpponentCandidates(&types.GetOpponentCandidatesReq{Limit: 10})
	if err != nil || !resp.Success || len(resp.List) != 2 || resp.CompetitiveRevision != 4 {
		t.Fatalf("get candidates: resp=%#v err=%v", resp, err)
	}
	if resp.List[0].UserId != 2 || resp.List[0].Source != "friend" || resp.List[0].RankName != "进阶" || resp.List[1].UserId != 3 || resp.List[1].Source != "recent" {
		t.Fatalf("unexpected deduplicated candidates: %#v", resp.List)
	}
}

func TestOpponentCandidatesUseFixedQueryCountForOneAndHundredRows(t *testing.T) {
	one := opponentCandidatesQueryCount(t, 1)
	hundred := opponentCandidatesQueryCount(t, 100)
	if one != 3 || hundred != 3 {
		t.Fatalf("opponent candidates must use three fixed queries: one=%d hundred=%d", one, hundred)
	}
}

func opponentCandidatesQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Friend{}, &model.UserRanking{}, &model.RankConfig{}, &model.UserOpponentStats{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare candidate query schema: %v", err)
	}
	users := []model.User{{Id: 1, Nickname: "我"}}
	friends := make([]model.Friend, 0, rows)
	rankings := make([]model.UserRanking, 0, rows)
	recent := make([]model.UserOpponentStats, 0, rows)
	now := time.Now()
	for index := 1; index <= rows; index++ {
		userID := int64(index + 1)
		users = append(users, model.User{Id: userID, Nickname: fmt.Sprintf("对手%d", index)})
		friends = append(friends, model.Friend{UserId: 1, FriendId: userID, Status: 1})
		rankings = append(rankings, model.UserRanking{UserId: userID, GameType: 3, RankLevel: 1})
		recent = append(recent, model.UserOpponentStats{UserId: 1, OpponentUserId: userID, OpponentNameKey: fmt.Sprintf("user:%d", userID), OpponentName: fmt.Sprintf("对手%d", index), GameType: 0, LastMatchAt: &now, LastMatchId: int64(index)})
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&model.RankConfig{Level: 1, Name: "新手"}).Error; err != nil {
		t.Fatalf("seed rank config: %v", err)
	}
	if err := db.Create(&friends).Error; err != nil {
		t.Fatalf("seed friends: %v", err)
	}
	if err := db.Create(&rankings).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	if err := db.Create(&recent).Error; err != nil {
		t.Fatalf("seed recent opponents: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, Revision: 5}).Error; err != nil {
		t.Fatalf("seed revision: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	requestDB := db.WithContext(ctx)
	resp, err := NewGetOpponentCandidatesLogic(ctx, &svc.ServiceContext{FriendModel: model.NewFriendModel(requestDB), CompetitiveReadModel: model.NewCompetitiveReadModel(requestDB), Config: config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}}}).GetOpponentCandidates(&types.GetOpponentCandidatesReq{Limit: 100})
	if err != nil || !resp.Success {
		t.Fatalf("get opponent candidates: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}
