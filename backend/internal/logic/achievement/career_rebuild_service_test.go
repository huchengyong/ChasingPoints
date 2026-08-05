package achievement

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCareerAchievementRebuildDryRunRebuildAndRetry(t *testing.T) {
	svcCtx := newCareerAchievementRebuildTestSvc(t)
	times := seedCareerAchievementRebuildScenario(t, svcCtx)
	service := NewCareerAchievementRebuildService(svcCtx)

	beforeEvents := countCareerRebuildRows[model.AchievementProgressEvent](t, svcCtx.DB)
	beforeAchievements := countCareerRebuildRows[model.UserAchievement](t, svcCtx.DB)
	beforeTitles := countCareerRebuildRows[model.UserTitle](t, svcCtx.DB)
	drySummary, err := service.DryRun(context.Background())
	if err != nil {
		t.Fatalf("dry run career achievement rebuild: %v", err)
	}
	if drySummary.MatchesTotal != 3 || drySummary.TournamentsTotal != 1 || drySummary.UsersTotal != 2 {
		t.Fatalf("unexpected dry-run source summary: %+v", drySummary)
	}
	if drySummary.EventsCreated == 0 || drySummary.AchievementsUnlocked == 0 || drySummary.TitlesGranted == 0 || drySummary.SkippedAmbiguousSpecial != 1 {
		t.Fatalf("unexpected dry-run projection: %+v", drySummary)
	}
	if countCareerRebuildRows[model.AchievementProgressEvent](t, svcCtx.DB) != beforeEvents ||
		countCareerRebuildRows[model.UserAchievement](t, svcCtx.DB) != beforeAchievements ||
		countCareerRebuildRows[model.UserTitle](t, svcCtx.DB) != beforeTitles {
		t.Fatal("dry run must not modify achievement data")
	}

	summary, err := service.Rebuild(context.Background())
	if err != nil {
		t.Fatalf("rebuild career achievements: %v", err)
	}
	if summary.EventsCreated != drySummary.EventsCreated || summary.AchievementsUnlocked != drySummary.AchievementsUnlocked || summary.TitlesGranted != drySummary.TitlesGranted {
		t.Fatalf("rebuild result should match dry-run projection: dry=%+v rebuild=%+v", drySummary, summary)
	}
	assertCareerRebuildAchievement(t, svcCtx, 10, "match_1", 3, true, times.firstMatch)
	assertCareerRebuildAchievement(t, svcCtx, 10, "match_2", 3, true, times.secondMatch)
	assertCareerRebuildAchievement(t, svcCtx, 10, "wins_1", 2, true, times.firstMatch)
	assertCareerRebuildAchievement(t, svcCtx, 10, "streak_2", 2, true, times.secondMatch)
	assertCareerRebuildAchievement(t, svcCtx, 10, "break_clear_1", 1, true, times.firstMatch)
	assertCareerRebuildAchievement(t, svcCtx, 20, "continue_clear_1", 1, true, times.secondMatch)
	assertCareerRebuildAchievement(t, svcCtx, 10, "tournament_join_1", 1, true, times.tournamentJoin)
	assertCareerRebuildAchievement(t, svcCtx, 10, "tournament_finish_1", 1, true, times.tournamentFinish)
	assertCareerRebuildAchievement(t, svcCtx, 10, "tournament_champion_1", 1, true, times.tournamentFinish)
	assertCareerRebuildTitle(t, svcCtx, 10, "title_match_2", times.secondMatch, false)
	assertCareerRebuildTitle(t, svcCtx, 10, "title_tournament_champion_1", times.tournamentFinish, false)
	assertPreservedCareerRebuildAsset(t, svcCtx, times.existingUnlock, times.existingEquip)

	var ambiguousEvents int64
	if err := svcCtx.DB.Model(&model.AchievementProgressEvent{}).
		Where("source_id = ? AND metric_key = ?", 102, MetricBreak100Total).
		Count(&ambiguousEvents).Error; err != nil {
		t.Fatalf("count ambiguous special events: %v", err)
	}
	if ambiguousEvents != 0 {
		t.Fatalf("ambiguous actor record must be skipped, got %d events", ambiguousEvents)
	}
	var inferredEvents int64
	if err := svcCtx.DB.Model(&model.AchievementProgressEvent{}).
		Where("user_id = ? AND source_id = ? AND metric_key = ?", 20, 102, MetricContinueClearTotal).
		Count(&inferredEvents).Error; err != nil {
		t.Fatalf("count action-inferred special events: %v", err)
	}
	if inferredEvents != 1 {
		t.Fatalf("expected one action-inferred special event, got %d", inferredEvents)
	}

	eventCount := countCareerRebuildRows[model.AchievementProgressEvent](t, svcCtx.DB)
	achievementCount := countCareerRebuildRows[model.UserAchievement](t, svcCtx.DB)
	titleCount := countCareerRebuildRows[model.UserTitle](t, svcCtx.DB)
	retry, err := service.Rebuild(context.Background())
	if err != nil {
		t.Fatalf("retry career achievement rebuild: %v", err)
	}
	if retry.EventsCreated != 0 || retry.AchievementsUnlocked != 0 || retry.TitlesGranted != 0 {
		t.Fatalf("retry must be idempotent: %+v", retry)
	}
	if countCareerRebuildRows[model.AchievementProgressEvent](t, svcCtx.DB) != eventCount ||
		countCareerRebuildRows[model.UserAchievement](t, svcCtx.DB) != achievementCount ||
		countCareerRebuildRows[model.UserTitle](t, svcCtx.DB) != titleCount {
		t.Fatal("retry created duplicate achievement data")
	}
}

func TestResolveCareerSpecialMetricsSkipsUntrustedActorAndUsesSyncedActor(t *testing.T) {
	winner := 2
	rounds := []model.MatchRound{{RoundNo: 1, Winner: &winner, WinType: "golden_break"}}
	stored := []model.MatchAchievement{
		{Actor: 1, AchievementType: "break_100", Count: 1},
		{Actor: 2, AchievementType: "golden_break", Count: 2},
	}
	metrics, skipped := resolveCareerSpecialMetrics(&model.Match{}, rounds, nil, stored)
	if skipped != 1 || metrics[1][MetricBreak100Total] != 0 || metrics[2][MetricGoldenBreakTotal] != 2 {
		t.Fatalf("unexpected unsynced special inference: metrics=%+v skipped=%d", metrics, skipped)
	}

	syncedAt := time.Now()
	metrics, skipped = resolveCareerSpecialMetrics(&model.Match{AchievementSyncedAt: &syncedAt}, rounds, nil, stored)
	if skipped != 0 || metrics[1][MetricBreak100Total] != 1 || metrics[2][MetricGoldenBreakTotal] != 2 {
		t.Fatalf("synced actor should be trusted: metrics=%+v skipped=%d", metrics, skipped)
	}
}

type careerRebuildScenarioTimes struct {
	firstMatch       time.Time
	secondMatch      time.Time
	tournamentJoin   time.Time
	tournamentFinish time.Time
	existingUnlock   time.Time
	existingEquip    time.Time
}

func newCareerAchievementRebuildTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Match{},
		&model.MatchRound{},
		&model.MatchAction{},
		&model.MatchAchievement{},
		&model.Tournament{},
		&model.TournamentParticipant{},
		&model.Achievement{},
		&model.AchievementProgressEvent{},
		&model.UserAchievement{},
		&model.UserTitle{},
	); err != nil {
		t.Fatalf("prepare career rebuild schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:                            db,
		MatchModel:                    model.NewMatchModel(db),
		TournamentModel:               model.NewTournamentModel(db),
		TournamentParticipantModel:    model.NewTournamentParticipantModel(db),
		AchievementModel:              model.NewAchievementModel(db),
		AchievementProgressEventModel: model.NewAchievementProgressEventModel(db),
		UserAchievementModel:          model.NewUserAchievementModel(db),
		UserTitleModel:                model.NewUserTitleModel(db),
	}
}

func seedCareerAchievementRebuildScenario(t *testing.T, svcCtx *svc.ServiceContext) careerRebuildScenarioTimes {
	t.Helper()
	definitions := []model.Achievement{
		{Id: 1, Key: "match_1", Name: "初入战局", Category: "match", GameType: 0, MetricKey: MetricMatchesTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 1, Status: 1},
		{Id: 2, Key: "match_2", Name: "渐入佳境", Category: "match", GameType: 0, MetricKey: MetricMatchesTotal, ProgressMode: ProgressModeSum, Threshold: 2, RewardTitleKey: "title_match_2", RewardTitleName: "资深球手", Sort: 2, Status: 1},
		{Id: 3, Key: "wins_1", Name: "首战告捷", Category: "wins", GameType: 0, MetricKey: MetricWinsTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 3, Status: 1},
		{Id: 4, Key: "streak_2", Name: "连胜", Category: "streak", GameType: 0, MetricKey: MetricMaxWinStreak, ProgressMode: ProgressModeMax, Threshold: 2, Sort: 4, Status: 1},
		{Id: 5, Key: "break_clear_1", Name: "初次炸清", Category: "special", GameType: 3, MetricKey: MetricBreakClearTotal, ProgressMode: ProgressModeSum, Threshold: 1, RewardTitleKey: "title_break_clear_1", RewardTitleName: "清台猎手", Sort: 5, Status: 1},
		{Id: 6, Key: "continue_clear_1", Name: "初次接清", Category: "special", GameType: 3, MetricKey: MetricContinueClearTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 6, Status: 1},
		{Id: 7, Key: "tournament_join_1", Name: "赛事启程", Category: "tournament", GameType: 0, MetricKey: MetricTournamentJoinTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 7, Status: 1},
		{Id: 8, Key: "tournament_finish_1", Name: "初登赛场", Category: "tournament", GameType: 0, MetricKey: MetricTournamentFinishTotal, ProgressMode: ProgressModeSum, Threshold: 1, Sort: 8, Status: 1},
		{Id: 9, Key: "tournament_champion_1", Name: "初次登顶", Category: "tournament", GameType: 0, MetricKey: MetricTournamentChampionTotal, ProgressMode: ProgressModeSum, Threshold: 1, RewardTitleKey: "title_tournament_champion_1", RewardTitleName: "冠军球手", Sort: 9, Status: 1},
		{Id: 10, Key: "match_50", Name: "五十征程", Category: "match", GameType: 0, MetricKey: MetricMatchesTotal, ProgressMode: ProgressModeSum, Threshold: 50, RewardTitleKey: "title_match_50", RewardTitleName: "老将", Sort: 10, Status: 1},
	}
	if err := svcCtx.DB.Create(&definitions).Error; err != nil {
		t.Fatalf("seed career definitions: %v", err)
	}

	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	firstMatch := base
	secondMatch := base.Add(24 * time.Hour)
	thirdMatch := base.Add(48 * time.Hour)
	opponentID := int64(20)
	win := 1
	lose := 2
	matches := []model.Match{
		{Id: 101, UserId: 10, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MatchTime: firstMatch.Add(-time.Hour), EndTime: &firstMatch},
		{Id: 102, UserId: 10, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &win, MatchTime: secondMatch.Add(-time.Hour), EndTime: &secondMatch},
		{Id: 103, UserId: 10, OpponentId: &opponentID, OpponentName: "对手", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &lose, MatchTime: thirdMatch.Add(-time.Hour), EndTime: &thirdMatch},
	}
	if err := svcCtx.DB.Create(&matches).Error; err != nil {
		t.Fatalf("seed career matches: %v", err)
	}
	winner1 := 1
	winner2 := 2
	if err := svcCtx.DB.Create(&[]model.MatchRound{
		{MatchId: 101, RoundNo: 1, Winner: &winner1, WinType: "break_clear"},
		{MatchId: 102, RoundNo: 1, Winner: &winner1, WinType: "normal"},
		{MatchId: 103, RoundNo: 1, Winner: &winner2, WinType: "normal"},
	}).Error; err != nil {
		t.Fatalf("seed career rounds: %v", err)
	}
	extra := `{"win_type":"continue_clear"}`
	if err := svcCtx.DB.Create(&model.MatchAction{MatchId: 102, RoundNo: 1, ActionType: "win", Actor: 2, ExtraData: &extra}).Error; err != nil {
		t.Fatalf("seed action-derived special: %v", err)
	}
	if err := svcCtx.DB.Create(&[]model.MatchAchievement{
		{MatchId: 101, Actor: 1, AchievementType: "break_and_run", Count: 1},
		{MatchId: 102, Actor: 1, AchievementType: "break_100", Count: 1},
		{MatchId: 103, Actor: 2, AchievementType: "golden_break", Count: 1},
	}).Error; err != nil {
		t.Fatalf("seed stored special records: %v", err)
	}

	tournamentJoin := base.Add(4 * 24 * time.Hour)
	tournamentFinish := base.Add(9 * 24 * time.Hour)
	tournament := model.Tournament{Id: 201, CreatorId: 10, Name: "新年杯", GameType: 3, Status: 2, EndTime: &tournamentFinish, CreatedAt: tournamentJoin}
	if err := svcCtx.DB.Create(&tournament).Error; err != nil {
		t.Fatalf("seed tournament: %v", err)
	}
	if err := svcCtx.DB.Create(&[]model.TournamentParticipant{
		{TournamentId: 201, UserId: 10, Status: 3, FinalRank: 1, CreatedAt: tournamentJoin},
		{TournamentId: 201, UserId: 20, Status: 2, FinalRank: 2, CreatedAt: tournamentJoin.Add(time.Minute)},
	}).Error; err != nil {
		t.Fatalf("seed tournament participants: %v", err)
	}

	existingUnlock := base.Add(-30 * 24 * time.Hour)
	existingEquip := existingUnlock.Add(time.Hour)
	if err := svcCtx.DB.Create(&model.UserAchievement{UserId: 10, AchievementId: 10, Progress: 70, Unlocked: 1, UnlockedAt: &existingUnlock, UnlockedSourceType: SourceTypeMatch, UnlockedSourceId: 90, RewardGranted: 1, RewardGrantedAt: &existingUnlock}).Error; err != nil {
		t.Fatalf("seed existing high progress: %v", err)
	}
	if err := svcCtx.DB.Create(&model.UserTitle{UserId: 10, TitleKey: "title_match_50", TitleName: "老将", Source: SourceTypeAchievement, SourceType: SourceTypeAchievement, SourceRefId: 10, SourceRefName: "五十征程", Equipped: 1, EquippedAt: &existingEquip, GrantedAt: &existingUnlock}).Error; err != nil {
		t.Fatalf("seed existing equipped title: %v", err)
	}

	return careerRebuildScenarioTimes{firstMatch: firstMatch, secondMatch: secondMatch, tournamentJoin: tournamentJoin, tournamentFinish: tournamentFinish, existingUnlock: existingUnlock, existingEquip: existingEquip}
}

func assertCareerRebuildAchievement(t *testing.T, svcCtx *svc.ServiceContext, userID int64, key string, wantProgress int, wantUnlocked bool, wantUnlockedAt time.Time) {
	t.Helper()
	var definition model.Achievement
	if err := svcCtx.DB.Where("`key` = ?", key).First(&definition).Error; err != nil {
		t.Fatalf("find definition %s: %v", key, err)
	}
	var item model.UserAchievement
	if err := svcCtx.DB.Where("user_id = ? AND achievement_id = ?", userID, definition.Id).First(&item).Error; err != nil {
		t.Fatalf("find user achievement %s: %v", key, err)
	}
	if item.Progress != wantProgress || (item.Unlocked == 1) != wantUnlocked {
		t.Fatalf("unexpected user achievement %s: %+v", key, item)
	}
	if wantUnlocked && (item.UnlockedAt == nil || !item.UnlockedAt.Equal(wantUnlockedAt)) {
		t.Fatalf("achievement %s unlocked_at=%v, want %v", key, item.UnlockedAt, wantUnlockedAt)
	}
}

func assertCareerRebuildTitle(t *testing.T, svcCtx *svc.ServiceContext, userID int64, titleKey string, wantGrantedAt time.Time, wantEquipped bool) {
	t.Helper()
	var title model.UserTitle
	if err := svcCtx.DB.Where("user_id = ? AND title_key = ?", userID, titleKey).First(&title).Error; err != nil {
		t.Fatalf("find title %s: %v", titleKey, err)
	}
	if title.GrantedAt == nil || !title.GrantedAt.Equal(wantGrantedAt) || (title.Equipped == 1) != wantEquipped {
		t.Fatalf("unexpected title %s: %+v", titleKey, title)
	}
}

func assertPreservedCareerRebuildAsset(t *testing.T, svcCtx *svc.ServiceContext, wantUnlockedAt, wantEquippedAt time.Time) {
	t.Helper()
	var item model.UserAchievement
	if err := svcCtx.DB.Where("user_id = ? AND achievement_id = ?", 10, 10).First(&item).Error; err != nil {
		t.Fatalf("find preserved achievement: %v", err)
	}
	if item.Progress != 70 || item.Unlocked != 1 || item.UnlockedAt == nil || !item.UnlockedAt.Equal(wantUnlockedAt) || item.UnlockedSourceType != SourceTypeMatch || item.UnlockedSourceId != 90 {
		t.Fatalf("existing progress or source was overwritten: %+v", item)
	}
	var title model.UserTitle
	if err := svcCtx.DB.Where("user_id = ? AND title_key = ?", 10, "title_match_50").First(&title).Error; err != nil {
		t.Fatalf("find preserved title: %v", err)
	}
	if title.Equipped != 1 || title.EquippedAt == nil || !title.EquippedAt.Equal(wantEquippedAt) {
		t.Fatalf("existing equipped title was overwritten: %+v", title)
	}
}

func countCareerRebuildRows[T any](t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Model(new(T)).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return count
}
