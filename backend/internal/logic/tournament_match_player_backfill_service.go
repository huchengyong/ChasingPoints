package logic

import (
	"context"
	"strings"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

type TournamentMatchPlayerBackfillSummary struct {
	MatchesScanned     int
	HomeLinked         int
	AwayLinked         int
	Player1Linked      int
	Player2Linked      int
	WinnerLinked       int
	PlaceholderSkipped int
}

type TournamentMatchPlayerBackfillService struct {
	svcCtx *svc.ServiceContext
}

func NewTournamentMatchPlayerBackfillService(svcCtx *svc.ServiceContext) *TournamentMatchPlayerBackfillService {
	return &TournamentMatchPlayerBackfillService{svcCtx: svcCtx}
}

func (s *TournamentMatchPlayerBackfillService) DryRun(ctx context.Context, tournamentID int64) (*TournamentMatchPlayerBackfillSummary, error) {
	return s.run(ctx, tournamentID, false)
}

func (s *TournamentMatchPlayerBackfillService) Rebuild(ctx context.Context, tournamentID int64) (*TournamentMatchPlayerBackfillSummary, error) {
	return s.run(ctx, tournamentID, true)
}

func (s *TournamentMatchPlayerBackfillService) run(ctx context.Context, tournamentID int64, apply bool) (*TournamentMatchPlayerBackfillSummary, error) {
	playerLookup, err := s.loadPlayerLookup(ctx)
	if err != nil {
		return nil, err
	}

	matches, err := s.loadMatches(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	summary := &TournamentMatchPlayerBackfillSummary{
		MatchesScanned: len(matches),
	}

	if !apply {
		for i := range matches {
			simulateBackfill(&matches[i], playerLookup, summary)
		}
		return summary, nil
	}

	err = s.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range matches {
			changed := applyBackfill(&matches[i], playerLookup, summary)
			if !changed {
				continue
			}
			if err := tx.Save(&matches[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return summary, nil
}

func (s *TournamentMatchPlayerBackfillService) loadPlayerLookup(ctx context.Context) (map[string]int64, error) {
	var players []model.Player
	if err := s.svcCtx.DB.WithContext(ctx).
		Model(&model.Player{}).
		Where("deleted_at IS NULL").
		Find(&players).Error; err != nil {
		return nil, err
	}

	lookup := make(map[string]int64, len(players))
	for _, item := range players {
		key := normalizePlayerName(item.DisplayName)
		if key == "" || lookup[key] > 0 {
			continue
		}
		lookup[key] = item.Id
	}
	return lookup, nil
}

func (s *TournamentMatchPlayerBackfillService) loadMatches(ctx context.Context, tournamentID int64) ([]model.TournamentMatch, error) {
	query := s.svcCtx.DB.WithContext(ctx).
		Model(&model.TournamentMatch{}).
		Where("deleted_at IS NULL")
	if tournamentID > 0 {
		query = query.Where("tournament_id = ?", tournamentID)
	}

	var matches []model.TournamentMatch
	if err := query.
		Order("tournament_id ASC, round_order ASC, round_number ASC, match_order ASC, id ASC").
		Find(&matches).Error; err != nil {
		return nil, err
	}
	return matches, nil
}

func simulateBackfill(match *model.TournamentMatch, playerLookup map[string]int64, summary *TournamentMatchPlayerBackfillSummary) {
	clone := *match
	applyBackfill(&clone, playerLookup, summary)
}

func applyBackfill(match *model.TournamentMatch, playerLookup map[string]int64, summary *TournamentMatchPlayerBackfillSummary) bool {
	changed := false
	placeholderSeen := false

	if match.HomePlayerId == 0 {
		if isPlaceholderPlayerName(match.HomePlayerName) {
			placeholderSeen = true
		} else if playerID := resolvePlayerID(playerLookup, match.HomePlayerName); playerID > 0 {
			match.HomePlayerId = playerID
			summary.HomeLinked++
			changed = true
		}
	}

	if match.AwayPlayerId == 0 {
		if isPlaceholderPlayerName(match.AwayPlayerName) {
			placeholderSeen = true
		} else if playerID := resolvePlayerID(playerLookup, match.AwayPlayerName); playerID > 0 {
			match.AwayPlayerId = playerID
			summary.AwayLinked++
			changed = true
		}
	}

	if placeholderSeen {
		summary.PlaceholderSkipped++
	}

	if match.Player1Id == 0 && match.HomePlayerId > 0 {
		match.Player1Id = match.HomePlayerId
		summary.Player1Linked++
		changed = true
	}

	if match.Player2Id == 0 && match.AwayPlayerId > 0 {
		match.Player2Id = match.AwayPlayerId
		summary.Player2Linked++
		changed = true
	}

	if match.WinnerId == 0 {
		switch match.WinnerSide {
		case 1:
			if match.HomePlayerId > 0 {
				match.WinnerId = match.HomePlayerId
				summary.WinnerLinked++
				changed = true
			}
		case 2:
			if match.AwayPlayerId > 0 {
				match.WinnerId = match.AwayPlayerId
				summary.WinnerLinked++
				changed = true
			}
		}
	}

	return changed
}

func resolvePlayerID(playerLookup map[string]int64, displayName string) int64 {
	return playerLookup[normalizePlayerName(displayName)]
}

func normalizePlayerName(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(strings.ToLower(value))), " ")
}

func isPlaceholderPlayerName(value string) bool {
	normalized := normalizePlayerName(value)
	if normalized == "" {
		return true
	}

	switch normalized {
	case "待定", "tbd", "tbc", "bye":
		return true
	default:
		return false
	}
}
