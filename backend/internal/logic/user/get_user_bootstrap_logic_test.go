package user

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/config"
	logicx "chasing_points/internal/logic"
	matchlogic "chasing_points/internal/logic/match"
	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserBootstrapReturnsSharedActivityAndCompetitiveRevision(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.Notification{}, &model.FriendRequest{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare bootstrap schema: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:                   db,
		UserModel:            model.NewUserModel(db),
		MatchModel:           model.NewMatchModel(db),
		NotificationModel:    model.NewNotificationModel(db),
		FriendModel:          model.NewFriendModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "我"}, {Id: 2, Nickname: "对手"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	opponentID := int64(2)
	if err := db.Create(&model.Match{Id: 10, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, Status: 1, MatchTime: time.Now().Add(-time.Minute)}).Error; err != nil {
		t.Fatalf("seed current match: %v", err)
	}
	if err := db.Create(&model.Notification{Id: 1, UserId: 1, Type: "season_rollover", Title: "S2 已开启", Content: "换季", IsRead: 0}).Error; err != nil {
		t.Fatalf("seed notification: %v", err)
	}
	if err := db.Create(&model.FriendRequest{Id: 1, FromUserId: 2, ToUserId: 1, Status: 0}).Error; err != nil {
		t.Fatalf("seed friend request: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, Revision: 7}).Error; err != nil {
		t.Fatalf("seed competitive revision: %v", err)
	}

	resp, err := NewGetUserBootstrapLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetUserBootstrap()
	if err != nil || !resp.Success || resp.UserInfo == nil || resp.CurrentMatch == nil {
		t.Fatalf("get bootstrap: resp=%#v err=%v", resp, err)
	}
	if resp.UserInfo.Nickname != "我" || resp.CurrentMatch.Id != 10 || resp.UnreadCount != 1 || resp.PendingFriendRequestCount != 1 || resp.LatestSeasonRollover == nil || resp.CompetitiveRevision != 7 {
		t.Fatalf("unexpected bootstrap payload: %+v", resp)
	}
	for _, scope := range []string{"user", "current_match", "unread_count", "pending_friend_request_count", "latest_season_rollover", "competitive_revision"} {
		if !resp.Availability[scope] {
			t.Fatalf("expected available scope %q: %+v", scope, resp.Availability)
		}
	}
}

func TestUserBootstrapRejectsMissingAuthentication(t *testing.T) {
	resp, err := NewGetUserBootstrapLogic(context.Background(), &svc.ServiceContext{}).GetUserBootstrap()
	if err != nil || resp.Success || resp.UserInfo != nil {
		t.Fatalf("unauthenticated bootstrap must not return user data: resp=%#v err=%v", resp, err)
	}
}

func TestUserBootstrapConvergesAfterCommittedFinishWhenUserWebSocketIsUnavailable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{},
		&model.UserRanking{}, &model.RankChangeLog{}, &model.Notification{},
		&model.MatchParticipantResult{}, &model.UserCompetitiveStats{},
		&model.UserOpponentStats{}, &model.UserOpponentStrengthBucket{},
	); err != nil {
		t.Fatalf("prepare settlement recovery schema: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:                   db,
		UserModel:            model.NewUserModel(db),
		MatchModel:           model.NewMatchModel(db),
		RankingModel:         model.NewRankingModel(db),
		NotificationModel:    model.NewNotificationModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	opponentID := int64(2)
	if err := svcCtx.MatchModel.Create(&model.Match{Id: 100, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 3, MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, FinishState: model.FinishStateNone, Status: 1, MyScore: 3, OpponentScore: 1, CurrentFrameStarted: true, MatchTime: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatalf("seed match: %v", err)
	}
	previousHub := ws.GlobalHub
	ws.GlobalHub = nil
	t.Cleanup(func() { ws.GlobalHub = previousHub })
	ctx := context.WithValue(context.Background(), "user_id", int64(1))
	finish, err := matchlogic.NewFinishMatchLogic(ctx, svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: 100, ClientActionId: "finish-without-user-ws", BaseRevision: 0})
	if err != nil || !finish.Success {
		t.Fatalf("finish match without user WS: resp=%#v err=%v", finish, err)
	}
	resp, err := NewGetUserBootstrapLogic(ctx, svcCtx).GetUserBootstrap()
	if err != nil || !resp.Success || !resp.Availability["competitive_revision"] || resp.CompetitiveRevision != 1 {
		t.Fatalf("bootstrap must converge from committed revision: resp=%#v err=%v", resp, err)
	}
}

func TestUserBootstrapReadsCompetitiveRevisionWhenUserEventIsUnavailable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare bootstrap schema: %v", err)
	}
	if err := db.Create(&model.User{Id: 1, Nickname: "我"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, Revision: 9}).Error; err != nil {
		t.Fatalf("seed revision: %v", err)
	}
	previousHub := ws.GlobalHub
	ws.GlobalHub = nil
	t.Cleanup(func() { ws.GlobalHub = previousHub })
	logicx.SendUserDataUpdated(1, logicx.UserDataUpdatedEvent{CompetitiveRevision: 9})

	resp, err := NewGetUserBootstrapLogic(context.WithValue(context.Background(), "user_id", int64(1)), &svc.ServiceContext{
		UserModel:            model.NewUserModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}).GetUserBootstrap()
	if err != nil || !resp.Success || !resp.Availability["competitive_revision"] || resp.CompetitiveRevision != 9 {
		t.Fatalf("bootstrap must converge from committed snapshot without user WS: resp=%#v err=%v", resp, err)
	}
}

func TestUserBootstrapUsesFixedQueryCountForOneAndHundredRows(t *testing.T) {
	one := userBootstrapQueryCount(t, 1)
	hundred := userBootstrapQueryCount(t, 100)
	if one != 6 || hundred != 6 {
		t.Fatalf("bootstrap must use six fixed queries: one=%d hundred=%d", one, hundred)
	}
}

func userBootstrapQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.Notification{}, &model.FriendRequest{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare bootstrap query schema: %v", err)
	}
	if err := db.Create(&model.User{Id: 1, Nickname: "我"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	matches := make([]model.Match, 0, rows)
	notifications := make([]model.Notification, 0, rows)
	requests := make([]model.FriendRequest, 0, rows)
	for index := 1; index <= rows; index++ {
		matches = append(matches, model.Match{Id: int64(index), UserId: 1, GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, MatchTime: time.Now().Add(-time.Duration(index) * time.Minute)})
		notifications = append(notifications, model.Notification{Id: int64(index), UserId: 1, Type: "system", Title: "通知", IsRead: index % 2})
		requests = append(requests, model.FriendRequest{Id: int64(index), FromUserId: int64(1000 + index), ToUserId: 1, Status: index % 2})
	}
	if err := db.Create(&matches).Error; err != nil {
		t.Fatalf("seed matches: %v", err)
	}
	if err := db.Create(&notifications).Error; err != nil {
		t.Fatalf("seed notifications: %v", err)
	}
	if err := db.Create(&requests).Error; err != nil {
		t.Fatalf("seed friend requests: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, Revision: 3}).Error; err != nil {
		t.Fatalf("seed revision: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	requestDB := db.WithContext(ctx)
	resp, err := NewGetUserBootstrapLogic(ctx, &svc.ServiceContext{
		DB:                   requestDB,
		UserModel:            model.NewUserModel(requestDB),
		MatchModel:           model.NewMatchModel(requestDB),
		NotificationModel:    model.NewNotificationModel(requestDB),
		FriendModel:          model.NewFriendModel(requestDB),
		CompetitiveReadModel: model.NewCompetitiveReadModel(requestDB),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}).GetUserBootstrap()
	if err != nil || !resp.Success {
		t.Fatalf("get bootstrap: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}

func TestUserBootstrapKeepsUserAvailableWhenOptionalModelsAreMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("prepare user schema: %v", err)
	}
	if err := db.Create(&model.User{Id: 1, Nickname: "我"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	resp, err := NewGetUserBootstrapLogic(context.WithValue(context.Background(), "user_id", int64(1)), &svc.ServiceContext{UserModel: model.NewUserModel(db)}).GetUserBootstrap()
	if err != nil || !resp.Success || resp.UserInfo == nil || !resp.Availability["user"] {
		t.Fatalf("user must remain available when optional blocks fail: resp=%#v err=%v", resp, err)
	}
	if len(resp.PartialErrors) != 5 {
		t.Fatalf("expected all five optional blocks marked partial: %+v", resp.PartialErrors)
	}
}
