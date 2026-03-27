package match

import (
	logic "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

type RankSettlementResult = logic.RankSettlementResult
type RankSettlementPolicy = logic.RankSettlementPolicy

const defaultDailyPositiveCap = logic.DefaultDailyPositiveCap

func NewRankSettlementService(rankingModel *model.RankingModel) *logic.RankSettlementService {
	return logic.NewRankSettlementService(rankingModel)
}

func buildRankSettlementRemark(settlement RankSettlementResult) string {
	return logic.BuildRankSettlementRemark(settlement)
}

func buildNotificationPayload(target string, matchId, requestId int64) *string {
	return logic.BuildNotificationPayload(target, matchId, requestId)
}

func normalizeStoredAchievementType(winType string) string {
	return logic.NormalizeStoredAchievementType(winType)
}

func resolveReplayAchievementScores(
	gameType int,
	rounds []model.MatchRound,
	actions []model.MatchAction,
	storedAchievements []model.MatchAchievement,
	rewardMap map[string]int,
) (int, int) {
	return logic.ResolveReplayAchievementScores(gameType, rounds, actions, storedAchievements, rewardMap)
}

func buildMatchAchievementPayload(list []model.MatchAchievement) types.MatchAchievement {
	return logic.BuildMatchAchievementPayload(list)
}

func calculateSnookerAchievementScoresByActor(actions []model.MatchAction, rewardMap map[string]int) (int, int) {
	return logic.CalculateSnookerAchievementScoresByActor(actions, rewardMap)
}

func loadUserMaxSingleScore(svcCtx *svc.ServiceContext, userId int64, gameType int) (int, error) {
	return logic.LoadUserMaxSingleScore(svcCtx, userId, gameType)
}

func filterSnookerActionsToCompletedRounds(actions []model.MatchAction, rounds []model.MatchRound) []model.MatchAction {
	return logic.FilterSnookerActionsToCompletedRounds(actions, rounds)
}

func calculateSnookerHighestBreaks(actions []model.MatchAction, myActor int) (int, int) {
	return logic.CalculateSnookerHighestBreaks(actions, myActor)
}

func buildMatchSummary(gameType int, myActor int, myScore int, opponentScore int, myWinRate float64, opponentWinRate float64, myMaxScore int, opponentMaxScore int, redBallCount int, createdAt string, rounds []model.MatchRound, actions []model.MatchAction, achievements types.MatchAchievement) ([]types.MatchSummaryItem, []types.MatchSummaryItem) {
	return logic.BuildMatchSummary(gameType, myActor, myScore, opponentScore, myWinRate, opponentWinRate, myMaxScore, opponentMaxScore, redBallCount, createdAt, rounds, actions, achievements)
}

func rankLogToDetails(log *model.RankChangeLog) []types.RankDetail {
	return logic.RankLogToDetails(log)
}

func findOpponentRankLog(logs []model.RankChangeLog, userId int64) *model.RankChangeLog {
	return logic.FindOpponentRankLog(logs, userId)
}

func GetGameTypeName(gameType int) string {
	return logic.GetGameTypeName(gameType)
}
