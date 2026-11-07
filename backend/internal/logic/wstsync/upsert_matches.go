package wstsync

import (
	"fmt"
	"strings"
	"time"

	"chasing_points/internal/model"

	"gorm.io/gorm"
)

type MatchUpsertRecord struct {
	SourceType              string
	SourceMatchId           string
	SourceTournamentId      string
	RoundName               string
	RoundOrder              int
	MatchOrder              int
	StartTime               *time.Time
	BestOf                  int
	HomePlayerSourceId      string
	HomePlayerName          string
	AwayPlayerSourceId      string
	AwayPlayerName          string
	HomeScore               int
	AwayScore               int
	ScoresConfirmed         bool
	SourceStatus            string
	PlayersAllocated        bool
	PlayersAllocatedPresent bool
	PreserveSourceStatus    bool
}

func UpsertMatches(db *gorm.DB, now time.Time, records []MatchUpsertRecord) error {
	if len(records) == 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		tournamentModel := model.NewTournamentModel(tx)
		playerModel := model.NewPlayerModel(tx)
		matchModel := model.NewTournamentMatchModel(tx)
		grouped := groupMatchUpsertRecords(records)

		for sourceType, items := range grouped {
			tournamentIDs := collectMatchTournamentSourceIDs(items)
			playerIDs := collectMatchPlayerSourceIDs(items)
			matchIDs := collectMatchSourceIDs(items)

			tournaments, err := tournamentModel.FindBySourceTournamentIds(sourceType, tournamentIDs)
			if err != nil {
				return err
			}
			players, err := playerModel.FindBySourcePlayerIds(sourceType, playerIDs)
			if err != nil {
				return err
			}
			existing, err := matchModel.FindBySourceMatchIds(sourceType, matchIDs)
			if err != nil {
				return err
			}

			for _, item := range items {
				match, err := buildOfficialMatch(item, tournaments, players, existing, now)
				if err != nil {
					return err
				}

				if match.Id == 0 {
					if err := tx.Create(match).Error; err != nil {
						return err
					}
					continue
				}

				if err := tx.Save(match).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func groupMatchUpsertRecords(records []MatchUpsertRecord) map[string][]MatchUpsertRecord {
	grouped := make(map[string][]MatchUpsertRecord)
	for _, record := range records {
		grouped[record.SourceType] = append(grouped[record.SourceType], record)
	}
	return grouped
}

func collectMatchSourceIDs(records []MatchUpsertRecord) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0, len(records))
	for _, record := range records {
		sourceID := strings.TrimSpace(record.SourceMatchId)
		if sourceID == "" {
			continue
		}
		if _, ok := seen[sourceID]; ok {
			continue
		}
		seen[sourceID] = struct{}{}
		ids = append(ids, sourceID)
	}
	return ids
}

func collectMatchTournamentSourceIDs(records []MatchUpsertRecord) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0, len(records))
	for _, record := range records {
		sourceID := strings.TrimSpace(record.SourceTournamentId)
		if sourceID == "" {
			continue
		}
		if _, ok := seen[sourceID]; ok {
			continue
		}
		seen[sourceID] = struct{}{}
		ids = append(ids, sourceID)
	}
	return ids
}

func collectMatchPlayerSourceIDs(records []MatchUpsertRecord) []string {
	seen := make(map[string]struct{})
	ids := make([]string, 0, len(records)*2)
	for _, record := range records {
		for _, sourceID := range []string{strings.TrimSpace(record.HomePlayerSourceId), strings.TrimSpace(record.AwayPlayerSourceId)} {
			if sourceID == "" {
				continue
			}
			if _, ok := seen[sourceID]; ok {
				continue
			}
			seen[sourceID] = struct{}{}
			ids = append(ids, sourceID)
		}
	}
	return ids
}

func buildOfficialMatch(
	record MatchUpsertRecord,
	tournaments map[string]model.Tournament,
	players map[string]model.Player,
	existing map[string]model.TournamentMatch,
	now time.Time,
) (*model.TournamentMatch, error) {
	tournamentID := strings.TrimSpace(record.SourceTournamentId)
	tournament, ok := tournaments[tournamentID]
	if !ok {
		return nil, fmt.Errorf("tournament not found for source id %q", tournamentID)
	}

	placeholder := isOfficialMatchPlaceholder(record)
	status := resolveOfficialMatchStatus(record, now, placeholder)
	if placeholder && !record.PreserveSourceStatus {
		status = model.EventNewsStatusUpcoming
	}

	homePlayerID, err := resolveOfficialMatchPlayerID(players, record.HomePlayerSourceId, placeholder)
	if err != nil {
		return nil, err
	}
	awayPlayerID, err := resolveOfficialMatchPlayerID(players, record.AwayPlayerSourceId, placeholder)
	if err != nil {
		return nil, err
	}

	match := &model.TournamentMatch{
		TournamentId:   tournament.Id,
		SourceType:     strings.TrimSpace(record.SourceType),
		SourceMatchId:  strings.TrimSpace(record.SourceMatchId),
		RoundName:      strings.TrimSpace(record.RoundName),
		RoundNumber:    record.RoundOrder,
		RoundOrder:     record.RoundOrder,
		MatchOrder:     record.MatchOrder,
		StartTime:      normalizeOfficialMatchStartTime(record.StartTime),
		BestOf:         record.BestOf,
		HomePlayerId:   homePlayerID,
		HomePlayerName: strings.TrimSpace(record.HomePlayerName),
		AwayPlayerId:   awayPlayerID,
		AwayPlayerName: strings.TrimSpace(record.AwayPlayerName),
		HomeScore:      record.HomeScore,
		AwayScore:      record.AwayScore,
		WinnerSide:     deriveOfficialWinnerSide(record, status, placeholder),
		IsPlaceholder:  placeholder,
		Player1Id:      homePlayerID,
		Player2Id:      awayPlayerID,
		Status:         status,
	}
	match.WinnerId = deriveOfficialWinnerID(match)

	if current, ok := existing[match.SourceMatchId]; ok {
		match.Id = current.Id
		match.MatchId = current.MatchId
		match.BracketPosition = current.BracketPosition
		match.CreatedAt = current.CreatedAt
		match.DeletedAt = current.DeletedAt
	}

	if match.Id == 0 {
		match.CreatedAt = now
	}

	return match, nil
}

func resolveOfficialMatchPlayerID(players map[string]model.Player, sourcePlayerID string, placeholder bool) (int64, error) {
	sourcePlayerID = strings.TrimSpace(sourcePlayerID)
	if sourcePlayerID == "" || placeholder {
		return 0, nil
	}

	player, ok := players[sourcePlayerID]
	if !ok {
		return 0, fmt.Errorf("player not found for source id %q", sourcePlayerID)
	}

	return player.Id, nil
}

func normalizeOfficialMatchStartTime(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}

func isOfficialMatchPlaceholder(record MatchUpsertRecord) bool {
	if !record.PlayersAllocated {
		return true
	}

	return isOfficialPlaceholderValue(record.HomePlayerSourceId) &&
		isOfficialPlaceholderValue(record.AwayPlayerSourceId) &&
		isOfficialPlaceholderValue(record.HomePlayerName) &&
		isOfficialPlaceholderValue(record.AwayPlayerName)
}

func isOfficialPlaceholderValue(value string) bool {
	switch normalizeOfficialText(value) {
	case "", "待定", "tbd", "tbc", "bye":
		return true
	default:
		return false
	}
}

func normalizeOfficialText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(strings.ToLower(value))), " ")
}

func mapOfficialMatchStatus(sourceStatus string) int {
	switch normalizeOfficialText(sourceStatus) {
	case "", "scheduled", "upcoming", "not started":
		return model.EventNewsStatusUpcoming
	case "live", "in progress", "inprogress", "playing", "suspended":
		return model.EventNewsStatusLive
	case "completed", "finished":
		return model.EventNewsStatusFinished
	case "canceled", "cancelled", "abandoned", "walkover", "withdrawn":
		return model.EventNewsStatusCanceled
	default:
		return model.EventNewsStatusUpcoming
	}
}

func isKnownOfficialMatchStatus(sourceStatus string) bool {
	switch normalizeOfficialText(sourceStatus) {
	case "scheduled", "live", "completed":
		return true
	default:
		return false
	}
}

func resolveOfficialMatchStatus(record MatchUpsertRecord, now time.Time, placeholder bool) int {
	if record.PreserveSourceStatus {
		return mapOfficialMatchStatus(record.SourceStatus)
	}
	if placeholder {
		return model.EventNewsStatusUpcoming
	}

	status := mapOfficialMatchStatus(record.SourceStatus)
	if status != model.EventNewsStatusUpcoming || record.StartTime == nil || record.StartTime.IsZero() {
		return status
	}

	startTime := record.StartTime.UTC()
	switch {
	case now.After(startTime.Add(12 * time.Hour)):
		return model.EventNewsStatusFinished
	case now.After(startTime):
		return model.EventNewsStatusLive
	default:
		return status
	}
}

func deriveOfficialWinnerSide(record MatchUpsertRecord, status int, placeholder bool) int {
	if placeholder || status != model.EventNewsStatusFinished {
		return 0
	}

	switch {
	case record.HomeScore > record.AwayScore:
		return 1
	case record.AwayScore > record.HomeScore:
		return 2
	default:
		return 0
	}
}

func deriveOfficialWinnerID(match *model.TournamentMatch) int64 {
	switch match.WinnerSide {
	case 1:
		return match.HomePlayerId
	case 2:
		return match.AwayPlayerId
	default:
		return 0
	}
}
