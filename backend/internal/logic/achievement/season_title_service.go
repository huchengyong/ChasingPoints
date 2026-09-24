package achievement

import (
	"fmt"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GrantSeasonTitles(svcCtx *svc.ServiceContext, seasons []model.Season, records []model.SeasonRecord) error {
	_, err := GrantSeasonTitlesWithTx(svcCtx, nil, seasons, records)
	return err
}

func GrantSeasonTitlesWithTx(svcCtx *svc.ServiceContext, tx *gorm.DB, seasons []model.Season, records []model.SeasonRecord) (int, error) {
	if svcCtx == nil || svcCtx.DB == nil {
		return 0, nil
	}

	seasonByID := make(map[int64]model.Season, len(seasons))
	for _, season := range seasons {
		seasonByID[season.Id] = season
	}

	granted := 0
	for _, record := range records {
		season, ok := seasonByID[record.SeasonId]
		if !ok {
			continue
		}
		titleName, ok := seasonTitleName(season.Name, record.GameType, record.FinalRank)
		if !ok {
			continue
		}
		created, err := grantSeasonTitleWithTx(svcCtx, tx, record.UserId, season, record.GameType, record.FinalRank, titleName)
		if err != nil {
			return 0, err
		}
		if created {
			granted++
		}
	}
	return granted, nil
}

func grantSeasonTitle(svcCtx *svc.ServiceContext, userId int64, season model.Season, gameType int, finalRank int, titleName string) error {
	_, err := grantSeasonTitleWithTx(svcCtx, nil, userId, season, gameType, finalRank, titleName)
	return err
}

func grantSeasonTitleWithTx(svcCtx *svc.ServiceContext, tx *gorm.DB, userId int64, season model.Season, gameType int, finalRank int, titleName string) (bool, error) {
	now := time.Now()
	title := &model.UserTitle{
		UserId:        userId,
		TitleKey:      fmt.Sprintf("season_%d_game_%d_rank_%d", season.Id, gameType, finalRank),
		TitleName:     titleName,
		Source:        SourceTypeSeason,
		SourceType:    SourceTypeSeason,
		SourceRefId:   season.Id,
		SourceRefName: season.Name,
		GrantedAt:     &now,
	}

	db := svcCtx.DB
	if tx != nil {
		db = tx
	}
	result := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "title_key"},
			{Name: "source_type"},
			{Name: "source_ref_id"},
		},
		DoNothing: true,
	}).Create(title)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func seasonTitleName(seasonName string, gameType int, finalRank int) (string, bool) {
	suffix := ""
	switch finalRank {
	case 1:
		suffix = "赛季冠军"
	case 2:
		suffix = "赛季亚军"
	case 3:
		suffix = "赛季季军"
	default:
		if finalRank >= 4 && finalRank <= 10 {
			suffix = "赛季前十"
		}
	}
	if suffix == "" {
		return "", false
	}
	return seasonName + seasonGameTypeName(gameType) + suffix, true
}

func seasonGameTypeName(gameType int) string {
	switch gameType {
	case 1:
		return "斯诺克"
	case 2:
		return "九球追分"
	case 3:
		return "中式八球"
	case 4:
		return "美式九球"
	default:
		return "未知"
	}
}
