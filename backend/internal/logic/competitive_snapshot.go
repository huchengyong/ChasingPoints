package logic

import (
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
)

const currentCompetitiveProfileFallbackMatchLimit = 100

// CurrentCompetitiveProfile holds live aggregate values that must not be cached
// with an immutable match core.
type CurrentCompetitiveProfile struct {
	WinRate  float64
	MaxScore int
	Revision int64
}

func LoadCurrentCompetitiveProfile(svcCtx *svc.ServiceContext, userID int64, gameType int) (CurrentCompetitiveProfile, error) {
	if svcCtx == nil || userID <= 0 {
		return CurrentCompetitiveProfile{}, nil
	}
	if !svcCtx.CompetitiveReadModelsEnabled() {
		return loadLegacyCurrentCompetitiveProfile(svcCtx, userID, gameType)
	}
	if svcCtx.CompetitiveReadModel == nil {
		return loadBoundedCurrentCompetitiveProfileFallback(svcCtx, userID, gameType)
	}
	stats, err := svcCtx.CompetitiveReadModel.ListStats(userID)
	if err != nil {
		return CurrentCompetitiveProfile{}, err
	}
	profile := CurrentCompetitiveProfile{}
	hasOverall, hasGame := false, false
	for _, stat := range stats {
		if stat.GameType == 0 {
			hasOverall = true
			profile.Revision = stat.Revision
			if stat.TotalMatches > 0 {
				profile.WinRate = float64(stat.Wins) / float64(stat.TotalMatches) * 100
			}
		}
		if stat.GameType == gameType {
			hasGame = true
			profile.MaxScore = stat.HighestScore
			if gameType == 1 {
				profile.MaxScore = stat.HighestBreak
			}
		}
	}
	if !hasOverall || !hasGame {
		return loadBoundedCurrentCompetitiveProfileFallback(svcCtx, userID, gameType)
	}
	return profile, nil
}

func loadBoundedCurrentCompetitiveProfileFallback(svcCtx *svc.ServiceContext, userID int64, gameType int) (CurrentCompetitiveProfile, error) {
	profile := CurrentCompetitiveProfile{}
	if svcCtx == nil || svcCtx.MatchModel == nil {
		return profile, nil
	}
	if svcCtx.RankingModel != nil {
		totalMatches, wins, err := svcCtx.RankingModel.GetCompetitiveWinSummary(userID)
		if err != nil {
			return profile, err
		}
		if totalMatches > 0 {
			profile.WinRate = float64(wins) / float64(totalMatches) * 100
		}
	}
	maxScore, err := loadBoundedCurrentCompetitiveMaxScore(svcCtx, userID, gameType)
	profile.MaxScore = maxScore
	if err != nil {
		return CurrentCompetitiveProfile{}, err
	}
	return profile, nil
}

func loadBoundedCurrentCompetitiveMaxScore(svcCtx *svc.ServiceContext, userID int64, gameType int) (int, error) {
	limit := 1
	if gameType == 1 {
		limit = currentCompetitiveProfileFallbackMatchLimit
	}
	matches, err := svcCtx.MatchModel.ListCompetitiveScoreCandidates(userID, gameType, limit)
	if err != nil || len(matches) == 0 {
		return 0, err
	}
	if gameType != 1 {
		return matches[0].Score, nil
	}
	matchIDs := make([]int64, 0, len(matches))
	for _, match := range matches {
		matchIDs = append(matchIDs, match.MatchId)
	}
	actions, err := svcCtx.MatchModel.ListActiveActionsByMatchIDs(matchIDs)
	if err != nil {
		return 0, err
	}
	actionMap := make(map[int64][]model.MatchAction, len(matches))
	for _, action := range actions {
		actionMap[action.MatchId] = append(actionMap[action.MatchId], action)
	}
	maxScore := 0
	for _, match := range matches {
		if score := calculateSnookerHighestBreak(actionMap[match.MatchId], match.ViewerActor); score > maxScore {
			maxScore = score
		}
	}
	return maxScore, nil
}

func loadLegacyCurrentCompetitiveProfile(svcCtx *svc.ServiceContext, userID int64, gameType int) (CurrentCompetitiveProfile, error) {
	profile := CurrentCompetitiveProfile{}
	if svcCtx == nil || svcCtx.MatchModel == nil {
		return profile, nil
	}
	stats, err := svcCtx.MatchModel.GetUserStats(userID)
	if err != nil {
		return profile, err
	}
	if stats != nil && stats.TotalMatches > 0 {
		profile.WinRate = float64(stats.Wins) / float64(stats.TotalMatches) * 100
	}
	records, err := loadLegacyUserSingleHighScoreRecords(svcCtx, userID, gameType, 1)
	if err != nil {
		return CurrentCompetitiveProfile{}, err
	}
	if len(records) > 0 {
		profile.MaxScore = records[0].Score
	}
	return profile, nil
}
