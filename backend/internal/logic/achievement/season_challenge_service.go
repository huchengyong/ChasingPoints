package achievement

import (
	"time"

	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

const (
	seasonChallengeArchiveBatchSize = 500
	SeasonChallengeMatchesKey       = "season_matches_20"
	SeasonChallengeWinsKey          = "season_wins_10"
	SeasonChallengeTournamentKey    = "season_tournament_finish_1"
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

	startAt, endExclusive, err := s.seasonBounds(season)
	if err != nil {
		return nil, err
	}
	metricKeys := seasonChallengeMetricKeys()
	totals, err := s.svcCtx.AchievementProgressEventModel.SumMetricsBetweenWithTx(
		tx,
		userId,
		gameType,
		metricKeys,
		startAt,
		endExclusive,
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
	return s.buildSnapshotBatchesWithTx(tx, season, archivedAt, nil)
}

func (s *SeasonChallengeService) ArchiveSeasonWithTx(tx *gorm.DB, season *model.Season, archivedAt time.Time) ([]model.SeasonChallengeSnapshot, error) {
	return s.buildSnapshotBatchesWithTx(tx, season, archivedAt, func(batch []model.SeasonChallengeSnapshot) error {
		return s.svcCtx.SeasonChallengeSnapshotModel.UpsertBatchWithTx(tx, batch)
	})
}

func (s *SeasonChallengeService) buildSnapshotBatchesWithTx(tx *gorm.DB, season *model.Season, archivedAt time.Time, persist func([]model.SeasonChallengeSnapshot) error) ([]model.SeasonChallengeSnapshot, error) {
	if season == nil {
		return []model.SeasonChallengeSnapshot{}, nil
	}
	if archivedAt.IsZero() {
		archivedAt = time.Now()
	}
	startAt, endExclusive, err := s.seasonBounds(season)
	if err != nil {
		return nil, err
	}
	metricKeys := seasonChallengeMetricKeys()
	allSnapshots := make([]model.SeasonChallengeSnapshot, 0)
	var afterUserID int64
	var afterGameType int
	for {
		pairs, err := s.svcCtx.AchievementProgressEventModel.ListUserGamesBatchBetweenWithTx(
			tx, metricKeys, startAt, endExclusive, afterUserID, afterGameType, seasonChallengeArchiveBatchSize,
		)
		if err != nil {
			return nil, err
		}
		if len(pairs) == 0 {
			break
		}
		totals, err := s.svcCtx.AchievementProgressEventModel.SumMetricsForUserGamesBetweenWithTx(tx, pairs, metricKeys, startAt, endExclusive)
		if err != nil {
			return nil, err
		}
		progressByPair := make(map[model.AchievementProgressUserGame]map[string]int, len(pairs))
		for _, total := range totals {
			pair := model.AchievementProgressUserGame{UserId: total.UserId, GameType: total.GameType}
			if progressByPair[pair] == nil {
				progressByPair[pair] = make(map[string]int, len(metricKeys))
			}
			progressByPair[pair][total.MetricKey] = total.Total
		}
		batch := make([]model.SeasonChallengeSnapshot, 0, len(pairs)*len(seasonChallengeDefinitions))
		for _, pair := range pairs {
			gameType := normalizeSeasonChallengeGameType(pair.GameType)
			progress := progressByPair[pair]
			for _, definition := range seasonChallengeDefinitions {
				value := progress[definition.MetricKey]
				if value < 0 {
					value = 0
				}
				completed := 0
				if value >= definition.Threshold {
					completed = 1
				}
				batch = append(batch, model.SeasonChallengeSnapshot{
					SeasonId: season.Id, UserId: pair.UserId, GameType: gameType,
					ChallengeKey: definition.Key, ChallengeName: definition.Name,
					Threshold: definition.Threshold, Progress: value, Completed: completed, ArchivedAt: archivedAt,
				})
			}
		}
		if persist != nil {
			if err := persist(batch); err != nil {
				return nil, err
			}
		}
		allSnapshots = append(allSnapshots, batch...)
		last := pairs[len(pairs)-1]
		afterUserID, afterGameType = last.UserId, last.GameType
		if len(pairs) < seasonChallengeArchiveBatchSize {
			break
		}
	}
	return allSnapshots, nil
}

func (s *SeasonChallengeService) FindArchived(userId, seasonId int64, gameType int) ([]model.SeasonChallengeSnapshot, error) {
	return s.svcCtx.SeasonChallengeSnapshotModel.FindByUserSeasonAndGameType(
		userId,
		seasonId,
		normalizeSeasonChallengeGameType(gameType),
	)
}

func (s *SeasonChallengeService) seasonBounds(season *model.Season) (time.Time, time.Time, error) {
	if s == nil || s.svcCtx == nil {
		return time.Time{}, time.Time{}, gorm.ErrInvalidDB
	}
	return seasonx.BoundsForConfig(s.svcCtx.Config.SeasonLifecycle, season)
}

func SeasonChallengeMetricKeys() []string {
	return []string{MetricMatchesTotal, MetricWinsTotal, MetricTournamentFinishTotal}
}

func seasonChallengeMetricKeys() []string {
	return SeasonChallengeMetricKeys()
}

func normalizeSeasonChallengeGameType(gameType int) int {
	if gameType < 1 || gameType > 4 {
		return 3
	}
	return gameType
}
