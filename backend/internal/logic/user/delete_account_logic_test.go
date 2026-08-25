package user

import (
	"context"
	"sync"
	"testing"

	authlogic "chasing_points/internal/logic/auth"
	"chasing_points/internal/model"
	oauthverify "chasing_points/internal/pkg/oauth"
	"chasing_points/internal/sms"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDeleteAccountTestSvc(t *testing.T) (*svc.ServiceContext, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			phone TEXT,
			nickname TEXT NOT NULL DEFAULT '',
			avatar TEXT NOT NULL DEFAULT '',
			status INTEGER NOT NULL DEFAULT 1,
			push_token TEXT NOT NULL DEFAULT '',
			member_expires_at DATETIME,
			hide_match_record BOOLEAN NOT NULL DEFAULT false,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE user_oauth (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, provider TEXT NOT NULL, open_id TEXT NOT NULL, union_id TEXT, created_at DATETIME, UNIQUE(provider, open_id));
		CREATE TABLE matches (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, opponent_id INTEGER, opponent_name TEXT NOT NULL DEFAULT '', referee_user_id INTEGER, completed_by_user_id INTEGER, status INTEGER NOT NULL DEFAULT 1, end_time DATETIME, finish_state TEXT NOT NULL DEFAULT '', finish_requested_by INTEGER, finish_requested_at DATETIME, finish_request_revision INTEGER NOT NULL DEFAULT 0, completion_source TEXT NOT NULL DEFAULT '', remark TEXT NOT NULL DEFAULT '', sync_revision INTEGER NOT NULL DEFAULT 0, updated_at DATETIME, deleted_at DATETIME);
		CREATE TABLE social_posts (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE social_post_likes (id INTEGER PRIMARY KEY, post_id INTEGER NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE social_post_comments (id INTEGER PRIMARY KEY, post_id INTEGER NOT NULL, user_id INTEGER NOT NULL);
		CREATE TABLE friends (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, friend_id INTEGER NOT NULL);
		CREATE TABLE friend_requests (id INTEGER PRIMARY KEY, from_user_id INTEGER NOT NULL, to_user_id INTEGER NOT NULL);
		CREATE TABLE friend_blacklists (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, blocked_user_id INTEGER NOT NULL);
		CREATE TABLE follows (id INTEGER PRIMARY KEY, follower_id INTEGER NOT NULL, following_id INTEGER NOT NULL);
		CREATE TABLE challenges (id INTEGER PRIMARY KEY, from_user_id INTEGER NOT NULL, to_user_id INTEGER NOT NULL);
		CREATE TABLE notifications (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE user_notification_preferences (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE opponents (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, name TEXT NOT NULL DEFAULT '', avatar TEXT NOT NULL DEFAULT '', linked_user_id INTEGER);
		CREATE TABLE match_participant_results (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, opponent_user_id INTEGER NOT NULL DEFAULT 0, opponent_name_key TEXT NOT NULL DEFAULT '', opponent_name TEXT NOT NULL DEFAULT '', opponent_avatar TEXT NOT NULL DEFAULT '');
		CREATE TABLE user_competitive_stats (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE user_opponent_stats (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, opponent_user_id INTEGER NOT NULL DEFAULT 0, opponent_name_key TEXT NOT NULL DEFAULT '', opponent_name TEXT NOT NULL DEFAULT '', opponent_avatar TEXT NOT NULL DEFAULT '');
		CREATE TABLE user_opponent_strength_buckets (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE user_ranking (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE user_achievements (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE user_titles (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE achievement_progress_events (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE user_reputation_profiles (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE member_growth_profiles (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE member_growth_logs (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE favorite_venue_reward_records (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE venue_checkins (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL);
		CREATE TABLE verification_codes (id INTEGER PRIMARY KEY, phone TEXT NOT NULL);
		CREATE TABLE rank_change_logs (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, operator_user_id INTEGER NOT NULL DEFAULT 0, remark TEXT NOT NULL DEFAULT '');
		CREATE TABLE user_reputation_logs (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, reason_detail TEXT NOT NULL DEFAULT '');
		CREATE TABLE feedback_tickets (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, content TEXT NOT NULL DEFAULT '', contact TEXT NOT NULL DEFAULT '');
	`).Error; err != nil {
		t.Fatalf("prepare account deletion schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:                     db,
		UserModel:              model.NewUserModel(db),
		UserDataLifecycleModel: model.NewUserDataLifecycleModel(db),
		OauthModel:             model.NewUserOauthModel(db),
		MatchModel:             model.NewMatchModel(db),
	}, db
}

func accountDeletionCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedOAuthDeletionAccount(t *testing.T, svcCtx *svc.ServiceContext, db *gorm.DB) {
	t.Helper()
	if err := svcCtx.UserModel.Create(&model.User{Id: 101, Nickname: "待注销用户", Avatar: "avatar.png", PushToken: "push-token", Status: 1}); err != nil {
		t.Fatalf("create deletion user: %v", err)
	}
	if err := svcCtx.UserModel.Create(&model.User{Id: 202, Nickname: "保留用户", Status: 1}); err != nil {
		t.Fatalf("create related user: %v", err)
	}
	if err := svcCtx.OauthModel.Create(&model.UserOauth{UserId: 101, Provider: "huawei", OpenId: "delete-subject"}); err != nil {
		t.Fatalf("create oauth association: %v", err)
	}
	for _, statement := range []struct {
		query string
		args  []any
	}{
		{"INSERT INTO matches (id, user_id, opponent_id, opponent_name, status, finish_state, completion_source, remark, sync_revision) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", []any{301, 202, 101, "待注销用户", 1, "none", "unknown", "private note", 5}},
		{"INSERT INTO social_posts (id, user_id) VALUES (?, ?)", []any{401, 101}},
		{"INSERT INTO social_post_likes (id, post_id, user_id) VALUES (?, ?, ?)", []any{402, 401, 202}},
		{"INSERT INTO social_post_comments (id, post_id, user_id) VALUES (?, ?, ?)", []any{403, 401, 202}},
		{"INSERT INTO friends (id, user_id, friend_id) VALUES (?, ?, ?)", []any{404, 101, 202}},
		{"INSERT INTO friend_requests (id, from_user_id, to_user_id) VALUES (?, ?, ?)", []any{405, 101, 202}},
		{"INSERT INTO friend_blacklists (id, user_id, blocked_user_id) VALUES (?, ?, ?)", []any{406, 202, 101}},
		{"INSERT INTO follows (id, follower_id, following_id) VALUES (?, ?, ?)", []any{407, 202, 101}},
		{"INSERT INTO challenges (id, from_user_id, to_user_id) VALUES (?, ?, ?)", []any{408, 101, 202}},
		{"INSERT INTO notifications (id, user_id) VALUES (?, ?)", []any{409, 101}},
		{"INSERT INTO user_notification_preferences (id, user_id) VALUES (?, ?)", []any{410, 101}},
		{"INSERT INTO opponents (id, user_id, name, avatar, linked_user_id) VALUES (?, ?, ?, ?, ?)", []any{411, 202, "待注销用户", "avatar.png", 101}},
		{"INSERT INTO match_participant_results (id, user_id, opponent_user_id, opponent_name_key, opponent_name, opponent_avatar) VALUES (?, ?, ?, ?, ?, ?)", []any{412, 202, 101, "user:101", "待注销用户", "avatar.png"}},
		{"INSERT INTO match_participant_results (id, user_id, opponent_user_id) VALUES (?, ?, ?)", []any{413, 101, 202}},
		{"INSERT INTO user_competitive_stats (id, user_id) VALUES (?, ?)", []any{414, 101}},
		{"INSERT INTO user_opponent_stats (id, user_id, opponent_user_id, opponent_name_key, opponent_name, opponent_avatar) VALUES (?, ?, ?, ?, ?, ?)", []any{415, 202, 101, "user:101", "待注销用户", "avatar.png"}},
		{"INSERT INTO user_opponent_stats (id, user_id, opponent_user_id) VALUES (?, ?, ?)", []any{416, 101, 202}},
		{"INSERT INTO user_opponent_strength_buckets (id, user_id) VALUES (?, ?)", []any{417, 101}},
		{"INSERT INTO user_ranking (id, user_id) VALUES (?, ?)", []any{418, 101}},
		{"INSERT INTO user_achievements (id, user_id) VALUES (?, ?)", []any{419, 101}},
		{"INSERT INTO user_titles (id, user_id) VALUES (?, ?)", []any{420, 101}},
		{"INSERT INTO achievement_progress_events (id, user_id) VALUES (?, ?)", []any{421, 101}},
		{"INSERT INTO user_reputation_profiles (id, user_id) VALUES (?, ?)", []any{422, 101}},
		{"INSERT INTO member_growth_profiles (id, user_id) VALUES (?, ?)", []any{423, 101}},
		{"INSERT INTO member_growth_logs (id, user_id) VALUES (?, ?)", []any{424, 101}},
		{"INSERT INTO favorite_venue_reward_records (id, user_id) VALUES (?, ?)", []any{425, 101}},
		{"INSERT INTO venue_checkins (id, user_id) VALUES (?, ?)", []any{426, 101}},
		{"INSERT INTO rank_change_logs (id, user_id, operator_user_id, remark) VALUES (?, ?, ?, ?)", []any{427, 101, 101, "包含身份信息"}},
		{"INSERT INTO user_reputation_logs (id, user_id, reason_detail) VALUES (?, ?, ?)", []any{428, 101, "包含身份信息"}},
		{"INSERT INTO feedback_tickets (id, user_id, content, contact) VALUES (?, ?, ?, ?)", []any{429, 101, "反馈正文", "13800138000"}},
	} {
		if err := db.Exec(statement.query, statement.args...).Error; err != nil {
			t.Fatalf("seed deletion data: %v", err)
		}
	}
}

func TestDeleteAccountOAuthClearsPersonalDataAndAnonymizesFacts(t *testing.T) {
	svcCtx, db := newDeleteAccountTestSvc(t)
	seedOAuthDeletionAccount(t, svcCtx, db)
	svcCtx.OAuthVerifier = oauthverify.StaticVerifier{Identity: &oauthverify.Identity{Provider: "huawei", Subject: "delete-subject"}}

	resp, err := NewDeleteAccountLogic(accountDeletionCtx(101), svcCtx).DeleteAccount(&types.DeleteAccountReq{
		ConfirmText:    deleteAccountConfirmText,
		VerifyType:     "oauth",
		Provider:       "huawei",
		Credential:     "fresh-provider-credential",
		CredentialType: "authorization_code",
	})
	if err != nil || !resp.Success {
		t.Fatalf("delete account: resp=%#v err=%v", resp, err)
	}

	var deletedUser model.User
	if err := db.Unscoped().First(&deletedUser, 101).Error; err != nil {
		t.Fatalf("find deleted user: %v", err)
	}
	if !deletedUser.DeletedAt.Valid || deletedUser.Status != 0 || deletedUser.Phone != nil || deletedUser.Nickname != model.DeletedUserDisplayName || deletedUser.Avatar != "" || deletedUser.PushToken != "" {
		t.Fatalf("user was not anonymized and soft-deleted: %#v", deletedUser)
	}
	if oauth, findErr := svcCtx.OauthModel.FindByProviderAndOpenId("huawei", "delete-subject"); findErr != nil || oauth != nil {
		t.Fatalf("oauth association must be deleted: oauth=%#v err=%v", oauth, findErr)
	}

	assertDeletedRows(t, db, "social_posts", "user_id = ?", 101)
	assertDeletedRows(t, db, "friends", "user_id = ? OR friend_id = ?", 101, 101)
	assertDeletedRows(t, db, "match_participant_results", "user_id = ?", 101)
	assertDeletedRows(t, db, "user_competitive_stats", "user_id = ?", 101)
	assertDeletedRows(t, db, "user_ranking", "user_id = ?", 101)

	var match struct {
		Status       int
		OpponentName string
		Remark       string
	}
	if err := db.Table("matches").Select("status, opponent_name, remark").Where("id = ?", 301).Scan(&match).Error; err != nil {
		t.Fatalf("read cancelled match: %v", err)
	}
	if match.Status != 3 || match.OpponentName != model.DeletedUserDisplayName || match.Remark != "account_deleted" {
		t.Fatalf("active match was not closed and anonymized: %#v", match)
	}

	var opponent struct {
		Name         string
		Avatar       string
		LinkedUserID *int64
	}
	if err := db.Table("opponents").Where("id = ?", 411).Scan(&opponent).Error; err != nil {
		t.Fatalf("read anonymized opponent: %v", err)
	}
	if opponent.Name != model.DeletedUserDisplayName || opponent.Avatar != "" || opponent.LinkedUserID != nil {
		t.Fatalf("opponent snapshot was not anonymized: %#v", opponent)
	}

	var projection struct {
		OpponentName   string
		OpponentAvatar string
	}
	if err := db.Table("match_participant_results").Where("id = ?", 412).Scan(&projection).Error; err != nil {
		t.Fatalf("read match projection: %v", err)
	}
	if projection.OpponentName != model.DeletedUserDisplayName || projection.OpponentAvatar != "" {
		t.Fatalf("match projection was not anonymized: %#v", projection)
	}

	var feedback struct {
		Content string
		Contact string
	}
	if err := db.Table("feedback_tickets").Where("id = ?", 429).Scan(&feedback).Error; err != nil {
		t.Fatalf("read feedback ticket: %v", err)
	}
	if feedback.Content != model.DeletedUserDisplayName || feedback.Contact != "" {
		t.Fatalf("feedback ticket was not anonymized: %#v", feedback)
	}

	secondResp, secondErr := NewDeleteAccountLogic(accountDeletionCtx(101), svcCtx).DeleteAccount(&types.DeleteAccountReq{ConfirmText: deleteAccountConfirmText})
	if secondErr != nil || !secondResp.Success {
		t.Fatalf("repeated deletion must converge to success: resp=%#v err=%v", secondResp, secondErr)
	}

	svcCtx.Config.Auth.AccessSecret = "delete-account-test-secret"
	svcCtx.Config.Auth.AccessExpire = 3600
	loginResp, loginErr := authlogic.NewLoginByOauthLogic(context.Background(), svcCtx).LoginByOauth(&types.LoginByOauthReq{
		Provider:       "huawei",
		Credential:     "new-provider-credential",
		CredentialType: "authorization_code",
	})
	if loginErr != nil || !loginResp.Success || loginResp.UserInfo == nil || loginResp.UserInfo.Id == 101 {
		t.Fatalf("old OAuth must not restore the deleted account: resp=%#v err=%v", loginResp, loginErr)
	}
}

func TestDeleteAccountRejectsFailedOAuthAndKeepsData(t *testing.T) {
	svcCtx, db := newDeleteAccountTestSvc(t)
	seedOAuthDeletionAccount(t, svcCtx, db)
	svcCtx.OAuthVerifier = oauthverify.StaticVerifier{Err: oauthverify.NewVerifyError(oauthverify.ErrorRejected, "credential expired", nil)}

	resp, err := NewDeleteAccountLogic(accountDeletionCtx(101), svcCtx).DeleteAccount(&types.DeleteAccountReq{
		ConfirmText: deleteAccountConfirmText,
		VerifyType:  "oauth",
		Provider:    "huawei",
		Credential:  "expired-provider-credential",
	})
	if err != nil || resp.Success {
		t.Fatalf("failed oauth reauthentication must reject deletion: resp=%#v err=%v", resp, err)
	}
	if user, findErr := svcCtx.UserModel.FindById(101); findErr != nil || user == nil {
		t.Fatalf("failed reauthentication must retain user: user=%#v err=%v", user, findErr)
	}
	if oauth, findErr := svcCtx.OauthModel.FindByProviderAndOpenId("huawei", "delete-subject"); findErr != nil || oauth == nil {
		t.Fatalf("failed reauthentication must retain oauth: oauth=%#v err=%v", oauth, findErr)
	}
	var status int
	if err := db.Table("matches").Select("status").Where("id = ?", 301).Scan(&status).Error; err != nil || status != 1 {
		t.Fatalf("failed reauthentication must retain active match: status=%d err=%v", status, err)
	}
}

func TestDeleteAccountRejectsWrongSmsCodeWithoutChangingData(t *testing.T) {
	svcCtx, db := newDeleteAccountTestSvc(t)
	phone := "13800138000"
	if err := svcCtx.UserModel.Create(&model.User{Id: 501, Phone: &phone, Nickname: "手机号用户", Status: 1}); err != nil {
		t.Fatalf("create phone user: %v", err)
	}
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })
	svcCtx.CodeManager = sms.NewCodeManager(redisClient)
	if err := svcCtx.CodeManager.SaveCodeForScene(context.Background(), phone, sms.SceneDeleteAccount, "123456"); err != nil {
		t.Fatalf("save delete-account sms code: %v", err)
	}

	resp, err := NewDeleteAccountLogic(accountDeletionCtx(501), svcCtx).DeleteAccount(&types.DeleteAccountReq{
		ConfirmText: deleteAccountConfirmText,
		VerifyType:  "sms",
		SmsCode:     "654321",
	})
	if err != nil || resp.Success {
		t.Fatalf("wrong sms code must reject deletion: resp=%#v err=%v", resp, err)
	}
	if user, findErr := svcCtx.UserModel.FindById(501); findErr != nil || user == nil || user.Phone == nil || *user.Phone != phone {
		t.Fatalf("wrong sms code must retain user: user=%#v err=%v", user, findErr)
	}
	assertDeletedRows(t, db, "user_oauth", "user_id = ?", 501)
}

func TestDeleteAccountConcurrentTransactionsConvergeOnOneDeletedAccount(t *testing.T) {
	svcCtx, db := newDeleteAccountTestSvc(t)
	if err := svcCtx.UserModel.Create(&model.User{Id: 777, Nickname: "并发注销用户", Status: 1}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	logic := NewDeleteAccountLogic(accountDeletionCtx(777), svcCtx)
	type deletionResult struct {
		alreadyDeleted bool
		err            error
	}
	results := make(chan deletionResult, 2)
	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			_, alreadyDeleted, deleteErr := logic.deleteAccountInTransaction(777, nil)
			results <- deletionResult{alreadyDeleted: alreadyDeleted, err: deleteErr}
		}()
	}
	close(start)
	waitGroup.Wait()
	close(results)

	var deleted, repeated int
	for result := range results {
		if result.err == nil && !result.alreadyDeleted {
			deleted++
			continue
		}
		if result.alreadyDeleted && result.err == errAccountDeletionAlreadyClosed {
			repeated++
			continue
		}
		t.Fatalf("unexpected concurrent deletion result: %+v", result)
	}
	if deleted != 1 || repeated != 1 {
		t.Fatalf("concurrent deletions must converge once: deleted=%d repeated=%d", deleted, repeated)
	}
}

func assertDeletedRows(t *testing.T, db *gorm.DB, table, where string, args ...any) {
	t.Helper()
	var count int64
	if err := db.Table(table).Where(where, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != 0 {
		t.Fatalf("expected deleted rows from %s, got %d", table, count)
	}
}
