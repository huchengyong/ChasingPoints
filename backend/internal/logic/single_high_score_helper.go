package logic

import (
	"sort"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

type singleHighScoreCandidate struct {
	MatchId      int64
	GameType     int
	OpponentName string
	Date         time.Time
	ViewerActor  int
	Score        int
	Actions      []model.MatchAction
}

func buildSingleHighScoreRecords(candidates []singleHighScoreCandidate, limit int) []types.SingleHighScoreRecord {
	if limit <= 0 {
		limit = 10
	}

	records := make([]types.SingleHighScoreRecord, 0, len(candidates))
	for _, candidate := range candidates {
		score := candidate.Score
		if candidate.GameType == 1 {
			score = calculateSnookerHighestBreak(candidate.Actions, candidate.ViewerActor)
		}
		records = append(records, types.SingleHighScoreRecord{
			MatchId:      candidate.MatchId,
			Score:        score,
			GameType:     candidate.GameType,
			GameTypeName: GetGameTypeName(candidate.GameType),
			OpponentName: candidate.OpponentName,
			Date:         candidate.Date.Format("2006-01-02"),
		})
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].Score == records[j].Score {
			return records[i].Date > records[j].Date
		}
		return records[i].Score > records[j].Score
	})

	if len(records) > limit {
		return records[:limit]
	}
	return records
}

func loadUserSingleHighScoreRecords(svcCtx *svc.ServiceContext, userId int64, gameType int, limit int) ([]types.SingleHighScoreRecord, error) {
	if svcCtx == nil || svcCtx.MatchModel == nil {
		return []types.SingleHighScoreRecord{}, nil
	}

	matches, err := svcCtx.MatchModel.ListCompletedWithPerspectiveByUserId(userId, gameType)
	if err != nil {
		return nil, err
	}

	matchIDs := make([]int64, 0, len(matches))
	for _, match := range matches {
		matchIDs = append(matchIDs, match.Id)
	}

	actionMap := map[int64][]model.MatchAction{}
	if len(matchIDs) > 0 {
		actions, err := svcCtx.MatchModel.ListActiveActionsByMatchIDs(matchIDs)
		if err != nil {
			return nil, err
		}
		for _, action := range actions {
			actionMap[action.MatchId] = append(actionMap[action.MatchId], action)
		}
	}

	candidates := make([]singleHighScoreCandidate, 0, len(matches))
	for _, match := range matches {
		viewerActor := 1
		if match.IsAsOpponent {
			viewerActor = 2
		}
		candidates = append(candidates, singleHighScoreCandidate{
			MatchId:      match.Id,
			GameType:     match.GameType,
			OpponentName: match.OpponentName,
			Date:         match.MatchTime,
			ViewerActor:  viewerActor,
			Score:        match.MyScore,
			Actions:      actionMap[match.Id],
		})
	}

	return buildSingleHighScoreRecords(candidates, limit), nil
}

func loadUserMaxSingleScore(svcCtx *svc.ServiceContext, userId int64, gameType int) (int, error) {
	records, err := loadUserSingleHighScoreRecords(svcCtx, userId, gameType, 1)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}
	return records[0].Score, nil
}
