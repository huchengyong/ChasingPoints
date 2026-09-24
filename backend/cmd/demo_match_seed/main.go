package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"chasing_points/internal/config"
	achievementx "chasing_points/internal/logic/achievement"
	matchlogic "chasing_points/internal/logic/match"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/conf"
	"gorm.io/gorm/clause"
)

var (
	configFile = flag.String("f", "etc/chasing_points-api.yaml", "the config file")
	dryRun     = flag.Bool("dry-run", false, "print scenarios without writing data")
	rankBoost  = flag.Int("rank-boost", 1200, "minimum rank score for demo users before seeding")
)

type roundSpec struct {
	Winner  int
	WinType string
}

type demoScenario struct {
	Label     string
	Player1ID int64
	Player2ID int64
	GameType  int
	GameMode  string
	Rounds    []roundSpec
}

func (s demoScenario) WinnerID() int64 {
	player1Wins := 0
	player2Wins := 0
	for _, round := range s.Rounds {
		if round.Winner == 1 {
			player1Wins++
		} else if round.Winner == 2 {
			player2Wins++
		}
	}
	if player2Wins > player1Wins {
		return s.Player2ID
	}
	return s.Player1ID
}

type seedSummary struct {
	MatchesCreated int
	RoundsCreated  int
}

func main() {
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("警告: 未找到 .env 文件，将使用系统环境变量")
	}

	scenarios := buildDemoScenarios()
	if *dryRun {
		for i, scenario := range scenarios {
			fmt.Printf("%02d %s: user%d vs user%d game_type=%d rounds=%d winner=user%d\n",
				i+1, scenario.Label, scenario.Player1ID, scenario.Player2ID, scenario.GameType, len(scenario.Rounds), scenario.WinnerID())
		}
		return
	}

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	svcCtx := svc.NewServiceContext(c)
	if err := ensureDemoUsers(svcCtx); err != nil {
		log.Fatalf("准备演示用户失败: %v", err)
	}
	if err := ensureMinimumAchievements(svcCtx); err != nil {
		log.Fatalf("准备成就定义失败: %v", err)
	}
	if err := boostDemoRankings(svcCtx, *rankBoost); err != nil {
		log.Fatalf("准备演示排位分失败: %v", err)
	}

	runID := time.Now().Format("20060102150405")
	summary := seedSummary{}
	for i, scenario := range scenarios {
		created, err := createCompletedDemoMatch(svcCtx, runID, i+1, scenario)
		if err != nil {
			log.Fatalf("构造对局失败: index=%d label=%s err=%v", i+1, scenario.Label, err)
		}
		summary.MatchesCreated++
		summary.RoundsCreated += created
	}

	fmt.Printf("demo match seed completed: matches=%d rounds=%d run_id=%s\n", summary.MatchesCreated, summary.RoundsCreated, runID)
	printUserAchievementSummary(svcCtx, []int64{1, 2, 10001, 10002, 10003, 10004, 10005, 10006})
}

func buildDemoScenarios() []demoScenario {
	scenarios := make([]demoScenario, 0, 24)

	for i := 0; i < 10; i++ {
		winType := "normal"
		switch i {
		case 1:
			winType = "break_clear"
		case 3:
			winType = "continue_clear"
		case 5:
			winType = "golden_break"
		case 7:
			winType = "nine_on_break"
		}
		scenarios = append(scenarios, demoScenario{
			Label:     fmt.Sprintf("user1 十胜起步 %02d", i+1),
			Player1ID: 1,
			Player2ID: 2,
			GameType:  3,
			GameMode:  "race",
			Rounds:    []roundSpec{{Winner: 1, WinType: winType}},
		})
	}

	scenarios = append(scenarios,
		demoScenario{Label: "user2 反向胜利", Player1ID: 1, Player2ID: 2, GameType: 3, GameMode: "race", Rounds: []roundSpec{{Winner: 2, WinType: "normal"}}},
		demoScenario{Label: "user1 三局两胜", Player1ID: 1, Player2ID: 2, GameType: 4, GameMode: "race", Rounds: []roundSpec{{Winner: 1, WinType: "normal"}, {Winner: 2, WinType: "normal"}, {Winner: 1, WinType: "break_clear"}}},
		demoScenario{Label: "会员用户特殊战绩胜", Player1ID: 10001, Player2ID: 10002, GameType: 2, GameMode: "points", Rounds: []roundSpec{{Winner: 1, WinType: "golden_break"}, {Winner: 1, WinType: "nine_on_break"}}},
		demoScenario{Label: "会员用户被击败", Player1ID: 10001, Player2ID: 10003, GameType: 2, GameMode: "points", Rounds: []roundSpec{{Winner: 2, WinType: "continue_clear"}}},
		demoScenario{Label: "斯诺克 50+", Player1ID: 10003, Player2ID: 10004, GameType: 1, GameMode: "frame", Rounds: []roundSpec{{Winner: 1, WinType: "break_50"}}},
		demoScenario{Label: "斯诺克 100+", Player1ID: 10004, Player2ID: 10003, GameType: 1, GameMode: "frame", Rounds: []roundSpec{{Winner: 1, WinType: "break_100"}}},
		demoScenario{Label: "斯诺克 147", Player1ID: 10005, Player2ID: 10006, GameType: 1, GameMode: "frame", Rounds: []roundSpec{{Winner: 1, WinType: "break_147"}}},
		demoScenario{Label: "普通用户多局失利", Player1ID: 10006, Player2ID: 10005, GameType: 4, GameMode: "race", Rounds: []roundSpec{{Winner: 2, WinType: "normal"}, {Winner: 2, WinType: "normal"}}},
		demoScenario{Label: "中式八球清台", Player1ID: 10002, Player2ID: 10004, GameType: 3, GameMode: "race", Rounds: []roundSpec{{Winner: 1, WinType: "break_clear"}}},
		demoScenario{Label: "美式九球大金", Player1ID: 10004, Player2ID: 10002, GameType: 4, GameMode: "race", Rounds: []roundSpec{{Winner: 1, WinType: "nine_on_break"}}},
		demoScenario{Label: "九球追分小金", Player1ID: 10003, Player2ID: 10006, GameType: 2, GameMode: "points", Rounds: []roundSpec{{Winner: 1, WinType: "golden_break"}}},
	)

	streakOpponents := []int64{10001, 10002, 10003, 10004, 10006}
	for i := 0; i < 10; i++ {
		scenarios = append(scenarios, demoScenario{
			Label:     fmt.Sprintf("user10005 十连胜冲刺 %02d", i+1),
			Player1ID: 10005,
			Player2ID: streakOpponents[i%len(streakOpponents)],
			GameType:  4,
			GameMode:  "race",
			Rounds:    []roundSpec{{Winner: 1, WinType: "normal"}},
		})
	}

	return scenarios
}

func ensureDemoUsers(svcCtx *svc.ServiceContext) error {
	now := time.Now()
	memberExpiresAt := now.AddDate(0, 1, 0)
	users := []model.User{
		{Id: 1, Phone: stringPtr("13900000001"), Nickname: "演示用户1", Status: 1},
		{Id: 2, Phone: stringPtr("13900000002"), Nickname: "演示用户2", Status: 1},
		{Id: 10001, Phone: stringPtr("13900100001"), Nickname: "演示会员A", Status: 1, MemberExpiresAt: &memberExpiresAt},
		{Id: 10002, Phone: stringPtr("13900100002"), Nickname: "演示会员B", Status: 1, MemberExpiresAt: &memberExpiresAt},
		{Id: 10003, Phone: stringPtr("13900100003"), Nickname: "演示选手C", Status: 1},
		{Id: 10004, Phone: stringPtr("13900100004"), Nickname: "演示选手D", Status: 1},
		{Id: 10005, Phone: stringPtr("13900100005"), Nickname: "演示选手E", Status: 1},
		{Id: 10006, Phone: stringPtr("13900100006"), Nickname: "演示选手F", Status: 1},
	}

	for i := range users {
		if err := svcCtx.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&users[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureMinimumAchievements(svcCtx *svc.ServiceContext) error {
	icon := func(key string) string { return "/static/images/achievements/" + key + ".png" }
	defs := []model.Achievement{
		{Key: "match_1", Name: "初入战局", Description: "累计完成 1 场有效比赛", Icon: icon("match_1"), Category: "match", GameType: 0, MetricKey: achievementx.MetricMatchesTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 100, Status: 1},
		{Key: "match_10", Name: "渐入佳境", Description: "累计完成 10 场有效比赛", Icon: icon("match_10"), Category: "match", GameType: 0, MetricKey: achievementx.MetricMatchesTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 10, Sort: 110, Status: 1},
		{Key: "match_50", Name: "五十征程", Description: "累计完成 50 场有效比赛", Icon: icon("match_50"), Category: "match", GameType: 0, MetricKey: achievementx.MetricMatchesTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 50, Sort: 120, Status: 1},
		{Key: "match_100", Name: "百战磨砺", Description: "累计完成 100 场有效比赛", Icon: icon("match_100"), Category: "match", GameType: 0, MetricKey: achievementx.MetricMatchesTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 100, RewardTitleKey: "title_match_100", RewardTitleName: "资深球手", Sort: 130, Status: 1},
		{Key: "wins_1", Name: "首战告捷", Description: "累计赢下 1 场有效比赛", Icon: icon("wins_1"), Category: "wins", GameType: 0, MetricKey: achievementx.MetricWinsTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 200, Status: 1},
		{Key: "wins_10", Name: "十胜起步", Description: "累计赢下 10 场有效比赛", Icon: icon("wins_10"), Category: "wins", GameType: 0, MetricKey: achievementx.MetricWinsTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 10, RewardTitleKey: "title_wins_10", RewardTitleName: "胜场新星", Sort: 210, Status: 1},
		{Key: "wins_50", Name: "五十胜将", Description: "累计赢下 50 场有效比赛", Icon: icon("wins_50"), Category: "wins", GameType: 0, MetricKey: achievementx.MetricWinsTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 50, Sort: 220, Status: 1},
		{Key: "wins_100", Name: "百胜丰碑", Description: "累计赢下 100 场有效比赛", Icon: icon("wins_100"), Category: "wins", GameType: 0, MetricKey: achievementx.MetricWinsTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 100, RewardTitleKey: "title_wins_100", RewardTitleName: "百胜名将", Sort: 230, Status: 1},
		{Key: "streak_3", Name: "状态正佳", Description: "个人最高连胜达到 3 场", Icon: icon("streak_3"), Category: "streak", GameType: 0, MetricKey: achievementx.MetricMaxWinStreak, ProgressMode: achievementx.ProgressModeMax, Threshold: 3, Sort: 300, Status: 1},
		{Key: "streak_5", Name: "势如破竹", Description: "个人最高连胜达到 5 场", Icon: icon("streak_5"), Category: "streak", GameType: 0, MetricKey: achievementx.MetricMaxWinStreak, ProgressMode: achievementx.ProgressModeMax, Threshold: 5, RewardTitleKey: "title_streak_5", RewardTitleName: "连胜猎手", Sort: 310, Status: 1},
		{Key: "streak_10", Name: "十连制霸", Description: "个人最高连胜达到 10 场", Icon: icon("streak_10"), Category: "streak", GameType: 0, MetricKey: achievementx.MetricMaxWinStreak, ProgressMode: achievementx.ProgressModeMax, Threshold: 10, RewardTitleKey: "title_streak_10", RewardTitleName: "连胜主宰", Sort: 320, Status: 1},
		{Key: "tournament_join_1", Name: "赛事启程", Description: "累计报名 1 场正式赛事", Icon: icon("tournament_join_1"), Category: "tournament", GameType: 0, MetricKey: achievementx.MetricTournamentJoinTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 400, Status: 1},
		{Key: "tournament_finish_1", Name: "初登赛场", Description: "累计完成 1 场正式赛事", Icon: icon("tournament_finish_1"), Category: "tournament", GameType: 0, MetricKey: achievementx.MetricTournamentFinishTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 410, Status: 1},
		{Key: "tournament_finish_10", Name: "久经赛场", Description: "累计完成 10 场正式赛事", Icon: icon("tournament_finish_10"), Category: "tournament", GameType: 0, MetricKey: achievementx.MetricTournamentFinishTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 10, RewardTitleKey: "title_tournament_finish_10", RewardTitleName: "赛事常客", Sort: 420, Status: 1},
		{Key: "tournament_champion_1", Name: "初次登顶", Description: "累计获得 1 次正式赛事冠军", Icon: icon("tournament_champion_1"), Category: "tournament", GameType: 0, MetricKey: achievementx.MetricTournamentChampionTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, RewardTitleKey: "title_tournament_champion_1", RewardTitleName: "冠军球手", Sort: 430, Status: 1},
		{Key: "snooker_break_50_1", Name: "半百一杆", Description: "在斯诺克对局中打出 1 次单杆 50+", Icon: icon("snooker_break_50_1"), Category: "special", GameType: 1, MetricKey: achievementx.MetricBreak50Total, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 500, Status: 1},
		{Key: "snooker_break_100_1", Name: "破百时刻", Description: "在斯诺克对局中打出 1 次单杆 100+", Icon: icon("snooker_break_100_1"), Category: "special", GameType: 1, MetricKey: achievementx.MetricBreak100Total, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 510, Status: 1},
		{Key: "break_147_1", Name: "满分时刻", Description: "在斯诺克对局中打出 1 次单杆 147", Icon: icon("break_147_1"), Category: "special", GameType: 1, MetricKey: achievementx.MetricBreak147Total, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, RewardTitleKey: "title_break_147_1", RewardTitleName: "满分王", Sort: 520, Status: 1},
		{Key: "chasing_golden_break_1", Name: "小金初现", Description: "在九球追分对局中打出 1 次小金", Icon: icon("chasing_golden_break_1"), Category: "special", GameType: 2, MetricKey: achievementx.MetricGoldenBreakTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 600, Status: 1},
		{Key: "chasing_nine_on_break_1", Name: "大金降临", Description: "在九球追分对局中打出 1 次大金", Icon: icon("chasing_nine_on_break_1"), Category: "special", GameType: 2, MetricKey: achievementx.MetricNineOnBreakTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 610, Status: 1},
		{Key: "continue_clear_1", Name: "初次接清", Description: "在中式八球对局中打出 1 次接清", Icon: icon("continue_clear_1"), Category: "special", GameType: 3, MetricKey: achievementx.MetricContinueClearTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 700, Status: 1},
		{Key: "break_clear_1", Name: "初次炸清", Description: "在中式八球对局中打出 1 次炸清", Icon: icon("break_clear_1"), Category: "special", GameType: 3, MetricKey: achievementx.MetricBreakClearTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, RewardTitleKey: "title_break_clear_1", RewardTitleName: "清台猎手", Sort: 710, Status: 1},
		{Key: "break_clear_20", Name: "全台掌控", Description: "在中式八球对局中累计打出 20 次炸清", Icon: icon("break_clear_20"), Category: "special", GameType: 3, MetricKey: achievementx.MetricBreakClearTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 20, RewardTitleKey: "title_break_clear_20", RewardTitleName: "炸清大师", Sort: 720, Status: 1},
		{Key: "american_golden_break_1", Name: "小金初现", Description: "在美式九球对局中打出 1 次小金", Icon: icon("american_golden_break_1"), Category: "special", GameType: 4, MetricKey: achievementx.MetricGoldenBreakTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 800, Status: 1},
		{Key: "american_nine_on_break_1", Name: "大金降临", Description: "在美式九球对局中打出 1 次大金", Icon: icon("american_nine_on_break_1"), Category: "special", GameType: 4, MetricKey: achievementx.MetricNineOnBreakTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Sort: 810, Status: 1},
	}
	for i := range defs {
		if err := svcCtx.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "description", "icon", "category", "game_type", "metric_key", "progress_mode",
				"threshold", "reward_title_key", "reward_title_name", "sort", "status",
			}),
		}).Create(&defs[i]).Error; err != nil {
			return err
		}
	}
	return nil
}

func boostDemoRankings(svcCtx *svc.ServiceContext, minimumScore int) error {
	if minimumScore <= 0 {
		return nil
	}
	userIDs := []int64{1, 2, 10001, 10002, 10003, 10004, 10005, 10006}
	gameTypes := []int{1, 2, 3, 4}
	for _, userID := range userIDs {
		for _, gameType := range gameTypes {
			ranking, err := svcCtx.RankingModel.FindOrCreateByGameType(userID, gameType)
			if err != nil {
				return err
			}
			if ranking.RankScore >= minimumScore {
				continue
			}
			ranking.RankScore = minimumScore
			ranking.RankLevel = svcCtx.RankingModel.CalculateLevel(minimumScore)
			if err := svcCtx.RankingModel.Update(ranking); err != nil {
				return err
			}
		}
	}
	return nil
}

func createCompletedDemoMatch(svcCtx *svc.ServiceContext, runID string, index int, scenario demoScenario) (int, error) {
	opponentName := fmt.Sprintf("用户%d", scenario.Player2ID)
	if user, err := svcCtx.UserModel.FindById(scenario.Player2ID); err == nil && user != nil && user.Nickname != "" {
		opponentName = user.Nickname
	}

	match := &model.Match{
		UserId:                    scenario.Player1ID,
		OpponentId:                &scenario.Player2ID,
		OpponentName:              opponentName,
		GameType:                  scenario.GameType,
		GameMode:                  scenario.GameMode,
		CurrentFrameStarted:       true,
		CurrentFrameMyScore:       0,
		CurrentFrameOpponentScore: 0,
		Status:                    1,
		MatchTime:                 time.Now(),
		Remark:                    "demo_match_seed:" + scenario.Label,
	}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		return 0, err
	}

	revision := int64(0)
	for roundIndex, round := range scenario.Rounds {
		resp, err := matchlogic.NewEndRoundLogic(userCtx(scenario.Player1ID), svcCtx).EndRound(&types.EndRoundReq{
			MatchId:        match.Id,
			Winner:         round.Winner,
			WinType:        round.WinType,
			Score:          1,
			ClientActionId: fmt.Sprintf("demo-%s-%02d-round-%02d", runID, index, roundIndex+1),
			BaseRevision:   revision,
		})
		if err != nil {
			return roundIndex, err
		}
		if !resp.Success || !resp.Accepted {
			return roundIndex, fmt.Errorf("end round rejected: label=%s round=%d resp=%+v", scenario.Label, roundIndex+1, resp)
		}
		revision = resp.ServerRevision
		if scenario.GameType == 1 && roundIndex < len(scenario.Rounds)-1 {
			nextResp, err := matchlogic.NewStartNextRoundLogic(userCtx(scenario.Player1ID), svcCtx).StartNextRound(&types.StartNextRoundReq{
				MatchId:        match.Id,
				ClientActionId: fmt.Sprintf("demo-%s-%02d-next-%02d", runID, index, roundIndex+1),
				BaseRevision:   revision,
			})
			if err != nil {
				return roundIndex + 1, err
			}
			if !nextResp.Success || !nextResp.Accepted {
				return roundIndex + 1, fmt.Errorf("start next round rejected: label=%s round=%d resp=%+v", scenario.Label, roundIndex+1, nextResp)
			}
			revision = nextResp.ServerRevision
		}
	}

	finishResp, err := matchlogic.NewFinishMatchLogic(userCtx(scenario.Player1ID), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId:        match.Id,
		Remark:         "demo_match_seed:" + scenario.Label,
		ClientActionId: fmt.Sprintf("demo-%s-%02d-finish", runID, index),
		BaseRevision:   revision,
	})
	if err != nil {
		return len(scenario.Rounds), err
	}
	if !finishResp.Success || !finishResp.Accepted {
		return len(scenario.Rounds), fmt.Errorf("finish match rejected: label=%s resp=%+v", scenario.Label, finishResp)
	}

	return len(scenario.Rounds), nil
}

func printUserAchievementSummary(svcCtx *svc.ServiceContext, userIDs []int64) {
	for _, userID := range userIDs {
		var rows []struct {
			Key      string
			Name     string
			Progress int
			Unlocked int
		}
		err := svcCtx.DB.Table("user_achievements AS ua").
			Select("a.`key`, a.name, ua.progress, ua.unlocked").
			Joins("JOIN achievements AS a ON a.id = ua.achievement_id").
			Where("ua.user_id = ?", userID).
			Order("a.sort ASC, a.id ASC").
			Find(&rows).Error
		if err != nil {
			log.Printf("查询用户成就失败: user_id=%d err=%v", userID, err)
			continue
		}
		fmt.Printf("user %d achievements:", userID)
		if len(rows) == 0 {
			fmt.Println(" none")
			continue
		}
		for _, row := range rows {
			if row.Unlocked == 1 {
				fmt.Printf(" [%s %s progress=%d]", row.Key, row.Name, row.Progress)
			}
		}
		fmt.Println()
	}
}

func userCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func stringPtr(value string) *string {
	return &value
}
