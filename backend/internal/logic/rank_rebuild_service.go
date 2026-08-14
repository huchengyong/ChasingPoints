package logic

import (
	"context"
	"sort"
	"time"

	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

type RankRebuildSummary struct {
	MatchesTotal    int
	UsersTotal      int
	GameTypes       []int
	RankLogsTotal   int
	SeasonRecords   int
	SkippedDraws    int
	SkippedNoPlayer int
}

type RankRebuildService struct {
	svcCtx *svc.ServiceContext
}

type rankUserGameKey struct {
	UserId   int64
	GameType int
}

type seasonUserGameStats struct {
	MatchesPlayed int
	Wins          int
}

type rankDailyGainKey struct {
	UserId   int64
	GameType int
	DayStart time.Time
}

type rankPairDailyKey struct {
	PlayerA  int64
	PlayerB  int64
	GameType int
	DayStart time.Time
}

func NewRankRebuildService(svcCtx *svc.ServiceContext) *RankRebuildService {
	return &RankRebuildService{svcCtx: svcCtx}
}

func (s *RankRebuildService) DryRun(_ context.Context) (*RankRebuildSummary, error) {
	matches, err := s.svcCtx.MatchModel.ListCompletedForRankingReplay()
	if err != nil {
		return nil, err
	}

	summary := &RankRebuildSummary{}
	userSet := make(map[int64]struct{})
	gameTypeSet := make(map[int]struct{})

	for _, match := range matches {
		summary.MatchesTotal++
		userSet[match.UserId] = struct{}{}
		gameTypeSet[match.GameType] = struct{}{}
		if match.OpponentId != nil && *match.OpponentId > 0 {
			userSet[*match.OpponentId] = struct{}{}
		} else {
			summary.SkippedNoPlayer++
		}

		if isDrawMatch(&match) {
			summary.SkippedDraws++
			continue
		}

		summary.RankLogsTotal++
		if match.OpponentId != nil && *match.OpponentId > 0 {
			summary.RankLogsTotal++
		}
	}

	summary.UsersTotal = len(userSet)
	summary.GameTypes = sortedGameTypes(gameTypeSet)

	return summary, nil
}

func (s *RankRebuildService) Rebuild(ctx context.Context) (*RankRebuildSummary, error) {
	matches, err := s.svcCtx.MatchModel.ListCompletedForRankingReplay()
	if err != nil {
		return nil, err
	}
	seasons, err := s.svcCtx.SeasonModel.ListAll()
	if err != nil {
		return nil, err
	}
	seasonLocation, err := seasonx.LocationForConfig(s.svcCtx.Config.SeasonLifecycle)
	if err != nil {
		return nil, err
	}

	summary, err := s.DryRun(ctx)
	if err != nil {
		return nil, err
	}

	settlementService := NewRankSettlementService(s.svcCtx.RankingModel)
	rewardCache := make(map[int]map[string]int)
	seasonStats := make(map[int64]map[rankUserGameKey]*seasonUserGameStats)
	userGamePairs := make(map[rankUserGameKey]struct{})
	dailyPositiveGains := make(map[rankDailyGainKey]int)
	sameOpponentDailyCounts := make(map[rankPairDailyKey]int)
	var rebuiltSeasonRecords []model.SeasonRecord

	err = s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.svcCtx.RankingModel.DeleteAllRankChangeLogs(tx); err != nil {
			return err
		}
		if err := s.svcCtx.RankingModel.DeleteAllRankings(tx); err != nil {
			return err
		}
		if err := s.svcCtx.SeasonRecordModel.DeleteAll(tx); err != nil {
			return err
		}

		for _, match := range matches {
			userGamePairs[rankUserGameKey{UserId: match.UserId, GameType: match.GameType}] = struct{}{}
			if match.OpponentId != nil && *match.OpponentId > 0 {
				userGamePairs[rankUserGameKey{UserId: *match.OpponentId, GameType: match.GameType}] = struct{}{}
			}
			recordSeasonStats(seasonStats, seasons, &match, seasonLocation)
			effectiveAt := resolveRankChangeEffectiveAt(&match)
			dayStart, _ := rankSettlementDayRange(effectiveAt)
			if isDrawMatch(&match) {
				updateReplaySettlementState(dailyPositiveGains, sameOpponentDailyCounts, &match, dayStart, 0, 0)
				continue
			}

			rewardMap, rewardErr := s.loadAchievementRewardMap(match.GameType, rewardCache)
			if rewardErr != nil {
				return rewardErr
			}

			rounds, roundsErr := s.svcCtx.MatchModel.ListCompletedRounds(match.Id)
			if roundsErr != nil {
				return roundsErr
			}
			actions, actionsErr := s.svcCtx.MatchModel.ListActiveActionsWithTx(tx, match.Id)
			if actionsErr != nil {
				return actionsErr
			}

			storedAchievements, achErr := s.svcCtx.MatchModel.GetAchievementsWithTx(tx, match.Id)
			if achErr != nil {
				return achErr
			}

			player1AchievementScore, player2AchievementScore := resolveReplayAchievementScores(
				match.GameType,
				rounds,
				actions,
				storedAchievements,
				rewardMap,
			)
			player1Win := isPlayer1Win(&match)
			player1Policy := RankSettlementPolicy{
				DailyPositiveCap: defaultDailyPositiveCap,
				TodayPositiveGain: dailyPositiveGains[rankDailyGainKey{
					UserId:   match.UserId,
					GameType: match.GameType,
					DayStart: dayStart,
				}],
			}
			var pairDailyKey rankPairDailyKey
			if match.OpponentId != nil && *match.OpponentId > 0 {
				pairDailyKey = buildRankPairDailyKey(match.UserId, *match.OpponentId, match.GameType, dayStart)
				player1Policy.SameOpponentMatchesToday = sameOpponentDailyCounts[pairDailyKey]
			}

			player1Ranking, rankingErr := s.svcCtx.RankingModel.FindOrCreateWithTx(tx, match.UserId, match.GameType)
			if rankingErr != nil {
				return rankingErr
			}
			player1Settlement := settlementService.SettleWithPolicy(player1Ranking, player1Win, player1AchievementScore, player1Policy)
			applySettlementToRanking(player1Ranking, player1Settlement)
			if err := s.svcCtx.RankingModel.UpdateRankingSnapshot(tx, player1Ranking); err != nil {
				return err
			}

			changeLogs := []model.RankChangeLog{
				buildRankChangeLog(match.Id, match.UserId, match.GameType, resultLabel(player1Win, false), effectiveAt, player1Settlement),
			}

			if match.OpponentId != nil && *match.OpponentId > 0 {
				player2Policy := RankSettlementPolicy{
					DailyPositiveCap: defaultDailyPositiveCap,
					TodayPositiveGain: dailyPositiveGains[rankDailyGainKey{
						UserId:   *match.OpponentId,
						GameType: match.GameType,
						DayStart: dayStart,
					}],
					SameOpponentMatchesToday: sameOpponentDailyCounts[pairDailyKey],
				}
				player2Ranking, rankingErr := s.svcCtx.RankingModel.FindOrCreateWithTx(tx, *match.OpponentId, match.GameType)
				if rankingErr != nil {
					return rankingErr
				}
				player2Settlement := settlementService.SettleWithPolicy(player2Ranking, !player1Win, player2AchievementScore, player2Policy)
				applySettlementToRanking(player2Ranking, player2Settlement)
				if err := s.svcCtx.RankingModel.UpdateRankingSnapshot(tx, player2Ranking); err != nil {
					return err
				}
				changeLogs = append(changeLogs, buildRankChangeLog(match.Id, *match.OpponentId, match.GameType, resultLabel(!player1Win, false), effectiveAt, player2Settlement))
				if player2Settlement.FinalChange > 0 {
					dailyPositiveGains[rankDailyGainKey{
						UserId:   *match.OpponentId,
						GameType: match.GameType,
						DayStart: dayStart,
					}] += player2Settlement.FinalChange
				}
			}

			if err := s.svcCtx.RankingModel.CreateRankChangeLogs(tx, changeLogs); err != nil {
				return err
			}
			player2FinalChange := 0
			if match.OpponentId != nil && *match.OpponentId > 0 {
				player2FinalChange = changeLogs[len(changeLogs)-1].FinalChange
			}
			updateReplaySettlementState(dailyPositiveGains, sameOpponentDailyCounts, &match, dayStart, player1Settlement.FinalChange, player2FinalChange)
		}

		records, recordsErr := s.buildSeasonRecords(tx, userGamePairs, seasonStats, seasons, seasonLocation)
		if recordsErr != nil {
			return recordsErr
		}
		summary.SeasonRecords = len(records)
		rebuiltSeasonRecords = records
		return s.svcCtx.SeasonRecordModel.CreateBatchWithTx(tx, records)
	})
	if err != nil {
		return nil, err
	}
	if err := achievementx.GrantSeasonTitles(s.svcCtx, seasons, rebuiltSeasonRecords); err != nil {
		return nil, err
	}

	return summary, nil
}

func buildRankPairDailyKey(userA, userB int64, gameType int, dayStart time.Time) rankPairDailyKey {
	if userA > userB {
		userA, userB = userB, userA
	}
	return rankPairDailyKey{
		PlayerA:  userA,
		PlayerB:  userB,
		GameType: gameType,
		DayStart: dayStart,
	}
}

func updateReplaySettlementState(
	dailyPositiveGains map[rankDailyGainKey]int,
	sameOpponentDailyCounts map[rankPairDailyKey]int,
	match *model.Match,
	dayStart time.Time,
	player1FinalChange int,
	player2FinalChange int,
) {
	if match == nil {
		return
	}

	if player1FinalChange > 0 {
		dailyPositiveGains[rankDailyGainKey{
			UserId:   match.UserId,
			GameType: match.GameType,
			DayStart: dayStart,
		}] += player1FinalChange
	}
	if match.OpponentId != nil && *match.OpponentId > 0 {
		if player2FinalChange > 0 {
			dailyPositiveGains[rankDailyGainKey{
				UserId:   *match.OpponentId,
				GameType: match.GameType,
				DayStart: dayStart,
			}] += player2FinalChange
		}
		sameOpponentDailyCounts[buildRankPairDailyKey(match.UserId, *match.OpponentId, match.GameType, dayStart)]++
	}
}

func (s *RankRebuildService) loadAchievementRewardMap(gameType int, rewardCache map[int]map[string]int) (map[string]int, error) {
	if rewardMap, ok := rewardCache[gameType]; ok {
		return rewardMap, nil
	}

	configs, err := s.svcCtx.RankingModel.GetAchievementRewardConfigs(gameType)
	if err != nil {
		return nil, err
	}

	rewardMap := make(map[string]int, len(configs))
	for _, item := range configs {
		rewardMap[item.AchievementType] = item.RewardScore
	}
	rewardCache[gameType] = rewardMap
	return rewardMap, nil
}

func (s *RankRebuildService) buildSeasonRecords(
	tx *gorm.DB,
	userGamePairs map[rankUserGameKey]struct{},
	seasonStats map[int64]map[rankUserGameKey]*seasonUserGameStats,
	seasons []model.Season,
	seasonLocation *time.Location,
) ([]model.SeasonRecord, error) {
	records := make([]model.SeasonRecord, 0)

	for _, season := range seasons {
		seasonRecords := make([]model.SeasonRecord, 0)
		statsByPair := seasonStats[season.Id]
		startAt, endExclusive := seasonx.Bounds(&season, seasonLocation)
		for pair := range userGamePairs {
			stats := statsByPair[pair]
			if stats == nil || stats.MatchesPlayed == 0 {
				continue
			}

			seasonLogs, err := s.svcCtx.RankingModel.ListRankChangesByUserAndGameTypeBetweenHalfOpenWithTx(tx, pair.UserId, pair.GameType, startAt, endExclusive)
			if err != nil {
				return nil, err
			}
			beforeLog, err := s.svcCtx.RankingModel.FindLatestRankChangeBeforeByGameTypeWithTx(tx, pair.UserId, pair.GameType, startAt)
			if err != nil {
				return nil, err
			}

			startScore, endScore, peakScore := buildSeasonSnapshotFromLogs(beforeLog, seasonLogs)
			seasonRecords = append(seasonRecords, model.SeasonRecord{
				SeasonId:       season.Id,
				UserId:         pair.UserId,
				GameType:       pair.GameType,
				StartRankScore: startScore,
				EndRankScore:   endScore,
				PeakRankScore:  peakScore,
				MatchesPlayed:  stats.MatchesPlayed,
				Wins:           stats.Wins,
			})
		}

		sort.SliceStable(seasonRecords, func(i, j int) bool {
			if seasonRecords[i].GameType != seasonRecords[j].GameType {
				return seasonRecords[i].GameType < seasonRecords[j].GameType
			}
			if seasonRecords[i].EndRankScore != seasonRecords[j].EndRankScore {
				return seasonRecords[i].EndRankScore > seasonRecords[j].EndRankScore
			}
			if seasonRecords[i].Wins != seasonRecords[j].Wins {
				return seasonRecords[i].Wins > seasonRecords[j].Wins
			}
			return seasonRecords[i].UserId < seasonRecords[j].UserId
		})

		finalRankByGame := make(map[int]int)
		for i := range seasonRecords {
			finalRankByGame[seasonRecords[i].GameType]++
			seasonRecords[i].FinalRank = finalRankByGame[seasonRecords[i].GameType]
		}

		records = append(records, seasonRecords...)
	}

	return records, nil
}

func recordSeasonStats(
	seasonStats map[int64]map[rankUserGameKey]*seasonUserGameStats,
	seasons []model.Season,
	match *model.Match,
	seasonLocation *time.Location,
) {
	if match == nil {
		return
	}

	season := findSeasonForTime(seasons, resolveRankChangeEffectiveAt(match), seasonLocation)
	if season == nil {
		return
	}

	isDraw := isDrawMatch(match)
	upsertSeasonStat(seasonStats, season.Id, rankUserGameKey{UserId: match.UserId, GameType: match.GameType}, !isDraw && isPlayer1Win(match), true)
	if match.OpponentId != nil && *match.OpponentId > 0 {
		upsertSeasonStat(seasonStats, season.Id, rankUserGameKey{UserId: *match.OpponentId, GameType: match.GameType}, !isDraw && !isPlayer1Win(match), true)
	}
}

func upsertSeasonStat(
	seasonStats map[int64]map[rankUserGameKey]*seasonUserGameStats,
	seasonId int64,
	key rankUserGameKey,
	isWin bool,
	countMatch bool,
) {
	if seasonStats[seasonId] == nil {
		seasonStats[seasonId] = make(map[rankUserGameKey]*seasonUserGameStats)
	}
	if seasonStats[seasonId][key] == nil {
		seasonStats[seasonId][key] = &seasonUserGameStats{}
	}
	if countMatch {
		seasonStats[seasonId][key].MatchesPlayed++
	}
	if isWin {
		seasonStats[seasonId][key].Wins++
	}
}

func isDrawMatch(match *model.Match) bool {
	if match == nil {
		return true
	}
	if match.Result != nil {
		return *match.Result == 3
	}
	return match.MyScore == match.OpponentScore
}

func isPlayer1Win(match *model.Match) bool {
	if match == nil {
		return false
	}
	if match.Result != nil {
		return *match.Result == 1
	}
	return match.MyScore > match.OpponentScore
}

func findSeasonForTime(seasons []model.Season, at time.Time, seasonLocation *time.Location) *model.Season {
	for i := range seasons {
		if seasonx.Contains(&seasons[i], at, seasonLocation) {
			return &seasons[i]
		}
	}
	return nil
}

func sortedGameTypes(gameTypeSet map[int]struct{}) []int {
	gameTypes := make([]int, 0, len(gameTypeSet))
	for gameType := range gameTypeSet {
		gameTypes = append(gameTypes, gameType)
	}
	sort.Ints(gameTypes)
	return gameTypes
}
