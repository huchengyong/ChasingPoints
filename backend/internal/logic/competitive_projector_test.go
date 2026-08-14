package logic

import (
	"errors"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCompetitiveProjectorTestContext(t *testing.T) (*gorm.DB, *svc.ServiceContext) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.MatchAction{},
		&model.MatchParticipantResult{},
		&model.UserCompetitiveStats{},
		&model.UserOpponentStats{},
		&model.UserOpponentStrengthBucket{},
		&model.CompetitiveReadModelRebuildCheckpoint{},
		&model.Season{},
		&model.SeasonRecord{},
	); err != nil {
		t.Fatalf("prepare competitive projector schema: %v", err)
	}
	ctx := &svc.ServiceContext{
		DB:                   db,
		UserModel:            model.NewUserModel(db),
		MatchModel:           model.NewMatchModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		SeasonModel:          model.NewSeasonModel(db),
		SeasonRecordModel:    model.NewSeasonRecordModel(db),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}
	return db, ctx
}

func completedRankedMatch(id, player1ID, player2ID int64, gameType, result int, completedAt time.Time) *model.Match {
	return &model.Match{
		Id:            id,
		UserId:        player1ID,
		OpponentId:    &player2ID,
		OpponentName:  "player two",
		GameType:      gameType,
		MatchMode:     model.MatchModeRanked,
		Status:        2,
		Result:        &result,
		MatchTime:     completedAt.Add(-20 * time.Minute),
		EndTime:       &completedAt,
		CompletedAt:   &completedAt,
		MyScore:       7,
		OpponentScore: 4,
	}
}

func rankedProjectionInput(match *model.Match) CompetitiveProjectionInput {
	return CompetitiveProjectionInput{
		Match:         match,
		Player1Before: &model.UserRanking{UserId: match.UserId, GameType: match.GameType, RankScore: 100, RankLevel: 2},
		Player1After:  &model.UserRanking{UserId: match.UserId, GameType: match.GameType, RankScore: 110, RankLevel: 2},
		Player2Before: &model.UserRanking{UserId: *match.OpponentId, GameType: match.GameType, RankScore: 120, RankLevel: 3},
		Player2After:  &model.UserRanking{UserId: *match.OpponentId, GameType: match.GameType, RankScore: 112, RankLevel: 3},
	}
}

func TestCompetitiveProjectorBuildsBothPerspectivesAndAppliesSnapshotsOnce(t *testing.T) {
	db, ctx := newCompetitiveProjectorTestContext(t)
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "one", Avatar: "one.png"}, {Id: 2, Nickname: "two", Avatar: "two.png"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	completedAt := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	if err := db.Create(&model.Season{Id: 1, Name: "S1", StartDate: completedAt.AddDate(0, 0, -1), EndDate: completedAt.AddDate(0, 0, 1), Status: 1}).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	match := completedRankedMatch(100, 1, 2, 3, 1, completedAt)
	projector := NewCompetitiveProjector(ctx)
	out, err := projector.ProjectWithTx(db, rankedProjectionInput(match))
	if err != nil {
		t.Fatalf("project result: %v", err)
	}
	if !out.AppliedUsers[1] || !out.AppliedUsers[2] || out.CompetitiveRevisions[1] != 1 || out.CompetitiveRevisions[2] != 1 {
		t.Fatalf("unexpected projection outcome: %+v", out)
	}

	first, err := ctx.CompetitiveReadModel.FindParticipantByMatchAndUser(match.Id, 1)
	if err != nil || first == nil {
		t.Fatalf("find first perspective: result=%+v err=%v", first, err)
	}
	second, err := ctx.CompetitiveReadModel.FindParticipantByMatchAndUser(match.Id, 2)
	if err != nil || second == nil {
		t.Fatalf("find second perspective: result=%+v err=%v", second, err)
	}
	if first.Result != 1 || first.MyScore != 7 || first.OpponentScore != 4 || first.OpponentRankBucket != "score_0_1000" {
		t.Fatalf("first perspective mismatch: %+v", first)
	}
	if second.Result != 2 || second.MyScore != 4 || second.OpponentScore != 7 || second.OpponentRankBucket != "score_0_1000" {
		t.Fatalf("second perspective must flip outcome and score: %+v", second)
	}

	stats, err := ctx.CompetitiveReadModel.FindStats(1, 0)
	if err != nil || stats == nil || stats.TotalMatches != 1 || stats.Wins != 1 || stats.CurrentWinStreak != 1 || stats.HighestScore != 7 {
		t.Fatalf("unexpected first player stats: %+v err=%v", stats, err)
	}
	seasonRecord, err := ctx.SeasonRecordModel.FindBySeasonAndUserAndGameType(1, 1, 3)
	if err != nil || seasonRecord == nil || seasonRecord.MatchesPlayed != 1 || seasonRecord.Wins != 1 || seasonRecord.StartRankScore != 100 || seasonRecord.EndRankScore != 110 {
		t.Fatalf("unexpected incremental season record: %+v err=%v", seasonRecord, err)
	}

	if _, err := projector.ProjectWithTx(db, rankedProjectionInput(match)); err != nil {
		t.Fatalf("repeat projection: %v", err)
	}
	stats, err = ctx.CompetitiveReadModel.FindStats(1, 0)
	if err != nil || stats.TotalMatches != 1 || stats.Revision != 1 {
		t.Fatalf("repeat projection must not increment stats: %+v err=%v", stats, err)
	}
}

func TestCompetitiveProjectorHandlesAllGameTypesGuestBoundarySnookerBreakAndRollback(t *testing.T) {
	db, ctx := newCompetitiveProjectorTestContext(t)
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "one"}, {Id: 2, Nickname: "two"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	projector := NewCompetitiveProjector(ctx)
	base := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	for gameType := 1; gameType <= 4; gameType++ {
		match := completedRankedMatch(int64(200+gameType), 1, 2, gameType, 1, base.Add(time.Duration(gameType)*time.Hour))
		if gameType == 1 {
			if err := db.Create(&[]model.MatchAction{
				{MatchId: match.Id, RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 60},
				{MatchId: match.Id, RoundNo: 1, ActionType: "score", Actor: 2, ScoreChange: 40},
			}).Error; err != nil {
				t.Fatalf("seed snooker actions: %v", err)
			}
		}
		if _, err := projector.ProjectWithTx(db, rankedProjectionInput(match)); err != nil {
			t.Fatalf("project game type %d: %v", gameType, err)
		}
	}
	snookerStats, err := ctx.CompetitiveReadModel.FindStats(1, 1)
	if err != nil || snookerStats == nil || snookerStats.HighestBreak != 60 {
		t.Fatalf("snooker break must read only this match actions: %+v err=%v", snookerStats, err)
	}

	guestResult := 1
	guestMatch := &model.Match{Id: 300, UserId: 1, OpponentName: "  临时 球友 ", GameType: 3, MatchMode: model.MatchModeRanked, Status: 2, Result: &guestResult, MatchTime: base, EndTime: &base, CompletedAt: &base, MyScore: 5, OpponentScore: 3}
	if _, err := projector.ProjectWithTx(db, CompetitiveProjectionInput{Match: guestMatch, Player1Before: &model.UserRanking{RankScore: 110}, Player1After: &model.UserRanking{RankScore: 115}}); err != nil {
		t.Fatalf("project guest match: %v", err)
	}
	guestProjection, err := ctx.CompetitiveReadModel.FindParticipantByMatchAndUser(300, 1)
	if err != nil || guestProjection == nil || guestProjection.OpponentUserId != 0 || guestProjection.OpponentNameKey != "guest:name:临时 球友" {
		t.Fatalf("guest identity boundary mismatch: %+v err=%v", guestProjection, err)
	}

	rollbackMatch := completedRankedMatch(400, 1, 2, 3, 1, base.Add(10*time.Hour))
	err = db.Transaction(func(tx *gorm.DB) error {
		if _, projectErr := projector.ProjectWithTx(tx, rankedProjectionInput(rollbackMatch)); projectErr != nil {
			return projectErr
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("expected forced rollback")
	}
	projection, err := ctx.CompetitiveReadModel.FindParticipantByMatchAndUser(400, 1)
	if err != nil || projection != nil {
		t.Fatalf("projection must roll back with match transaction: %+v err=%v", projection, err)
	}
}
