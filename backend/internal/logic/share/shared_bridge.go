package share

import (
	logic "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func buildMatchAchievementPayload(list []model.MatchAchievement) types.MatchAchievement {
	return logic.BuildMatchAchievementPayload(list)
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

func GetGameTypeName(gameType int) string {
	return logic.GetGameTypeName(gameType)
}
