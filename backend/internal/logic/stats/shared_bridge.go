package stats

import (
	logic "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func GetGameTypeName(gameType int) string {
	return logic.GetGameTypeName(gameType)
}

func buildTrendPointsFromRankChanges(logs []model.RankChangeLog) []types.TrendPoint {
	return logic.BuildTrendPointsFromRankChanges(logs)
}

func loadUserSingleHighScoreRecords(svcCtx *svc.ServiceContext, userId int64, gameType int, limit int) ([]types.SingleHighScoreRecord, error) {
	return logic.LoadUserSingleHighScoreRecords(svcCtx, userId, gameType, limit)
}
