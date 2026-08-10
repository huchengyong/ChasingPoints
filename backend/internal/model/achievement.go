package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Achievement struct {
	Id              int64     `gorm:"primarykey"`
	Key             string    `gorm:"uniqueIndex:uk_key;size:64;not null"`
	Name            string    `gorm:"size:128;not null"`
	Description     string    `gorm:"size:512;not null;default:''"`
	Icon            string    `gorm:"size:512;not null;default:''"`
	Category        string    `gorm:"size:32;not null;default:''"`
	GameType        int       `gorm:"not null;default:0"`
	MetricKey       string    `gorm:"size:64;not null;default:''"`
	ProgressMode    string    `gorm:"size:32;not null;default:'sum'"`
	RewardTitleKey  string    `gorm:"size:64;not null;default:''"`
	RewardTitleName string    `gorm:"size:64;not null;default:''"`
	Sort            int       `gorm:"not null;default:0"`
	Status          int       `gorm:"not null;default:1"`
	Threshold       int       `gorm:"not null;default:1"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}

func (Achievement) TableName() string {
	return "achievements"
}

type UserAchievement struct {
	Id                 int64 `gorm:"primarykey"`
	UserId             int64 `gorm:"not null;index;uniqueIndex:uk_user_achievement,priority:1"`
	AchievementId      int64 `gorm:"not null;index;uniqueIndex:uk_user_achievement,priority:2"`
	Progress           int   `gorm:"not null;default:0"`
	Unlocked           int   `gorm:"not null;default:0"`
	RewardGranted      int   `gorm:"not null;default:0"`
	UnlockedAt         *time.Time
	UnlockedSourceType string `gorm:"size:32;not null;default:'';index:idx_user_achievement_unlock_source,priority:1"`
	UnlockedSourceId   int64  `gorm:"type:bigint unsigned;not null;default:0;index:idx_user_achievement_unlock_source,priority:2"`
	RewardGrantedAt    *time.Time
	CreatedAt          time.Time `gorm:"autoCreateTime"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime"`
}

func (UserAchievement) TableName() string {
	return "user_achievements"
}

type UserTitle struct {
	Id                     int64  `gorm:"primarykey"`
	UserId                 int64  `gorm:"not null;index;uniqueIndex:uk_user_title_source,priority:1"`
	TitleKey               string `gorm:"size:64;not null;default:'';uniqueIndex:uk_user_title_source,priority:2"`
	TitleName              string `gorm:"size:64;not null"`
	Source                 string `gorm:"size:32;not null;default:''"`
	SourceType             string `gorm:"size:32;not null;default:'';uniqueIndex:uk_user_title_source,priority:3"`
	SourceRefId            int64  `gorm:"type:bigint unsigned;not null;default:0;uniqueIndex:uk_user_title_source,priority:4"`
	SourceRefName          string `gorm:"size:128;not null;default:''"`
	GrantedByAchievementId *int64 `gorm:"type:bigint unsigned"`
	Equipped               int    `gorm:"not null;default:0"`
	EquippedAt             *time.Time
	GrantedAt              *time.Time
	CreatedAt              time.Time `gorm:"autoCreateTime"`
}

func (UserTitle) TableName() string {
	return "user_titles"
}

type AchievementModel struct {
	db *gorm.DB
}

func NewAchievementModel(db *gorm.DB) *AchievementModel {
	return &AchievementModel{db: db}
}

func (m *AchievementModel) FindAll() ([]Achievement, error) {
	var list []Achievement
	err := m.db.Order("id ASC").Find(&list).Error
	return list, err
}

func (m *AchievementModel) FindActive() ([]Achievement, error) {
	var list []Achievement
	err := m.db.Where("status = ?", 1).Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

func (m *AchievementModel) FindByCategory(category string) ([]Achievement, error) {
	var list []Achievement
	err := m.db.Where("category = ? AND status = ?", category, 1).Order("sort ASC, id ASC").Find(&list).Error
	return list, err
}

func (m *AchievementModel) FindByIds(ids []int64) ([]Achievement, error) {
	var list []Achievement
	if len(ids) == 0 {
		return list, nil
	}

	err := m.db.Where("id IN ?", ids).Order("id ASC").Find(&list).Error
	return list, err
}

type UserAchievementModel struct {
	db *gorm.DB
}

func NewUserAchievementModel(db *gorm.DB) *UserAchievementModel {
	return &UserAchievementModel{db: db}
}

func (m *UserAchievementModel) FindByUserId(userId int64) ([]UserAchievement, error) {
	var list []UserAchievement
	err := m.db.Where("user_id = ?", userId).Order("id ASC").Find(&list).Error
	return list, err
}

func (m *UserAchievementModel) FindByUserIds(userIds []int64) ([]UserAchievement, error) {
	if len(userIds) == 0 {
		return []UserAchievement{}, nil
	}
	var list []UserAchievement
	err := m.db.Where("user_id IN ?", userIds).Order("user_id ASC, achievement_id ASC").Find(&list).Error
	return list, err
}

func (m *UserAchievementModel) UpsertCareerRebuildSnapshot(snapshot *UserAchievement) error {
	if snapshot == nil {
		return nil
	}
	return m.db.Clauses(m.careerRebuildSnapshotConflict()).Create(snapshot).Error
}

func (m *UserAchievementModel) UpsertCareerRebuildSnapshots(snapshots []UserAchievement) error {
	if len(snapshots) == 0 {
		return nil
	}
	return m.db.Clauses(m.careerRebuildSnapshotConflict()).Create(&snapshots).Error
}

func (m *UserAchievementModel) careerRebuildSnapshotConflict() clause.OnConflict {
	incoming := func(column string) string {
		if m.db.Dialector.Name() == "mysql" {
			return "VALUES(" + column + ")"
		}
		return "excluded." + column
	}
	incomingProgress := incoming("progress")
	incomingUnlocked := incoming("unlocked")
	incomingUnlockedAt := incoming("unlocked_at")
	incomingSourceType := incoming("unlocked_source_type")
	incomingSourceID := incoming("unlocked_source_id")
	incomingRewardGranted := incoming("reward_granted")
	incomingRewardGrantedAt := incoming("reward_granted_at")
	earlierUnlock := incomingUnlockedAt + " IS NOT NULL AND (unlocked_at IS NULL OR " + incomingUnlockedAt + " < unlocked_at)"

	return clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "achievement_id"}},
		DoUpdates: clause.Set{
			{Column: clause.Column{Name: "progress"}, Value: gorm.Expr("CASE WHEN " + incomingProgress + " > progress THEN " + incomingProgress + " ELSE progress END")},
			{Column: clause.Column{Name: "unlocked_source_type"}, Value: gorm.Expr("CASE WHEN " + earlierUnlock + " THEN " + incomingSourceType + " ELSE unlocked_source_type END")},
			{Column: clause.Column{Name: "unlocked_source_id"}, Value: gorm.Expr("CASE WHEN " + earlierUnlock + " THEN " + incomingSourceID + " ELSE unlocked_source_id END")},
			{Column: clause.Column{Name: "unlocked_at"}, Value: gorm.Expr("CASE WHEN unlocked_at IS NULL THEN " + incomingUnlockedAt + " WHEN " + incomingUnlockedAt + " IS NULL THEN unlocked_at WHEN " + incomingUnlockedAt + " < unlocked_at THEN " + incomingUnlockedAt + " ELSE unlocked_at END")},
			{Column: clause.Column{Name: "unlocked"}, Value: gorm.Expr("CASE WHEN " + incomingUnlocked + " > unlocked THEN " + incomingUnlocked + " ELSE unlocked END")},
			{Column: clause.Column{Name: "reward_granted_at"}, Value: gorm.Expr("CASE WHEN reward_granted_at IS NULL THEN " + incomingRewardGrantedAt + " WHEN " + incomingRewardGrantedAt + " IS NULL THEN reward_granted_at WHEN " + incomingRewardGrantedAt + " < reward_granted_at THEN " + incomingRewardGrantedAt + " ELSE reward_granted_at END")},
			{Column: clause.Column{Name: "reward_granted"}, Value: gorm.Expr("CASE WHEN " + incomingRewardGranted + " > reward_granted THEN " + incomingRewardGranted + " ELSE reward_granted END")},
		},
	}
}

func (m *UserAchievementModel) FindUnlockedByUserId(userId int64) ([]UserAchievement, error) {
	var list []UserAchievement
	err := m.db.Where("user_id = ? AND unlocked = 1", userId).Order("unlocked_at DESC, id DESC").Find(&list).Error
	return list, err
}

func (m *UserAchievementModel) FindUnlockedByUserIdBetween(userId int64, startDate, endDate time.Time, limit int) ([]UserAchievement, error) {
	var list []UserAchievement
	query := m.db.Where(
		"user_id = ? AND unlocked = 1 AND unlocked_at IS NOT NULL AND unlocked_at >= ? AND unlocked_at <= ?",
		userId,
		startDate,
		endDate,
	).Order("unlocked_at DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&list).Error
	return list, err
}

func (m *UserAchievementModel) FindUnlockedByUserIdBetweenHalfOpen(userId int64, startDate, endExclusive time.Time, limit int) ([]UserAchievement, error) {
	var list []UserAchievement
	query := m.db.Where(
		"user_id = ? AND unlocked = 1 AND unlocked_at IS NOT NULL AND unlocked_at >= ? AND unlocked_at < ?",
		userId,
		startDate,
		endExclusive,
	).Order("unlocked_at DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&list).Error
	return list, err
}

func (m *UserAchievementModel) FindUnlockedBySource(userId int64, sourceType string, sourceId int64) ([]UserAchievement, error) {
	var list []UserAchievement
	err := m.db.Where(
		"user_id = ? AND unlocked = 1 AND unlocked_source_type = ? AND unlocked_source_id = ?",
		userId,
		sourceType,
		sourceId,
	).Order("unlocked_at ASC, id ASC").Find(&list).Error
	return list, err
}

func (m *UserAchievementModel) Unlock(userId, achievementId int64) error {
	now := time.Now()
	ua := &UserAchievement{
		UserId:        userId,
		AchievementId: achievementId,
		Unlocked:      1,
		UnlockedAt:    &now,
	}

	return m.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "achievement_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"unlocked":    1,
			"unlocked_at": &now,
		}),
	}).Create(ua).Error
}

type UserTitleModel struct {
	db *gorm.DB
}

func NewUserTitleModel(db *gorm.DB) *UserTitleModel {
	return &UserTitleModel{db: db}
}

func (m *UserTitleModel) FindByUserId(userId int64) ([]UserTitle, error) {
	var list []UserTitle
	err := m.db.Where("user_id = ?", userId).Order("id DESC").Find(&list).Error
	return list, err
}

func (m *UserTitleModel) FindAchievementTitlesByUserIds(userIds []int64) ([]UserTitle, error) {
	if len(userIds) == 0 {
		return []UserTitle{}, nil
	}
	var list []UserTitle
	err := m.db.
		Where("user_id IN ? AND source_type = ?", userIds, "achievement").
		Order("user_id ASC, source_ref_id ASC, id ASC").
		Find(&list).Error
	return list, err
}

func (m *UserTitleModel) FindEquippedByUserId(userId int64) (*UserTitle, error) {
	var title UserTitle
	err := m.db.Where("user_id = ? AND equipped = 1", userId).Order("equipped_at DESC, id DESC").First(&title).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &title, err
}

func (m *UserTitleModel) FindRecentPermanentByUserId(userId int64, limit int) ([]UserTitle, error) {
	var list []UserTitle
	query := m.db.Where("user_id = ? AND source_type IN ?", userId, []string{"season", "tournament"}).
		Order("COALESCE(granted_at, created_at) DESC, id DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&list).Error
	return list, err
}

func (m *UserTitleModel) FindPermanentByUserId(userId int64, page, pageSize int) ([]UserTitle, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	query := m.db.Model(&UserTitle{}).Where("user_id = ? AND source_type IN ?", userId, []string{"season", "tournament"})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []UserTitle
	err := query.Order("COALESCE(granted_at, created_at) DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

func (m *UserTitleModel) CountBySourceType(userId int64, sourceType string) (int64, error) {
	var total int64
	err := m.db.Model(&UserTitle{}).Where("user_id = ? AND source_type = ?", userId, sourceType).Count(&total).Error
	return total, err
}

func (m *UserTitleModel) Create(title *UserTitle) error {
	return m.db.Create(title).Error
}

func (m *UserTitleModel) EquipTitle(userId int64, titleId int64, equip bool) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		if equip {
			if err := tx.Model(&UserTitle{}).Where("user_id = ?", userId).Updates(map[string]any{
				"equipped":    0,
				"equipped_at": nil,
			}).Error; err != nil {
				return err
			}
			now := time.Now()
			result := tx.Model(&UserTitle{}).Where("user_id = ? AND id = ?", userId, titleId).Updates(map[string]any{
				"equipped":    1,
				"equipped_at": &now,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
			return nil
		}

		result := tx.Model(&UserTitle{}).Where("user_id = ? AND id = ?", userId, titleId).Updates(map[string]any{
			"equipped":    0,
			"equipped_at": nil,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

type AchievementProgressEvent struct {
	Id          int64     `gorm:"primarykey"`
	UserId      int64     `gorm:"not null;index:idx_achievement_progress_events_user_id;index:idx_achievement_progress_events_season_scope,priority:1;uniqueIndex:uk_user_source_metric,priority:1"`
	SourceType  string    `gorm:"size:32;not null;default:'';uniqueIndex:uk_user_source_metric,priority:2"`
	SourceId    int64     `gorm:"type:bigint unsigned;not null;default:0;uniqueIndex:uk_user_source_metric,priority:3"`
	GameType    int       `gorm:"not null;default:0;index:idx_achievement_progress_events_game_type;index:idx_achievement_progress_events_season_scope,priority:2"`
	MetricKey   string    `gorm:"size:64;not null;default:'';index:idx_achievement_progress_events_season_scope,priority:3;uniqueIndex:uk_user_source_metric,priority:4"`
	MetricValue int       `gorm:"not null;default:0"`
	OccurredAt  time.Time `gorm:"not null;index:idx_achievement_progress_events_season_scope,priority:4"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (AchievementProgressEvent) TableName() string {
	return "achievement_progress_events"
}

func NewAchievementProgressEvent(userId int64, sourceType string, sourceId int64, gameType int, metricKey string, metricValue int, occurredAt ...time.Time) *AchievementProgressEvent {
	at := time.Now()
	if len(occurredAt) > 0 && !occurredAt[0].IsZero() {
		at = occurredAt[0]
	}
	return &AchievementProgressEvent{
		UserId:      userId,
		SourceType:  sourceType,
		SourceId:    sourceId,
		GameType:    gameType,
		MetricKey:   metricKey,
		MetricValue: metricValue,
		OccurredAt:  at,
	}
}

type AchievementProgressEventModel struct {
	db *gorm.DB
}

func NewAchievementProgressEventModel(db *gorm.DB) *AchievementProgressEventModel {
	return &AchievementProgressEventModel{db: db}
}

func (m *AchievementProgressEventModel) CreateIfAbsent(event *AchievementProgressEvent) (bool, error) {
	result := m.db.Clauses(achievementProgressEventConflict()).Create(event)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (m *AchievementProgressEventModel) CreateCareerRebuildBatch(events []AchievementProgressEvent) (int64, error) {
	if len(events) == 0 {
		return 0, nil
	}
	result := m.db.Clauses(achievementProgressEventConflict()).Create(&events)
	return result.RowsAffected, result.Error
}

func achievementProgressEventConflict() clause.OnConflict {
	return clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "source_type"},
			{Name: "source_id"},
			{Name: "metric_key"},
		},
		DoNothing: true,
	}
}

func (m *AchievementProgressEventModel) ListAllForCareerRebuild() ([]AchievementProgressEvent, error) {
	var list []AchievementProgressEvent
	err := m.db.Order("occurred_at ASC, id ASC").Find(&list).Error
	return list, err
}

func (m *AchievementProgressEventModel) CountUsersBetween(startAt, endExclusive time.Time) (int64, int64, error) {
	query := m.db.Model(&AchievementProgressEvent{}).
		Where("occurred_at >= ? AND occurred_at < ?", startAt, endExclusive)
	var events int64
	if err := query.Count(&events).Error; err != nil {
		return 0, 0, err
	}
	var users int64
	if err := m.db.Model(&AchievementProgressEvent{}).
		Where("occurred_at >= ? AND occurred_at < ?", startAt, endExclusive).
		Distinct("user_id").
		Count(&users).Error; err != nil {
		return 0, 0, err
	}
	return events, users, nil
}

func (m *AchievementProgressEventModel) FindBySourceMetric(userId int64, sourceType string, sourceId int64, metricKey string) (*AchievementProgressEvent, error) {
	var event AchievementProgressEvent
	err := m.db.Where(
		"user_id = ? AND source_type = ? AND source_id = ? AND metric_key = ?",
		userId,
		sourceType,
		sourceId,
		metricKey,
	).First(&event).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &event, err
}

func (m *AchievementProgressEventModel) SumMetricsBetween(userId int64, gameType int, metricKeys []string, startAt, endAt time.Time) (map[string]int, error) {
	return m.SumMetricsBetweenWithTx(nil, userId, gameType, metricKeys, startAt, endAt)
}

func (m *AchievementProgressEventModel) SumMetricsBetweenWithTx(tx *gorm.DB, userId int64, gameType int, metricKeys []string, startAt, endAt time.Time) (map[string]int, error) {
	type metricTotal struct {
		MetricKey string
		Total     int
	}

	result := make(map[string]int, len(metricKeys))
	if len(metricKeys) == 0 {
		return result, nil
	}

	db := m.db
	if tx != nil {
		db = tx
	}
	var rows []metricTotal
	err := db.Model(&AchievementProgressEvent{}).
		Select("metric_key, COALESCE(SUM(metric_value), 0) AS total").
		Where("user_id = ? AND game_type = ? AND metric_key IN ? AND occurred_at >= ? AND occurred_at < ?", userId, gameType, metricKeys, startAt, endAt).
		Group("metric_key").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.MetricKey] = row.Total
	}
	return result, nil
}

type AchievementProgressUserGame struct {
	UserId   int64
	GameType int
}

func (m *AchievementProgressEventModel) ListUserGamesBetween(metricKeys []string, startAt, endAt time.Time) ([]AchievementProgressUserGame, error) {
	return m.ListUserGamesBetweenWithTx(nil, metricKeys, startAt, endAt)
}

func (m *AchievementProgressEventModel) ListUserGamesBetweenWithTx(tx *gorm.DB, metricKeys []string, startAt, endAt time.Time) ([]AchievementProgressUserGame, error) {
	var list []AchievementProgressUserGame
	if len(metricKeys) == 0 {
		return list, nil
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	err := db.Model(&AchievementProgressEvent{}).
		Select("DISTINCT user_id, game_type").
		Where("metric_key IN ? AND occurred_at >= ? AND occurred_at < ?", metricKeys, startAt, endAt).
		Order("user_id ASC, game_type ASC").
		Scan(&list).Error
	return list, err
}
