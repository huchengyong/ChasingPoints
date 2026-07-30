package achievement

import (
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

const (
	SeasonChallengeMatchesKey    = "season_matches_20"
	SeasonChallengeWinsKey       = "season_wins_10"
	SeasonChallengeTournamentKey = "season_tournament_finish_1"
)

type SeasonChallengeDefinition struct {
	Key         string
	Name        string
	Description string
	MetricKey   string
	Threshold   int
}

type SeasonChallengeProgress struct {
	SeasonChallengeDefinition
	Progress  int
	Completed bool
}

var seasonChallengeDefinitions = []SeasonChallengeDefinition{
	{
		Key:         SeasonChallengeMatchesKey,
		Name:        "赛季常客",
		Description: "完成 20 场有效排位比赛",
		MetricKey:   MetricMatchesTotal,
		Threshold:   20,
	},
	{
		Key:         SeasonChallengeWinsKey,
		Name:        "状态正盛",
		Description: "赢得 10 场有效排位比赛",
		MetricKey:   MetricWinsTotal,
		Threshold:   10,
	},
	{
		Key:         SeasonChallengeTournamentKey,
		Name:        "赛事参与者",
		Description: "完成 1 场正式赛事",
		MetricKey:   MetricTournamentFinishTotal,
		Threshold:   1,
	},
}

type SeasonChallengeService struct {
	svcCtx *svc.ServiceContext
}

func NewSeasonChallengeService(svcCtx *svc.ServiceContext) *SeasonChallengeService {
	return &SeasonChallengeService{svcCtx: svcCtx}
}

func FixedSeasonChallengeDefinitions() []SeasonChallengeDefinition {
	definitions := make([]SeasonChallengeDefinition, len(seasonChallengeDefinitions))
	copy(definitions, seasonChallengeDefinitions)
	return definitions
}

func (s *SeasonChallengeService) GetProgress(userId int64, season *model.Season, gameType int) ([]SeasonChallengeProgress, error) {
	return s.getProgressWithTx(nil, userId, season, normalizeSeasonChallengeGameType(gameType))
}

func (s *SeasonChallengeService) getProgressWithTx(tx *gorm.DB, userId int64, season *model.Season, gameType int) ([]SeasonChallengeProgress, error) {
	definitions := FixedSeasonChallengeDefinitions()
	result := make([]SeasonChallengeProgress, 0, len(definitions))
	if season == nil || userId <= 0 {
		for _, definition := range definitions {
			result = append(result, SeasonChallengeProgress{SeasonChallengeDefinition: definition})
		}
		return result, nil
	}

	metricKeys := seasonChallengeMetricKeys()
	totals, err := s.svcCtx.AchievementProgressEventModel.SumMetricsBetweenWithTx(
		tx,
		userId,
		gameType,
		metricKeys,
		season.StartDate,
		seasonChallengeEndExclusive(season.EndDate),
	)
	if err != nil {
		return nil, err
	}

	for _, definition := range definitions {
		progress := totals[definition.MetricKey]
		if progress < 0 {
			progress = 0
		}
		result = append(result, SeasonChallengeProgress{
			SeasonChallengeDefinition: definition,
			Progress:                  progress,
			Completed:                 progress >= definition.Threshold,
		})
	}
	return result, nil
}

func (s *SeasonChallengeService) BuildSnapshotsWithTx(tx *gorm.DB, season *model.Season, archivedAt time.Time) ([]model.SeasonChallengeSnapshot, error) {
	if season == nil {
		return []model.SeasonChallengeSnapshot{}, nil
	}
	if archivedAt.IsZero() {
		archivedAt = time.Now()
	}

	pairs, err := s.svcCtx.AchievementProgressEventModel.ListUserGamesBetweenWithTx(
		tx,
		seasonChallengeMetricKeys(),
		season.StartDate,
		seasonChallengeEndExclusive(season.EndDate),
	)
	if err != nil {
		return nil, err
	}

	snapshots := make([]model.SeasonChallengeSnapshot, 0, len(pairs)*len(seasonChallengeDefinitions))
	for _, pair := range pairs {
		gameType := normalizeSeasonChallengeGameType(pair.GameType)
		progressList, progressErr := s.getProgressWithTx(tx, pair.UserId, season, gameType)
		if progressErr != nil {
			return nil, progressErr
		}
		for _, item := range progressList {
			completed := 0
			if item.Completed {
				completed = 1
			}
			snapshots = append(snapshots, model.SeasonChallengeSnapshot{
				SeasonId:      season.Id,
				UserId:        pair.UserId,
				GameType:      gameType,
				ChallengeKey:  item.Key,
				ChallengeName: item.Name,
				Threshold:     item.Threshold,
				Progress:      item.Progress,
				Completed:     completed,
				ArchivedAt:    archivedAt,
			})
		}
	}
	return snapshots, nil
}

func (s *SeasonChallengeService) ArchiveSeasonWithTx(tx *gorm.DB, season *model.Season, archivedAt time.Time) ([]model.SeasonChallengeSnapshot, error) {
	snapshots, err := s.BuildSnapshotsWithTx(tx, season, archivedAt)
	if err != nil {
		return nil, err
	}
	if err := s.svcCtx.SeasonChallengeSnapshotModel.UpsertBatchWithTx(tx, snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (s *SeasonChallengeService) FindArchived(userId, seasonId int64, gameType int) ([]model.SeasonChallengeSnapshot, error) {
	return s.svcCtx.SeasonChallengeSnapshotModel.FindByUserSeasonAndGameType(
		userId,
		seasonId,
		normalizeSeasonChallengeGameType(gameType),
	)
}

func seasonChallengeMetricKeys() []string {
	return []string{MetricMatchesTotal, MetricWinsTotal, MetricTournamentFinishTotal}
}

func seasonChallengeEndExclusive(endDate time.Time) time.Time {
	return time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location()).AddDate(0, 0, 1)
}

func normalizeSeasonChallengeGameType(gameType int) int {
	if gameType < 1 || gameType > 4 {
		return 3
	}
	return gameType
}
