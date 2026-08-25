package user

import (
	"context"
	"strings"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPersonalDataExportTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open export db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.UserOauth{}, &model.UserNotificationPreference{},
		&model.Friend{}, &model.FriendRequest{}, &model.FriendBlacklist{}, &model.Follow{}, &model.Challenge{},
		&model.Notification{}, &model.SocialPost{}, &model.SocialPostComment{}, &model.SocialPostLike{},
		&model.Match{}, &model.Opponent{}, &model.UserCompetitiveStats{}, &model.UserOpponentStats{},
		&model.UserAchievement{}, &model.UserTitle{}, &model.AchievementProgressEvent{},
		&model.UserRanking{}, &model.RankChangeLog{}, &model.UserReputationProfile{}, &model.UserReputationLog{},
		&model.VenueCheckin{}, &model.MemberSubscriptionOrder{}, &model.MemberGrowthProfile{}, &model.MemberGrowthLog{},
		&model.FavoriteVenueRewardRecord{}, &model.FeedbackTicket{}, &model.TournamentParticipant{},
		&model.SeasonRecord{}, &model.SeasonChallengeSnapshot{},
	); err != nil {
		t.Fatalf("prepare export schema: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get export sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	return &svc.ServiceContext{
		Config:                 config.Config{Security: config.SecurityConfig{MatchInvite: config.MatchInviteConfig{SigningSecret: "export-test-secret"}}},
		DB:                     db,
		UserModel:              model.NewUserModel(db),
		UserDataLifecycleModel: model.NewUserDataLifecycleModel(db),
	}
}

func personalDataExportCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestExportPersonalDataUsesBoundedSignedPagesAndExcludesSecrets(t *testing.T) {
	svcCtx := newPersonalDataExportTestSvc(t)
	phone := "13800138000"
	if err := svcCtx.UserModel.Create(&model.User{
		Id:        101,
		Phone:     &phone,
		Nickname:  "导出用户",
		Avatar:    "avatar.png",
		PushToken: "push-token-must-not-export",
		Status:    1,
	}); err != nil {
		t.Fatalf("create export user: %v", err)
	}
	if err := svcCtx.UserModel.Create(&model.User{Id: 202, Nickname: "对手", Status: 1}); err != nil {
		t.Fatalf("create opponent user: %v", err)
	}
	if err := svcCtx.DB.Create(&model.UserOauth{UserId: 101, Provider: "huawei", OpenId: "oauth-open-id-abcdef"}).Error; err != nil {
		t.Fatalf("create oauth: %v", err)
	}
	if err := svcCtx.DB.Create(&model.UserNotificationPreference{UserId: 101, MatchResultEnabled: true}).Error; err != nil {
		t.Fatalf("create notification preference: %v", err)
	}
	for index := 1; index <= model.PersonalDataExportMaxBatchSize+5; index++ {
		if err := svcCtx.DB.Create(&model.Friend{UserId: 101, FriendId: int64(1000 + index), Status: 1}).Error; err != nil {
			t.Fatalf("create friend %d: %v", index, err)
		}
	}
	if err := svcCtx.DB.Create(&model.Match{
		UserId:       101,
		OpponentId:   pointerToInt64(202),
		OpponentName: "对手",
		GameType:     3,
		MatchMode:    model.MatchModePractice,
		Visibility:   model.MatchVisibilityPrivate,
		Status:       2,
		MatchTime:    time.Now().Add(-time.Hour),
	}).Error; err != nil {
		t.Fatalf("create match: %v", err)
	}
	if err := svcCtx.DB.Create(&model.SocialPost{UserId: 101, Content: "我的动态", Status: model.SocialPostStatusPublished}).Error; err != nil {
		t.Fatalf("create social post: %v", err)
	}

	logic := NewExportPersonalDataLogic(personalDataExportCtx(101), svcCtx)
	first, err := logic.ExportPersonalData(&types.PersonalDataExportReq{})
	if err != nil || !first.Success || first.Complete || first.NextCursor == "" {
		t.Fatalf("expected first bounded export page: resp=%+v err=%v", first, err)
	}
	if len(first.Items) > model.PersonalDataExportMaxBatchSize || first.ItemCount != len(first.Items) {
		t.Fatalf("invalid first page bounds: count=%d itemCount=%d", len(first.Items), first.ItemCount)
	}

	otherUserLogic := NewExportPersonalDataLogic(personalDataExportCtx(202), svcCtx)
	crossUser, err := otherUserLogic.ExportPersonalData(&types.PersonalDataExportReq{Cursor: first.NextCursor})
	if err != nil || crossUser.Success {
		t.Fatalf("cursor must be bound to its user: resp=%+v err=%v", crossUser, err)
	}
	tampered, err := logic.ExportPersonalData(&types.PersonalDataExportReq{Cursor: first.NextCursor + "x"})
	if err != nil || tampered.Success {
		t.Fatalf("tampered cursor must be rejected: resp=%+v err=%v", tampered, err)
	}

	seenCategories := map[string]bool{}
	page := first
	for requestCount := 0; requestCount < 64; requestCount++ {
		if page.FormatVersion != personalDataExportFormatVersion || page.SnapshotAt == "" || page.ItemCount != len(page.Items) {
			t.Fatalf("invalid export metadata: %+v", page)
		}
		if len(page.Items) > model.PersonalDataExportMaxBatchSize {
			t.Fatalf("page exceeds bounded export size: %d", len(page.Items))
		}
		for _, item := range page.Items {
			seenCategories[item.Category] = true
			for _, forbidden := range []string{"push-token-must-not-export", "oauth-open-id-abcdef", "open_id", "union_id", "refresh_token"} {
				if strings.Contains(item.DataJson, forbidden) {
					t.Fatalf("export leaked forbidden field/value %q in %+v", forbidden, item)
				}
			}
		}
		if page.Complete {
			break
		}
		page, err = logic.ExportPersonalData(&types.PersonalDataExportReq{Cursor: page.NextCursor})
		if err != nil || !page.Success {
			t.Fatalf("read next export page: resp=%+v err=%v", page, err)
		}
		if requestCount == 63 {
			t.Fatal("export did not complete within bounded category/page count")
		}
	}
	for _, category := range []string{"profile", "authentication_connections", "notification_preferences", "friends", "social_posts", "matches"} {
		if !seenCategories[category] {
			t.Fatalf("expected export category %q, got %v", category, seenCategories)
		}
	}
}

func TestExportPersonalDataHandlesEmptyCategoryAndUnavailableDependencies(t *testing.T) {
	svcCtx := newPersonalDataExportTestSvc(t)
	if err := svcCtx.UserModel.Create(&model.User{Id: 303, Nickname: "空数据用户", Status: 1}); err != nil {
		t.Fatalf("create user: %v", err)
	}
	records, hasMore, err := svcCtx.UserDataLifecycleModel.ListExportRecords(303, "friends", 0, time.Now(), model.PersonalDataExportMaxBatchSize)
	if err != nil || hasMore || len(records) != 0 {
		t.Fatalf("empty category must be a bounded empty result: records=%+v hasMore=%v err=%v", records, hasMore, err)
	}

	unavailable := NewExportPersonalDataLogic(personalDataExportCtx(303), &svc.ServiceContext{})
	resp, err := unavailable.ExportPersonalData(&types.PersonalDataExportReq{})
	if err != nil || resp.Success || resp.Message != "导出服务暂不可用" {
		t.Fatalf("missing dependencies must fail without export: resp=%+v err=%v", resp, err)
	}
}

func pointerToInt64(value int64) *int64 {
	return &value
}
