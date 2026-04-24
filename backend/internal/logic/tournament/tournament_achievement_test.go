package tournament

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTournamentAchievementTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Tournament{},
		&model.TournamentParticipant{},
		&model.TournamentMatch{},
		&model.Notification{},
		&model.Achievement{},
		&model.UserAchievement{},
		&model.UserTitle{},
		&model.AchievementProgressEvent{},
	); err != nil {
		t.Fatalf("prepare tournament achievement schema: %v", err)
	}

	return &svc.ServiceContext{
		Config: config.Config{
			Compliance: config.ComplianceConfig{RestrictedMode: false},
		},
		DB:                            db,
		UserModel:                     model.NewUserModel(db),
		TournamentModel:               model.NewTournamentModel(db),
		TournamentParticipantModel:    model.NewTournamentParticipantModel(db),
		TournamentMatchModel:          model.NewTournamentMatchModel(db),
		NotificationModel:             model.NewNotificationModel(db),
		AchievementModel:              model.NewAchievementModel(db),
		UserAchievementModel:          model.NewUserAchievementModel(db),
		UserTitleModel:                model.NewUserTitleModel(db),
		AchievementProgressEventModel: model.NewAchievementProgressEventModel(db),
	}
}

func tournamentAchievementCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func TestJoinTournamentWritesAchievementProgressIdempotently(t *testing.T) {
	svcCtx := newTournamentAchievementTestSvc(t)
	seedTournamentAchievementDefinitions(t, svcCtx)
	seedTournamentAchievementUsers(t, svcCtx, 1001, 9001)
	seedTournamentAchievementTournament(t, svcCtx, &model.Tournament{
		Id:             8001,
		CreatorId:      9001,
		Name:           "追分杯",
		GameType:       3,
		Format:         3,
		MaxPlayers:     16,
		CurrentPlayers: 0,
		Status:         0,
	})

	resp, err := NewJoinTournamentLogic(tournamentAchievementCtx(1001), svcCtx).JoinTournament(&types.TournamentIdReq{TournamentId: 8001})
	if err != nil {
		t.Fatalf("join tournament: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected join success, got %#v", resp)
	}

	assertTournamentUserAchievementProgress(t, svcCtx, 1001, "tournament_join_1", 1, true)
	if countTournamentProgressEvents(t, svcCtx, 1001, achievementx.MetricTournamentJoinTotal) != 1 {
		t.Fatal("expected one tournament_join_total event")
	}

	duplicateResp, err := NewJoinTournamentLogic(tournamentAchievementCtx(1001), svcCtx).JoinTournament(&types.TournamentIdReq{TournamentId: 8001})
	if err != nil {
		t.Fatalf("duplicate join tournament: %v", err)
	}
	if duplicateResp.Success {
		t.Fatalf("expected duplicate join to fail, got %#v", duplicateResp)
	}
	if countTournamentProgressEvents(t, svcCtx, 1001, achievementx.MetricTournamentJoinTotal) != 1 {
		t.Fatal("expected duplicate join to keep one tournament_join_total event")
	}
}

func TestFinishTournamentWritesAchievementProgressAndTraceableTitles(t *testing.T) {
	svcCtx := newTournamentAchievementTestSvc(t)
	seedTournamentAchievementDefinitions(t, svcCtx)
	seedTournamentAchievementUsers(t, svcCtx, 9001, 1001, 1002)
	seedTournamentAchievementTournament(t, svcCtx, &model.Tournament{
		Id:             8002,
		CreatorId:      9001,
		Name:           "城市公开赛",
		GameType:       3,
		Format:         3,
		MaxPlayers:     8,
		CurrentPlayers: 2,
		Status:         1,
	})
	seedTournamentAchievementParticipant(t, svcCtx, 8002, 1001, 1)
	seedTournamentAchievementParticipant(t, svcCtx, 8002, 1002, 2)
	seedTournamentAchievementMatch(t, svcCtx, &model.TournamentMatch{
		TournamentId: 8002,
		RoundNumber:  1,
		MatchOrder:   1,
		Player1Id:    1001,
		Player2Id:    1002,
		WinnerId:     1001,
		Status:       2,
	})

	resp, err := NewFinishTournamentLogic(tournamentAchievementCtx(9001), svcCtx).FinishTournament(&types.TournamentIdReq{TournamentId: 8002})
	if err != nil {
		t.Fatalf("finish tournament: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected finish success, got %#v", resp)
	}

	assertTournamentUserAchievementProgress(t, svcCtx, 1001, "tournament_finish_1", 1, true)
	assertTournamentUserAchievementProgress(t, svcCtx, 1002, "tournament_finish_1", 1, true)
	assertTournamentUserAchievementProgress(t, svcCtx, 1001, "tournament_champion_1", 1, true)
	assertTournamentUserAchievementProgress(t, svcCtx, 1002, "tournament_champion_1", 0, false)
	assertTournamentTitle(t, svcCtx, 1001, "城市公开赛冠军", 8002, "城市公开赛")
	assertTournamentTitle(t, svcCtx, 1002, "城市公开赛亚军", 8002, "城市公开赛")

	if err := svcCtx.DB.Model(&model.Tournament{}).Where("id = ?", 8002).Update("status", 1).Error; err != nil {
		t.Fatalf("reset tournament status for replay: %v", err)
	}
	replayResp, err := NewFinishTournamentLogic(tournamentAchievementCtx(9001), svcCtx).FinishTournament(&types.TournamentIdReq{TournamentId: 8002})
	if err != nil {
		t.Fatalf("replay finish tournament: %v", err)
	}
	if !replayResp.Success {
		t.Fatalf("expected replay finish success, got %#v", replayResp)
	}
	if countTournamentTitles(t, svcCtx, 1001, "城市公开赛冠军") != 1 {
		t.Fatal("expected champion title to stay idempotent")
	}
	if countTournamentProgressEvents(t, svcCtx, 1001, achievementx.MetricTournamentChampionTotal) != 1 {
		t.Fatal("expected champion progress event to stay idempotent")
	}
}

func seedTournamentAchievementDefinitions(t *testing.T, svcCtx *svc.ServiceContext) {
	t.Helper()
	defs := []model.Achievement{
		{Key: "tournament_join_1", Name: "赛事初体验", Category: "tournament", GameType: 0, MetricKey: achievementx.MetricTournamentJoinTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Status: 1},
		{Key: "tournament_finish_1", Name: "赛事完赛", Category: "tournament", GameType: 0, MetricKey: achievementx.MetricTournamentFinishTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Status: 1},
		{Key: "tournament_champion_1", Name: "赛事冠军", Category: "tournament", GameType: 0, MetricKey: achievementx.MetricTournamentChampionTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Status: 1},
	}
	for i := range defs {
		if err := svcCtx.DB.Create(&defs[i]).Error; err != nil {
			t.Fatalf("seed achievement %s: %v", defs[i].Key, err)
		}
	}
}

func seedTournamentAchievementUsers(t *testing.T, svcCtx *svc.ServiceContext, userIDs ...int64) {
	t.Helper()
	for _, userID := range userIDs {
		user := model.User{Id: userID, Nickname: "赛事用户"}
		if err := svcCtx.UserModel.Create(&user); err != nil {
			t.Fatalf("seed user %d: %v", userID, err)
		}
	}
}

func seedTournamentAchievementTournament(t *testing.T, svcCtx *svc.ServiceContext, tournament *model.Tournament) {
	t.Helper()
	if tournament.StartTime == nil {
		now := time.Now()
		tournament.StartTime = &now
	}
	if err := svcCtx.TournamentModel.Create(tournament); err != nil {
		t.Fatalf("seed tournament %d: %v", tournament.Id, err)
	}
}

func seedTournamentAchievementParticipant(t *testing.T, svcCtx *svc.ServiceContext, tournamentID, userID int64, seed int) {
	t.Helper()
	if err := svcCtx.TournamentParticipantModel.Create(&model.TournamentParticipant{
		TournamentId: tournamentID,
		UserId:       userID,
		Seed:         seed,
		Status:       1,
	}); err != nil {
		t.Fatalf("seed participant: %v", err)
	}
}

func seedTournamentAchievementMatch(t *testing.T, svcCtx *svc.ServiceContext, match *model.TournamentMatch) {
	t.Helper()
	if err := svcCtx.TournamentMatchModel.Create(match); err != nil {
		t.Fatalf("seed tournament match: %v", err)
	}
}

func assertTournamentUserAchievementProgress(t *testing.T, svcCtx *svc.ServiceContext, userID int64, key string, wantProgress int, wantUnlocked bool) {
	t.Helper()

	var achievement model.Achievement
	if err := svcCtx.DB.Where("`key` = ?", key).First(&achievement).Error; err != nil {
		t.Fatalf("find achievement %s: %v", key, err)
	}
	var ua model.UserAchievement
	err := svcCtx.DB.Where("user_id = ? AND achievement_id = ?", userID, achievement.Id).First(&ua).Error
	if err != nil {
		t.Fatalf("find user achievement user=%d key=%s: %v", userID, key, err)
	}
	if ua.Progress != wantProgress {
		t.Fatalf("expected user=%d key=%s progress %d, got %d", userID, key, wantProgress, ua.Progress)
	}
	gotUnlocked := ua.Unlocked == 1
	if gotUnlocked != wantUnlocked {
		t.Fatalf("expected user=%d key=%s unlocked=%v, got %v", userID, key, wantUnlocked, gotUnlocked)
	}
}

func assertTournamentTitle(t *testing.T, svcCtx *svc.ServiceContext, userID int64, titleName string, refID int64, refName string) {
	t.Helper()

	var title model.UserTitle
	if err := svcCtx.DB.Where("user_id = ? AND title_name = ?", userID, titleName).First(&title).Error; err != nil {
		t.Fatalf("find user title %s: %v", titleName, err)
	}
	if title.SourceType != achievementx.SourceTypeTournament || title.SourceRefId != refID || title.SourceRefName != refName {
		t.Fatalf("unexpected tournament title source: %+v", title)
	}
}

func countTournamentTitles(t *testing.T, svcCtx *svc.ServiceContext, userID int64, titleName string) int64 {
	t.Helper()

	var total int64
	if err := svcCtx.DB.Model(&model.UserTitle{}).
		Where("user_id = ? AND title_name = ?", userID, titleName).
		Count(&total).Error; err != nil {
		t.Fatalf("count tournament titles: %v", err)
	}
	return total
}

func countTournamentProgressEvents(t *testing.T, svcCtx *svc.ServiceContext, userID int64, metricKey string) int64 {
	t.Helper()

	var total int64
	if err := svcCtx.DB.Model(&model.AchievementProgressEvent{}).
		Where("user_id = ? AND metric_key = ?", userID, metricKey).
		Count(&total).Error; err != nil {
		t.Fatalf("count progress events: %v", err)
	}
	return total
}
