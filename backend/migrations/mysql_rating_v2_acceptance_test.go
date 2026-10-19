package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"chasing_points/internal/config"
	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestMySQLRatingV2SettlementQualificationAndAudit 在真实 MySQL 上走一遍 V2 线上结算（行锁、就地切换、幂等日志）、
// 赛季达标维护与达标优先榜单，再用离线重放审计，差异必须为 0。
func TestMySQLRatingV2SettlementQualificationAndAudit(t *testing.T) {
	if os.Getenv("RUN_MYSQL_RATING_V2_TESTS") != "1" {
		t.Skip("set RUN_MYSQL_RATING_V2_TESTS=1 to run MySQL rating V2 acceptance")
	}
	admin, err := sql.Open("mysql", mysqlTestDSN(t, ""))
	if err != nil {
		t.Fatalf("open MySQL admin connection: %v", err)
	}
	defer admin.Close()
	databaseName := fmt.Sprintf("chasing_points_rating_v2_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE `" + databaseName + "`"); err != nil {
		t.Fatalf("create rating v2 database: %v", err)
	}
	defer func() { _, _ = admin.Exec("DROP DATABASE IF EXISTS `" + databaseName + "`") }()
	dsn := mysqlTestDSN(t, databaseName)
	if output, err := exec.Command("goose", "-dir", ".", "mysql", dsn, "up").CombinedOutput(); err != nil {
		t.Fatalf("apply migrations: %v: %s", err, output)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open Gorm database: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB: db, UserModel: model.NewUserModel(db), MatchModel: model.NewMatchModel(db), RankingModel: model.NewRankingModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db), SeasonModel: model.NewSeasonModel(db), SeasonRecordModel: model.NewSeasonRecordModel(db),
		Config: config.Config{
			CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"},
			SeasonLifecycle:      config.SeasonLifecycleConfig{Enabled: true, AnchorDate: "2026-08-01", InitialNumber: 1, CycleMonths: 1, Timezone: "Asia/Shanghai"},
			Ranking:              config.RankingConfig{RatingV2StartDate: "2026-11-01"},
		},
	}
	rules := ranking.RulesFor(svcCtx.Config)
	if !rules.V2Configured() {
		t.Fatalf("rating v2 must be configured: %v", rules.Err())
	}

	for userID := int64(1); userID <= 4; userID++ {
		if err := db.Create(&model.User{Id: userID, Nickname: fmt.Sprintf("user-%d", userID)}).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
		if err := db.Create(&model.UserRanking{UserId: userID, GameType: 3, RankScore: int(userID) * 700, RankLevel: model.RankLevelForScore(int(userID) * 700), TotalWins: 5}).Error; err != nil {
			t.Fatalf("seed legacy ranking: %v", err)
		}
	}
	if err := db.Create(&model.UserRanking{UserId: 4, GameType: 1, RankScore: 900, RankLevel: 2}).Error; err != nil {
		t.Fatalf("seed idle legacy ranking: %v", err)
	}
	if err := db.Create(&model.RankRatingSeed{UserId: 1, GameType: 3, SeedScore: 1500, SeedMatches: 15, CutoffAt: rules.V2StartAt().Add(-time.Hour)}).Error; err != nil {
		t.Fatalf("seed rating seed: %v", err)
	}
	if err := db.Create(&[]model.Season{
		{Id: 3, Name: "S3", StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), Status: 2},
		{Id: 4, Name: "S4", StartDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC), Status: 1},
	}).Error; err != nil {
		t.Fatalf("seed seasons: %v", err)
	}

	// 1 号先与 2、3、4 号各打一场，次日连续 7 场对 2 号：覆盖同对手衰减与第 7 场不计分。
	opponents := []int64{2, 3, 4, 2, 2, 2, 2, 2, 2, 2}
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	for index, opponent := range opponents {
		matchID := int64(100 + index)
		day := 3
		if index >= 3 {
			day = 4 // 第 4 场起都在同一天，对 2 号依次为当日第 1–7 场
		}
		completedAt := time.Date(2026, 11, day, 10+index, 0, 0, 0, shanghai)
		result := 1 + index%2
		opponentID := opponent
		match := model.Match{
			Id: matchID, UserId: 1, OpponentId: &opponentID, OpponentName: "对手", GameType: 3,
			MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, MatchFormat: model.MatchFormatRaceTo, TargetWins: 3,
			Status: 2, Result: &result, MyScore: 3, OpponentScore: 1, MatchTime: completedAt.Add(-time.Hour), EndTime: &completedAt, CompletedAt: &completedAt,
		}
		if err := db.Create(&match).Error; err != nil {
			t.Fatalf("create match %d: %v", matchID, err)
		}
		rounds := []model.MatchRound{}
		for round, winner := range []int{result, result, 3 - result, result} {
			w := winner
			rounds = append(rounds, model.MatchRound{MatchId: matchID, RoundNo: round + 1, Winner: &w, WinType: "normal"})
		}
		if err := db.Create(&rounds).Error; err != nil {
			t.Fatalf("create rounds: %v", err)
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			settlement, err := logicx.SettleRatingV2WithTx(tx, svcCtx, logicx.RatingV2MatchFacts{
				MatchId: matchID, GameType: 3, Player1Id: 1, Player2Id: opponent, Player1Won: result == 1,
				RaceLength: logicx.RatingV2RaceLength(&match, rounds), CompletedAt: completedAt,
			})
			if err != nil {
				return err
			}
			_, err = logicx.NewCompetitiveProjector(svcCtx).ProjectWithTx(tx, logicx.CompetitiveProjectionInput{
				Match: &match, Player1Before: settlement.Player1Before, Player1After: settlement.Player1After,
				Player2Before: settlement.Player2Before, Player2After: settlement.Player2After,
			})
			return err
		})
		if err != nil {
			t.Fatalf("settle match %d on MySQL: %v", matchID, err)
		}
	}

	var resets int64
	if err := db.Model(&model.RankChangeLog{}).Where("change_type = ?", model.RankChangeTypeRatingReset).Count(&resets).Error; err != nil || resets != 4 {
		t.Fatalf("four played rankings must be switched exactly once: resets=%d err=%v", resets, err)
	}
	if switched, err := logicx.SwitchPendingRankingsToRatingV2(svcCtx, rules.V2StartAt().Add(time.Minute)); err != nil || switched != 1 {
		t.Fatalf("worker must switch the idle snooker ranking: switched=%d err=%v", switched, err)
	}

	var leader, repeat model.SeasonRecord
	if err := db.Where("season_id = 4 AND user_id = 1 AND game_type = 3").First(&leader).Error; err != nil {
		t.Fatalf("load qualified record: %v", err)
	}
	if err := db.Where("season_id = 4 AND user_id = 2 AND game_type = 3").First(&repeat).Error; err != nil {
		t.Fatalf("load repeat record: %v", err)
	}
	if leader.MatchesPlayed != 10 || leader.DistinctOpponents != 3 || leader.Qualified != 1 || repeat.DistinctOpponents != 1 || repeat.Qualified != 0 {
		t.Fatalf("MySQL upsert must maintain qualification: leader=%+v repeat=%+v", leader, repeat)
	}
	board, _, err := svcCtx.SeasonRecordModel.FindLeaderboardWithProfilesByGameType(4, 3, 1, 20)
	if err != nil || len(board) != 4 || board[0].UserId != 1 || board[0].Qualified != 1 {
		t.Fatalf("qualified record must lead the MySQL leaderboard: %+v err=%v", board, err)
	}
	if ahead, err := svcCtx.SeasonRecordModel.CountAheadInSeason(&leader); err != nil || ahead != 0 {
		t.Fatalf("forced-index rank count: ahead=%d err=%v", ahead, err)
	}

	resetLogs, err := svcCtx.RankingModel.ListRatingResetLogsWithTx(nil)
	if err != nil {
		t.Fatalf("list resets: %v", err)
	}
	start := map[logicx.RatingReplayKey]logicx.RatingReplayPlayer{}
	for _, reset := range resetLogs {
		remark, _ := ranking.ParseRemark(reset.Remark)
		start[logicx.RatingReplayKey{UserId: reset.UserId, GameType: reset.GameType}] = logicx.RatingReplayPlayer{Rating: reset.AfterScore, RatedMatches: remark.SeedMatches}
	}
	replay := logicx.NewRatingReplay(rules, logicx.RatingV2DailyPositiveCap(svcCtx), start)
	differences := 0
	if err := logicx.ReplayCompletedMatches(svcCtx, replay, rules.V2StartAt(), rules.V2StartAt().AddDate(0, 1, 0), func(result *logicx.RatingReplayMatchResult) error {
		logs, err := svcCtx.RankingModel.ListMatchRankChangesByMatchIdsWithTx(nil, []int64{result.MatchId})
		if err != nil {
			return err
		}
		for _, item := range logs {
			outcome := result.Player1
			if item.UserId == result.Player2Id {
				outcome = result.Player2
			}
			if item.FinalChange != outcome.Change || item.AfterScore != outcome.After {
				differences++
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("replay on MySQL: %v", err)
	}
	for key, player := range replay.Players() {
		var row model.UserRanking
		if err := db.Where("user_id = ? AND game_type = ?", key.UserId, key.GameType).First(&row).Error; err != nil {
			t.Fatalf("load ranking: %v", err)
		}
		if row.RankScore != player.Rating || row.RatedMatches != player.RatedMatches {
			differences++
		}
	}
	if differences != 0 || replay.Applied != len(opponents) {
		t.Fatalf("MySQL audit must converge: differences=%d applied=%d", differences, replay.Applied)
	}
}
