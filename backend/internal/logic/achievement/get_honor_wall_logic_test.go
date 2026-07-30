package achievement

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

func newHonorWallTestSvc(t *testing.T) (*svc.ServiceContext, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Friend{},
		&model.FriendBlacklist{},
		&model.Achievement{},
		&model.UserAchievement{},
		&model.UserTitle{},
		&model.AchievementProgressEvent{},
		&model.Season{},
		&model.SeasonChallengeSnapshot{},
	); err != nil {
		t.Fatalf("prepare honor wall schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:                            db,
		UserModel:                     model.NewUserModel(db),
		FriendModel:                   model.NewFriendModel(db),
		AchievementModel:              model.NewAchievementModel(db),
		UserAchievementModel:          model.NewUserAchievementModel(db),
		UserTitleModel:                model.NewUserTitleModel(db),
		AchievementProgressEventModel: model.NewAchievementProgressEventModel(db),
		SeasonModel:                   model.NewSeasonModel(db),
		SeasonChallengeSnapshotModel:  model.NewSeasonChallengeSnapshotModel(db),
	}, db
}

func honorWallContext(userId int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userId)
}

func TestBuildRecentHonorsSortsLimitsAndDeduplicatesAchievementTitle(t *testing.T) {
	base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	definitions := []model.Achievement{
		{Id: 1, Name: "十胜起步", RewardTitleName: "胜场新星"},
		{Id: 2, Name: "初入战局"},
	}
	userAchievements := []model.UserAchievement{
		{AchievementId: 1, Unlocked: 1, UnlockedAt: timePointer(base.Add(3 * time.Hour))},
		{AchievementId: 2, Unlocked: 1, UnlockedAt: timePointer(base.Add(time.Hour))},
	}
	titles := []model.UserTitle{
		{Id: 11, TitleName: "胜场新星", SourceType: SourceTypeAchievement, SourceRefId: 1, GrantedAt: timePointer(base.Add(5 * time.Hour))},
		{Id: 12, TitleName: "S3 赛季前十", SourceType: SourceTypeSeason, SourceRefId: 3, GrantedAt: timePointer(base.Add(4 * time.Hour))},
		{Id: 13, TitleName: "城市杯冠军", SourceType: SourceTypeTournament, SourceRefId: 8, GrantedAt: timePointer(base.Add(6 * time.Hour))},
	}

	list := buildRecentHonors(userAchievements, definitions, titles)
	if len(list) != 3 {
		t.Fatalf("expected three recent honors, got %d", len(list))
	}
	if list[0].Name != "城市杯冠军" || list[1].Name != "S3 赛季前十" || list[2].Name != "十胜起步" {
		t.Fatalf("unexpected recent honor order: %+v", list)
	}
	if list[2].RewardTitleName != "胜场新星" {
		t.Fatalf("expected reward title merged into achievement item: %+v", list[2])
	}
	for _, item := range list {
		if item.Id == 11 {
			t.Fatal("achievement title should not occupy a separate recent honor slot")
		}
	}
}

func TestGetHonorWallSelfIncludesLockedProgressCurrentSeasonAndHistory(t *testing.T) {
	svcCtx, db := newHonorWallTestSvc(t)
	seedHonorWallUser(t, svcCtx, 1001, "本人")
	definitions := seedHonorWallAchievements(t, db)
	base := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	seedHonorWallUnlocked(t, db, 1001, definitions[0].Id, 10, base)
	seedHonorWallTitle(t, db, model.UserTitle{UserId: 1001, TitleKey: "season_2", TitleName: "S2 前十", Source: SourceTypeSeason, SourceType: SourceTypeSeason, SourceRefId: 2, SourceRefName: "S2", Equipped: 1, GrantedAt: timePointer(base)})

	current := model.Season{Id: 3, Name: "S3", StartDate: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), Status: 1}
	history := model.Season{Id: 2, Name: "S2", StartDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), Status: 2}
	if err := db.Create(&[]model.Season{current, history}).Error; err != nil {
		t.Fatalf("seed seasons: %v", err)
	}
	if _, err := svcCtx.AchievementProgressEventModel.CreateIfAbsent(model.NewAchievementProgressEvent(1001, SourceTypeMatch, 77, 3, MetricMatchesTotal, 8, current.StartDate.Add(time.Hour))); err != nil {
		t.Fatalf("seed current progress: %v", err)
	}
	archivedAt := history.EndDate.AddDate(0, 0, 1)
	if err := svcCtx.SeasonChallengeSnapshotModel.UpsertBatchWithTx(nil, []model.SeasonChallengeSnapshot{{SeasonId: 2, UserId: 1001, GameType: 3, ChallengeKey: SeasonChallengeMatchesKey, ChallengeName: "赛季常客", Threshold: 20, Progress: 7, ArchivedAt: archivedAt}}); err != nil {
		t.Fatalf("seed archived challenge: %v", err)
	}

	resp, err := NewGetHonorWallLogic(honorWallContext(1001), svcCtx).GetHonorWall(&types.GetHonorWallReq{HistorySeasonId: 2})
	if err != nil {
		t.Fatalf("get self honor wall: %v", err)
	}
	if !resp.Success || resp.ViewerScope != "self" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if len(resp.CareerAchievements) != 2 || resp.Summary.CareerUnlocked != 1 || resp.Summary.CareerTotal != 2 {
		t.Fatalf("unexpected career data: %+v", resp)
	}
	if resp.Summary.UniversalUnlocked != 1 || resp.Summary.UniversalTotal != 1 || resp.Summary.SpecialtyGameType != 3 || resp.Summary.SpecialtyUnlocked != 0 || resp.Summary.SpecialtyTotal != 1 {
		t.Fatalf("unexpected split career summary: %+v", resp.Summary)
	}
	if resp.CareerAchievements[0].GameType != 0 || resp.CareerAchievements[0].RewardTitleName != "胜场新星" || resp.CareerAchievements[1].GameType != 3 {
		t.Fatalf("expected career metadata in payload: %+v", resp.CareerAchievements)
	}
	if resp.EquippedTitle == nil || resp.EquippedTitle.SourceRefName != "S2" {
		t.Fatalf("expected equipped title source details: %+v", resp.EquippedTitle)
	}
	if resp.CurrentSeason == nil || resp.CurrentSeason.GameType != 3 || len(resp.CurrentSeason.Challenges) != 3 {
		t.Fatalf("unexpected current season: %+v", resp.CurrentSeason)
	}
	if resp.CurrentSeason.Challenges[0].Progress != 8 {
		t.Fatalf("expected current match progress 8: %+v", resp.CurrentSeason.Challenges)
	}
	if resp.History.ChallengeSeason == nil || len(resp.History.ChallengeRecords) != 1 || resp.History.ChallengeRecords[0].Progress != 7 {
		t.Fatalf("unexpected archived challenge data: %+v", resp.History)
	}
}

func TestGetHonorWallFriendOnlyReceivesUnlockedAndPermanentHonors(t *testing.T) {
	svcCtx, db := newHonorWallTestSvc(t)
	seedHonorWallUser(t, svcCtx, 1001, "查看者")
	seedHonorWallUser(t, svcCtx, 2002, "好友")
	if err := svcCtx.FriendModel.AddFriend(1001, 2002); err != nil {
		t.Fatalf("add friend: %v", err)
	}
	definitions := seedHonorWallAchievements(t, db)
	seedHonorWallUnlocked(t, db, 2002, definitions[0].Id, 10, time.Now())
	seedHonorWallTitle(t, db, model.UserTitle{UserId: 2002, TitleKey: "season_2", TitleName: "S2 前十", SourceType: SourceTypeSeason, SourceRefId: 2, SourceRefName: "S2", GrantedAt: timePointer(time.Now())})
	if err := db.Create(&model.Season{Id: 3, Name: "S3", StartDate: time.Now().AddDate(0, 0, -3), EndDate: time.Now().AddDate(0, 0, 20), Status: 1}).Error; err != nil {
		t.Fatalf("seed current season: %v", err)
	}

	resp, err := NewGetHonorWallLogic(honorWallContext(1001), svcCtx).GetHonorWall(&types.GetHonorWallReq{UserId: 2002, HistorySeasonId: 2})
	if err != nil {
		t.Fatalf("get friend honor wall: %v", err)
	}
	if !resp.Success || resp.ViewerScope != "friend" {
		t.Fatalf("unexpected friend response: %+v", resp)
	}
	if len(resp.CareerAchievements) != 1 || !resp.CareerAchievements[0].Unlocked {
		t.Fatalf("friend must only receive unlocked achievements: %+v", resp.CareerAchievements)
	}
	if resp.Summary.CareerTotal != 2 || resp.Summary.UniversalTotal != 1 || resp.Summary.SpecialtyGameType != 3 || resp.Summary.SpecialtyTotal != 1 {
		t.Fatalf("friend summary must retain full catalog counts: %+v", resp.Summary)
	}
	if resp.CurrentSeason != nil || resp.History.ChallengeSeason != nil || len(resp.History.ChallengeRecords) != 0 {
		t.Fatalf("friend must not receive private season progress: %+v", resp)
	}
	if len(resp.History.Honors) != 1 || resp.History.Honors[0].Name != "S2 前十" {
		t.Fatalf("friend should receive permanent honors: %+v", resp.History.Honors)
	}
}

func TestGetHonorWallSplitsSpecialtySummaryForAllGameTypes(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		requestedGameType     int
		wantGameType          int
		wantSpecialtyUnlocked int
	}{
		{name: "snooker", requestedGameType: 1, wantGameType: 1},
		{name: "chasing nine ball", requestedGameType: 2, wantGameType: 2},
		{name: "chinese eight ball", requestedGameType: 3, wantGameType: 3, wantSpecialtyUnlocked: 1},
		{name: "american nine ball", requestedGameType: 4, wantGameType: 4},
		{name: "invalid defaults to chinese eight ball", requestedGameType: 99, wantGameType: 3, wantSpecialtyUnlocked: 1},
		{name: "zero defaults to chinese eight ball", requestedGameType: 0, wantGameType: 3, wantSpecialtyUnlocked: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx, db := newHonorWallTestSvc(t)
			seedHonorWallUser(t, svcCtx, 1001, "本人")
			definitions := []model.Achievement{
				{Key: "match_1", Name: "初入战局", Category: "match", GameType: 0, MetricKey: MetricMatchesTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 1, Status: 1},
				{Key: "snooker_break_50_1", Name: "半百一杆", Category: "special", GameType: 1, MetricKey: MetricBreak50Total, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 2, Status: 1},
				{Key: "chasing_golden_break_1", Name: "小金初现", Category: "special", GameType: 2, MetricKey: MetricGoldenBreakTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 3, Status: 1},
				{Key: "break_clear_1", Name: "初次炸清", Category: "special", GameType: 3, MetricKey: MetricBreakClearTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 4, Status: 1},
				{Key: "american_golden_break_1", Name: "小金初现", Category: "special", GameType: 4, MetricKey: MetricGoldenBreakTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 5, Status: 1},
			}
			if err := db.Create(&definitions).Error; err != nil {
				t.Fatalf("seed scoped achievements: %v", err)
			}
			seedHonorWallUnlocked(t, db, 1001, definitions[0].Id, 1, time.Now())
			seedHonorWallUnlocked(t, db, 1001, definitions[3].Id, 1, time.Now())

			resp, err := NewGetHonorWallLogic(honorWallContext(1001), svcCtx).GetHonorWall(&types.GetHonorWallReq{GameType: tc.requestedGameType})
			if err != nil {
				t.Fatalf("get honor wall: %v", err)
			}
			if len(resp.CareerAchievements) != 5 || resp.Summary.CareerTotal != 5 || resp.Summary.CareerUnlocked != 2 {
				t.Fatalf("unexpected full career summary: %+v", resp.Summary)
			}
			if resp.Summary.UniversalTotal != 1 || resp.Summary.UniversalUnlocked != 1 || resp.Summary.SpecialtyTotal != 1 || resp.Summary.SpecialtyUnlocked != tc.wantSpecialtyUnlocked || resp.Summary.SpecialtyGameType != tc.wantGameType {
				t.Fatalf("unexpected scoped summary: %+v", resp.Summary)
			}
		})
	}
}

func TestGetHonorWallRejectsNonFriendAndBlacklistRelation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		seedBlock bool
	}{
		{name: "non friend"},
		{name: "blacklisted friend", seedBlock: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx, db := newHonorWallTestSvc(t)
			seedHonorWallUser(t, svcCtx, 1001, "查看者")
			seedHonorWallUser(t, svcCtx, 2002, "目标")
			if tc.seedBlock {
				if err := db.Create(&model.Friend{UserId: 1001, FriendId: 2002, Status: 1}).Error; err != nil {
					t.Fatalf("seed friend: %v", err)
				}
				if err := db.Create(&model.FriendBlacklist{UserId: 2002, BlockedUserId: 1001}).Error; err != nil {
					t.Fatalf("seed blacklist: %v", err)
				}
			}

			resp, err := NewGetHonorWallLogic(honorWallContext(1001), svcCtx).GetHonorWall(&types.GetHonorWallReq{UserId: 2002})
			if err != nil {
				t.Fatalf("get honor wall: %v", err)
			}
			if resp.Success {
				t.Fatalf("expected access denial, got %+v", resp)
			}
		})
	}
}

func TestGetHonorWallEmptyStateWithoutActiveSeason(t *testing.T) {
	svcCtx, _ := newHonorWallTestSvc(t)
	seedHonorWallUser(t, svcCtx, 1001, "空状态用户")

	resp, err := NewGetHonorWallLogic(honorWallContext(1001), svcCtx).GetHonorWall(nil)
	if err != nil {
		t.Fatalf("get empty honor wall: %v", err)
	}
	if !resp.Success || resp.CurrentSeason != nil || resp.EquippedTitle != nil {
		t.Fatalf("unexpected empty response: %+v", resp)
	}
	if resp.RecentHonors == nil || resp.CareerAchievements == nil || resp.History.Honors == nil || resp.History.ChallengeRecords == nil {
		t.Fatalf("empty collections must be non-nil: %+v", resp)
	}
}

func seedHonorWallUser(t *testing.T, svcCtx *svc.ServiceContext, userId int64, nickname string) {
	t.Helper()
	if err := svcCtx.UserModel.Create(&model.User{Id: userId, Nickname: nickname, Status: 1}); err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func seedHonorWallAchievements(t *testing.T, db *gorm.DB) []model.Achievement {
	t.Helper()
	definitions := []model.Achievement{
		{Key: "wins_10", Name: "十胜起步", Description: "累计赢下 10 场", Category: "wins", GameType: 0, MetricKey: MetricWinsTotal, ProgressMode: ProgressModeSum, Threshold: 10, RewardTitleName: "胜场新星", Sort: 1, Status: 1},
		{Key: "break_clear_1", Name: "初次炸清", Description: "打出 1 次炸清", Category: "special", GameType: 3, MetricKey: MetricBreakClearTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 2, Status: 1},
	}
	if err := db.Create(&definitions).Error; err != nil {
		t.Fatalf("seed achievements: %v", err)
	}
	return definitions
}

func seedHonorWallUnlocked(t *testing.T, db *gorm.DB, userId, achievementId int64, progress int, unlockedAt time.Time) {
	t.Helper()
	if err := db.Create(&model.UserAchievement{UserId: userId, AchievementId: achievementId, Progress: progress, Unlocked: 1, UnlockedAt: &unlockedAt}).Error; err != nil {
		t.Fatalf("seed user achievement: %v", err)
	}
}

func seedHonorWallTitle(t *testing.T, db *gorm.DB, title model.UserTitle) {
	t.Helper()
	if title.Source == "" {
		title.Source = title.SourceType
	}
	if err := db.Create(&title).Error; err != nil {
		t.Fatalf("seed title: %v", err)
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}
