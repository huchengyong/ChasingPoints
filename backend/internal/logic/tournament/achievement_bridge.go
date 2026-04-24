package tournament

import (
	"fmt"
	"time"

	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm/clause"
)

func (l *JoinTournamentLogic) syncTournamentJoinAchievement(userId int64, tournament *model.Tournament) {
	if err := appendTournamentProgressAndRefresh(l.svcCtx, userId, tournament, achievementx.SourceTypeTournamentJoin, achievementx.MetricTournamentJoinTotal, 1); err != nil {
		l.Logger.Errorf("同步赛事报名成就进度失败: tournamentId=%d userId=%d err=%v", tournament.Id, userId, err)
	}
}

func (l *FinishTournamentLogic) syncTournamentFinishAchievements(tournament *model.Tournament, settlement map[int64]tournamentSettlement, participantCount int) {
	if tournament == nil {
		return
	}
	for userId, result := range settlement {
		if result.FinalRank <= 0 {
			continue
		}
		if err := appendTournamentProgressAndRefresh(l.svcCtx, userId, tournament, achievementx.SourceTypeTournamentFinish, achievementx.MetricTournamentFinishTotal, 1); err != nil {
			l.Logger.Errorf("同步赛事完赛成就进度失败: tournamentId=%d userId=%d err=%v", tournament.Id, userId, err)
			continue
		}
		if result.FinalRank == 1 {
			if err := appendTournamentProgressAndRefresh(l.svcCtx, userId, tournament, achievementx.SourceTypeTournamentFinish, achievementx.MetricTournamentChampionTotal, 1); err != nil {
				l.Logger.Errorf("同步赛事冠军成就进度失败: tournamentId=%d userId=%d err=%v", tournament.Id, userId, err)
			}
		}
		if err := grantTournamentRankTitle(l.svcCtx, userId, tournament, result.FinalRank, participantCount); err != nil {
			l.Logger.Errorf("发放赛事称号失败: tournamentId=%d userId=%d rank=%d err=%v", tournament.Id, userId, result.FinalRank, err)
		}
	}
}

func appendTournamentProgressAndRefresh(svcCtx *svc.ServiceContext, userId int64, tournament *model.Tournament, sourceType string, metricKey string, metricValue int) error {
	if !hasTournamentAchievementInfra(svcCtx) || tournament == nil {
		return nil
	}

	progressService := achievementx.NewAchievementProgressService(svcCtx)
	if _, err := progressService.AppendEvent(achievementx.AchievementProgressEventInput{
		UserId:      userId,
		SourceType:  sourceType,
		SourceId:    tournament.Id,
		GameType:    tournament.GameType,
		MetricKey:   metricKey,
		MetricValue: metricValue,
	}); err != nil {
		return err
	}
	return progressService.RefreshUserAchievements(userId)
}

func grantTournamentRankTitle(svcCtx *svc.ServiceContext, userId int64, tournament *model.Tournament, finalRank int, participantCount int) error {
	if !hasTournamentAchievementInfra(svcCtx) || tournament == nil {
		return nil
	}

	titleName, ok := tournamentRankTitleName(tournament.Name, finalRank, participantCount)
	if !ok {
		return nil
	}
	now := time.Now()
	title := &model.UserTitle{
		UserId:        userId,
		TitleKey:      fmt.Sprintf("tournament_%d_rank_%d", tournament.Id, finalRank),
		TitleName:     titleName,
		Source:        achievementx.SourceTypeTournament,
		SourceType:    achievementx.SourceTypeTournament,
		SourceRefId:   tournament.Id,
		SourceRefName: tournament.Name,
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

func hasTournamentAchievementInfra(svcCtx *svc.ServiceContext) bool {
	return svcCtx != nil && svcCtx.DB != nil && svcCtx.AchievementProgressEventModel != nil
}

func tournamentRankTitleName(tournamentName string, finalRank int, participantCount int) (string, bool) {
	switch finalRank {
	case 1:
		return tournamentName + "冠军", true
	case 2:
		return tournamentName + "亚军", true
	case 3:
		return tournamentName + "季军", true
	case 4:
		if participantCount >= 4 {
			return tournamentName + "四强", true
		}
	}
	return "", false
}
