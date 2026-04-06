package wstsync

import (
	"strings"
	"time"

	"chasing_points/internal/model"

	"gorm.io/gorm"
)

type TournamentUpsertRecord struct {
	SourceType         string
	SourceTournamentId string
	SourceSeasonId     string
	Name               string
	CoverImage         string
	GameType           int
	Status             int
	Country            string
	City               string
	VenueName          string
	StartDate          *time.Time
	EndDate            *time.Time
	InformationPage    string
	TicketingLink      string
}

func UpsertTournaments(db *gorm.DB, now time.Time, records []TournamentUpsertRecord) error {
	if len(records) == 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		tournamentModel := model.NewTournamentModel(tx)
		grouped := groupTournamentUpsertRecords(records)

		for sourceType, items := range grouped {
			sourceIDs := make([]string, 0, len(items))
			for _, item := range items {
				sourceIDs = append(sourceIDs, item.SourceTournamentId)
			}

			existing, err := tournamentModel.FindBySourceTournamentIds(sourceType, sourceIDs)
			if err != nil {
				return err
			}

			for _, item := range items {
				tournament := model.Tournament{
					SourceType:         item.SourceType,
					SourceTournamentId: item.SourceTournamentId,
					SourceSeasonId:     item.SourceSeasonId,
					Name:               item.Name,
					CoverImage:         item.CoverImage,
					GameType:           item.GameType,
					Status:             item.Status,
					Country:            item.Country,
					City:               item.City,
					VenueName:          item.VenueName,
					StartDate:          item.StartDate,
					EndDate:            item.EndDate,
					InformationPage:    item.InformationPage,
					TicketingLink:      item.TicketingLink,
					LastSyncedAt:       &now,
				}

				if current, ok := existing[item.SourceTournamentId]; ok {
					tournament.Id = current.Id
					tournament.CreatorId = current.CreatorId
					tournament.Description = current.Description
					tournament.Format = current.Format
					tournament.MaxPlayers = current.MaxPlayers
					tournament.CurrentPlayers = current.CurrentPlayers
					tournament.StartTime = current.StartTime
					tournament.EndTime = current.EndTime
					tournament.CreatedAt = current.CreatedAt
					if strings.TrimSpace(tournament.CoverImage) == "" || tournament.CoverImage == defaultTournamentCoverImage {
						tournament.CoverImage = firstNonEmptyTournamentField(current.CoverImage, tournament.CoverImage)
					}
					tournament.Country = firstNonEmptyTournamentField(tournament.Country, current.Country)
					tournament.City = firstNonEmptyTournamentField(tournament.City, current.City)
					tournament.VenueName = firstNonEmptyTournamentField(tournament.VenueName, current.VenueName)
					tournament.InformationPage = firstNonEmptyTournamentField(tournament.InformationPage, current.InformationPage)
					tournament.TicketingLink = firstNonEmptyTournamentField(tournament.TicketingLink, current.TicketingLink)
					if tournament.StartDate == nil {
						tournament.StartDate = current.StartDate
					}
					if tournament.EndDate == nil {
						tournament.EndDate = current.EndDate
					}
				}

				if tournament.Id == 0 {
					if err := tx.Create(&tournament).Error; err != nil {
						return err
					}
					continue
				}

				if err := tx.Save(&tournament).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func firstNonEmptyTournamentField(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func groupTournamentUpsertRecords(records []TournamentUpsertRecord) map[string][]TournamentUpsertRecord {
	grouped := make(map[string][]TournamentUpsertRecord)
	for _, record := range records {
		grouped[record.SourceType] = append(grouped[record.SourceType], record)
	}
	return grouped
}
