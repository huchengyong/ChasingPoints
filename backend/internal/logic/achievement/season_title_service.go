package achievement

import (
	"fmt"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm/clause"
)

func GrantSeasonTitles(svcCtx *svc.ServiceContext, seasons []model.Season, records []model.SeasonRecord) error {
	if svcCtx == nil || svcCtx.DB == nil {
		return nil
	}

	seasonByID := make(map[int64]model.Season, len(seasons))
	for _, season := range seasons {
		seasonByID[season.Id] = season
	}

	for _, record := range records {
		season, ok := seasonByID[record.SeasonId]
		if !ok {
			continue
		}
		titleName, ok := seasonTitleName(season.Name, record.GameType, record.FinalRank)
		if !ok {
			continue
		}
		if err := grantSeasonTitle(svcCtx, record.UserId, season, record.GameType, record.FinalRank, titleName); err != nil {
			return err
		}
	}
	return nil
}

func grantSeasonTitle(svcCtx *svc.ServiceContext, userId int64, season model.Season, gameType int, finalRank int, titleName string) error {
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

	return svcCtx.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "title_key"},
			{Name: "source_type"},
			{Name: "source_ref_id"},
		},
		DoNothing: true,
	}).Create(title).Error
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
