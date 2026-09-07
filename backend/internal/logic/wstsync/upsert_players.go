package wstsync

import (
	"time"

	"chasing_points/internal/model"

	"gorm.io/gorm"
)

type PlayerUpsertRecord struct {
	SourceType     string
	SourcePlayerId string
	FirstName      string
	LastName       string
	DisplayName    string
	Avatar         string
	CountryCode    string
	FlagEmoji      string
}

func UpsertPlayers(db *gorm.DB, now time.Time, records []PlayerUpsertRecord) error {
	if len(records) == 0 {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		playerModel := model.NewPlayerModel(tx)
		grouped := groupPlayerUpsertRecords(records)

		for sourceType, items := range grouped {
			sourceIDs := make([]string, 0, len(items))
			for _, item := range items {
				sourceIDs = append(sourceIDs, item.SourcePlayerId)
			}

			existing, err := playerModel.FindBySourcePlayerIds(sourceType, sourceIDs)
			if err != nil {
				return err
			}

			for _, item := range items {
				player := model.Player{
					SourceType:     item.SourceType,
					SourcePlayerId: item.SourcePlayerId,
					FirstName:      item.FirstName,
					LastName:       item.LastName,
					DisplayName:    item.DisplayName,
					Avatar:         item.Avatar,
					CountryCode:    item.CountryCode,
					FlagEmoji:      item.FlagEmoji,
				}

				if current, ok := existing[item.SourcePlayerId]; ok {
					player.Id = current.Id
					player.CreatedAt = current.CreatedAt
					player.DeletedAt = current.DeletedAt
				}

				if player.Id == 0 {
					if err := tx.Create(&player).Error; err != nil {
						return err
					}
					continue
				}

				if err := tx.Save(&player).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func groupPlayerUpsertRecords(records []PlayerUpsertRecord) map[string][]PlayerUpsertRecord {
	grouped := make(map[string][]PlayerUpsertRecord)
	for _, record := range records {
		grouped[record.SourceType] = append(grouped[record.SourceType], record)
	}
	return grouped
}
