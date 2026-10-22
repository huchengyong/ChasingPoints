package model

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRanking 用户段位记录
type UserRanking struct {
	Id            int64     `gorm:"primarykey" json:"id"`
	UserId        int64     `gorm:"uniqueIndex:uniq_user_game_type;not null" json:"user_id"`
	GameType      int       `gorm:"uniqueIndex:uniq_user_game_type;not null;default:3" json:"game_type"`
	RankScore     int       `gorm:"not null;default:0" json:"rank_score"`
	RankLevel     int       `gorm:"not null;default:1" json:"rank_level"`
	TotalWins     int       `gorm:"not null;default:0" json:"total_wins"`
	TotalLosses   int       `gorm:"not null;default:0" json:"total_losses"`
	CurrentStreak int       `gorm:"not null;default:0" json:"current_streak"`
	MaxStreak     int       `gorm:"not null;default:0" json:"max_streak"`
	RatingVersion int       `gorm:"not null;default:1;index:idx_user_ranking_rating_version,priority:1" json:"rating_version"`
	RatedMatches  int       `gorm:"not null;default:0" json:"rated_matches"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserRanking) TableName() string {
	return "user_ranking"
}

// RatingVersionV2 表示段位行已切换到排位规则 V2。
const RatingVersionV2 = 2

// RankingDefaults 是新建段位行的初始值，由调用方按当前生效的规则版本决定。
type RankingDefaults struct {
	RankScore     int
	RankLevel     int
	RatingVersion int
}

// LegacyRankingDefaults 是旧规则下的新段位行。
func LegacyRankingDefaults() RankingDefaults {
	return RankingDefaults{RankScore: 0, RankLevel: 1, RatingVersion: 1}
}

// LeaderboardScope 决定全站榜的上榜资格：V2 生效后只收已切换且有 V2 计分场次的行。
type LeaderboardScope struct {
	RatingV2 bool
}

// RankRatingSeed 是排位规则 V2 的初始分：用 V2 规则重放切换前的全部排位得到。
type RankRatingSeed struct {
	Id          int64     `gorm:"primarykey" json:"id"`
	UserId      int64     `gorm:"not null;uniqueIndex:uk_rank_rating_seed_user_game,priority:1" json:"user_id"`
	GameType    int       `gorm:"not null;uniqueIndex:uk_rank_rating_seed_user_game,priority:2" json:"game_type"`
	SeedScore   int       `gorm:"not null" json:"seed_score"`
	SeedMatches int       `gorm:"not null;default:0" json:"seed_matches"`
	CutoffAt    time.Time `gorm:"not null" json:"cutoff_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (RankRatingSeed) TableName() string {
	return "rank_rating_seeds"
}

// RankConfig 段位配置
type RankConfig struct {
	Id       int    `gorm:"primarykey" json:"id"`
	Level    int    `gorm:"uniqueIndex;not null" json:"level"`
	Name     string `gorm:"size:20;not null" json:"name"`
	Icon     string `gorm:"size:100;not null" json:"icon"`
	MinScore int    `gorm:"not null" json:"min_score"`
}

func (RankConfig) TableName() string {
	return "rank_config"
}

// AchievementRewardConfig 特殊战绩奖励配置
type AchievementRewardConfig struct {
	Id              int    `gorm:"primarykey" json:"id"`
	GameType        int    `gorm:"not null" json:"game_type"`
	AchievementType string `gorm:"size:30;not null" json:"achievement_type"`
	Name            string `gorm:"size:50;not null" json:"name"`
	RewardScore     int    `gorm:"not null" json:"reward_score"`
}

func (AchievementRewardConfig) TableName() string {
	return "achievement_reward_config"
}

// RankChangeLog 段位分变更明细
type RankChangeLog struct {
	Id               int64     `gorm:"primarykey" json:"id"`
	UserId           int64     `gorm:"index:idx_user_effective_at;uniqueIndex:uniq_user_match_type;not null" json:"user_id"`
	MatchId          int64     `gorm:"index:idx_match_id;uniqueIndex:uniq_user_match_type;not null;default:0" json:"match_id"`
	ChangeType       string    `gorm:"size:32;uniqueIndex:uniq_user_match_type;not null;default:match_result" json:"change_type"`
	GameType         int       `gorm:"uniqueIndex:uniq_user_match_type;not null;default:3" json:"game_type"`
	Result           string    `gorm:"size:16;not null;default:" json:"result"`
	BaseScore        int       `gorm:"not null;default:0" json:"base_score"`
	AchievementScore int       `gorm:"not null;default:0" json:"achievement_score"`
	FinalChange      int       `gorm:"not null;default:0" json:"final_change"`
	BeforeScore      int       `gorm:"not null;default:0" json:"before_score"`
	AfterScore       int       `gorm:"not null;default:0" json:"after_score"`
	BeforeLevel      int       `gorm:"not null;default:1" json:"before_level"`
	AfterLevel       int       `gorm:"not null;default:1" json:"after_level"`
	OperatorUserId   int64     `gorm:"not null;default:0" json:"operator_user_id"`
	Remark           string    `gorm:"size:255;not null;default:" json:"remark"`
	EffectiveAt      time.Time `gorm:"not null;index:idx_user_effective_at" json:"effective_at"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (RankChangeLog) TableName() string {
	return "rank_change_logs"
}

const rankChangeTypeMatchResult = "match_result"

// RankChangeTypeRatingReset 是排位规则 V2 切换时写入的记录类型。
const RankChangeTypeRatingReset = "rating_reset"

// RankChangeTypeMatchResult 是对局结算写入的记录类型。
const RankChangeTypeMatchResult = rankChangeTypeMatchResult

var supportedRankingGameTypes = []int{1, 2, 3, 4}

const rankingInitializationMaxAttempts = 3

func SupportedRankingGameTypes() []int {
	return append([]int(nil), supportedRankingGameTypes...)
}

type rankingSchemaCapabilities struct {
	rankChangeLogGameTypeOnce sync.Once
	rankChangeLogHasGameType  bool
}

// RankingModel 段位相关数据库操作
type RankingModel struct {
	db           *gorm.DB
	capabilities *rankingSchemaCapabilities
}

func NewRankingModel(db *gorm.DB) *RankingModel {
	return &RankingModel{db: db, capabilities: &rankingSchemaCapabilities{}}
}

func (m *RankingModel) WithDB(db *gorm.DB) *RankingModel {
	if m == nil {
		return NewRankingModel(db)
	}
	capabilities := m.capabilities
	if capabilities == nil {
		capabilities = &rankingSchemaCapabilities{}
	}
	return &RankingModel{db: db, capabilities: capabilities}
}

func (m *RankingModel) resolveDB(tx *gorm.DB) (*gorm.DB, error) {
	if tx != nil {
		return tx, nil
	}
	if m.db != nil {
		return m.db, nil
	}
	return nil, errors.New("ranking db is nil")
}

func (m *RankingModel) rankChangeLogSupportsGameType(tx *gorm.DB) (bool, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return false, err
	}

	capabilities := m.capabilities
	if capabilities == nil {
		capabilities = &rankingSchemaCapabilities{}
		m.capabilities = capabilities
	}
	capabilities.rankChangeLogGameTypeOnce.Do(func() {
		capabilities.rankChangeLogHasGameType = db.Migrator().HasColumn(&RankChangeLog{}, "game_type")
	})

	return capabilities.rankChangeLogHasGameType, nil
}

func applyRankChangeLogGameTypeFilter(query *gorm.DB, enabled bool, gameType int) *gorm.DB {
	if !enabled {
		return query
	}
	return query.Where("game_type = ?", normalizeRankingGameType(gameType))
}

func buildLegacyRankChangeLogRows(logs []RankChangeLog) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0, len(logs))
	for _, log := range logs {
		rows = append(rows, map[string]interface{}{
			"user_id":           log.UserId,
			"match_id":          log.MatchId,
			"change_type":       log.ChangeType,
			"result":            log.Result,
			"base_score":        log.BaseScore,
			"achievement_score": log.AchievementScore,
			"final_change":      log.FinalChange,
			"before_score":      log.BeforeScore,
			"after_score":       log.AfterScore,
			"before_level":      log.BeforeLevel,
			"after_level":       log.AfterLevel,
			"operator_user_id":  log.OperatorUserId,
			"remark":            log.Remark,
			"effective_at":      log.EffectiveAt,
			"created_at":        log.CreatedAt,
		})
	}
	return rows
}

func leaderboardOrderClause(alias string) string {
	if alias == "" {
		return "rank_score DESC, total_wins DESC, updated_at ASC, user_id ASC"
	}
	return alias + ".rank_score DESC, " + alias + ".total_wins DESC, " + alias + ".updated_at ASC, " + alias + ".user_id ASC"
}

func leaderboardEligibilityCondition(alias string, scope LeaderboardScope) string {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	if scope.RatingV2 {
		return "(" + prefix + "rating_version = 2 AND " + prefix + "rated_matches > 0)"
	}
	return "(" + prefix + "total_wins + " + prefix + "total_losses) > 0"
}

func isLeaderboardEligible(ranking *UserRanking, scope LeaderboardScope) bool {
	if ranking == nil {
		return false
	}
	if scope.RatingV2 {
		return ranking.RatingVersion == RatingVersionV2 && ranking.RatedMatches > 0
	}
	return ranking.TotalWins+ranking.TotalLosses > 0
}

func higherRankingCondition(ranking *UserRanking) (string, []interface{}) {
	return "game_type = ? AND (" +
			"(rank_score > ?) OR " +
			"(rank_score = ? AND total_wins > ?) OR " +
			"(rank_score = ? AND total_wins = ? AND updated_at < ?) OR " +
			"(rank_score = ? AND total_wins = ? AND updated_at = ? AND user_id < ?))",
		[]interface{}{
			ranking.GameType,
			ranking.RankScore,
			ranking.RankScore, ranking.TotalWins,
			ranking.RankScore, ranking.TotalWins, ranking.UpdatedAt,
			ranking.RankScore, ranking.TotalWins, ranking.UpdatedAt, ranking.UserId,
		}
}

// ========== UserRanking 操作 ==========

// FindByUserId 根据用户ID查找默认展示球种（中式八球）的段位信息
func (m *RankingModel) FindByUserId(userId int64) (*UserRanking, error) {
	return m.FindByUserIdAndGameType(userId, defaultRankingGameType)
}

// FindByUserIdAndGameType 根据用户ID和球种查找段位信息
func (m *RankingModel) FindByUserIdAndGameType(userId int64, gameType int) (*UserRanking, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}

	var ranking UserRanking
	err = db.Where("user_id = ? AND game_type = ?", userId, normalizeRankingGameType(gameType)).First(&ranking).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &ranking, err
}

func (m *RankingModel) ListByUserId(userId int64) ([]UserRanking, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}
	var rankings []UserRanking
	err = db.Where("user_id = ?", userId).Order("game_type ASC").Find(&rankings).Error
	return rankings, err
}

func (m *RankingModel) GetCompetitiveWinSummary(userId int64) (totalMatches, wins int, err error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return 0, 0, err
	}
	var summary struct {
		TotalMatches int
		Wins         int
	}
	err = db.Model(&UserRanking{}).
		Select("COALESCE(SUM(total_wins + total_losses), 0) AS total_matches, COALESCE(SUM(total_wins), 0) AS wins").
		Where("user_id = ? AND game_type IN ?", userId, supportedRankingGameTypes).
		Find(&summary).Error
	return summary.TotalMatches, summary.Wins, err
}

// FindOrCreateByGameTypes 获取或按 defaults 初始化用户支持的全部球种段位记录。
func (m *RankingModel) FindOrCreateByGameTypes(userId int64, defaults RankingDefaults) ([]UserRanking, error) {
	var lastErr error
	for attempt := 1; attempt <= rankingInitializationMaxAttempts; attempt++ {
		rankings, err := m.findOrCreateByGameTypes(userId, defaults)
		if err == nil {
			return rankings, nil
		}
		lastErr = err
		if attempt == rankingInitializationMaxAttempts || !isRetryableRankingInitializationError(err) {
			return nil, err
		}
		time.Sleep(time.Duration(attempt) * 5 * time.Millisecond)
	}
	return nil, lastErr
}

func (m *RankingModel) findOrCreateByGameTypes(userId int64, defaults RankingDefaults) ([]UserRanking, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}

	var rankings []UserRanking
	if err = db.Where("user_id = ? AND game_type IN ?", userId, supportedRankingGameTypes).
		Order("game_type ASC").Find(&rankings).Error; err != nil {
		return nil, err
	}

	found := make(map[int]bool, len(rankings))
	for i := range rankings {
		found[rankings[i].GameType] = true
	}
	missing := make([]UserRanking, 0, len(supportedRankingGameTypes)-len(rankings))
	for _, gameType := range supportedRankingGameTypes {
		if found[gameType] {
			continue
		}
		missing = append(missing, UserRanking{
			UserId:        userId,
			GameType:      gameType,
			RankScore:     defaults.RankScore,
			RankLevel:     defaults.RankLevel,
			RatingVersion: defaults.RatingVersion,
		})
	}
	if len(missing) > 0 {
		if err = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&missing).Error; err != nil {
			return nil, err
		}
	}

	if err = db.Where("user_id = ? AND game_type IN ?", userId, supportedRankingGameTypes).
		Order("game_type ASC").Find(&rankings).Error; err != nil {
		return nil, err
	}
	if len(rankings) != len(supportedRankingGameTypes) {
		return nil, fmt.Errorf("expected %d rankings for user %d, got %d", len(supportedRankingGameTypes), userId, len(rankings))
	}
	return rankings, nil
}

func isRetryableRankingInitializationError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") ||
		strings.Contains(message, "database table is locked") ||
		strings.Contains(message, "deadlock found") ||
		strings.Contains(message, "lock wait timeout")
}

// Create 创建用户段位记录
func (m *RankingModel) Create(ranking *UserRanking) error {
	return m.db.Create(ranking).Error
}

// Update 更新用户段位记录
func (m *RankingModel) Update(ranking *UserRanking) error {
	return m.db.Save(ranking).Error
}

// UpdateRankingSnapshot 使用事务上下文更新当前段位快照
func (m *RankingModel) UpdateRankingSnapshot(tx *gorm.DB, ranking *UserRanking) error {
	if ranking == nil {
		return errors.New("ranking is nil")
	}

	db, err := m.resolveDB(tx)
	if err != nil {
		return err
	}

	return db.Save(ranking).Error
}

// FindOrCreate 查找或创建用户段位记录
func (m *RankingModel) FindOrCreate(userId int64) (*UserRanking, error) {
	return m.FindOrCreateWithTx(nil, userId, defaultRankingGameType)
}

// FindOrCreateWithTx 使用事务上下文查找或创建用户段位记录
func (m *RankingModel) FindOrCreateWithTx(tx *gorm.DB, userId int64, gameType int) (*UserRanking, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return nil, err
	}

	gameType = normalizeRankingGameType(gameType)
	ranking := &UserRanking{
		UserId:    userId,
		GameType:  gameType,
		RankScore: 0,
		RankLevel: 1,
	}
	// 使用 FirstOrCreate 原子操作解决并发问题
	err = db.Where("user_id = ? AND game_type = ?", userId, gameType).FirstOrCreate(ranking).Error
	if err != nil {
		return nil, err
	}
	return ranking, nil
}

// FindOrCreateByGameType 查找或创建指定球种段位记录
func (m *RankingModel) FindOrCreateByGameType(userId int64, gameType int) (*UserRanking, error) {
	return m.FindOrCreateWithTx(nil, userId, gameType)
}

// FindOrCreateForUpdateWithTx 按 defaults 创建缺失的段位行后加行锁读取，供需要读改写的结算与规则切换使用。
func (m *RankingModel) FindOrCreateForUpdateWithTx(tx *gorm.DB, userId int64, gameType int, defaults RankingDefaults) (*UserRanking, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return nil, err
	}
	gameType = normalizeRankingGameType(gameType)
	lockRow := func() (*UserRanking, error) {
		var ranking UserRanking
		err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND game_type = ?", userId, gameType).
			Take(&ranking).Error
		if err != nil {
			return nil, err
		}
		return &ranking, nil
	}
	// 段位行通常在注册时已建好：先加锁读取，缺失时才创建，避免每次结算都尝试插入。
	ranking, err := lockRow()
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		return ranking, err
	}
	created := &UserRanking{
		UserId:        userId,
		GameType:      gameType,
		RankScore:     defaults.RankScore,
		RankLevel:     defaults.RankLevel,
		RatingVersion: defaults.RatingVersion,
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(created).Error; err != nil {
		return nil, err
	}
	return lockRow()
}

// HasRankingsBeforeRatingVersion 判断是否还有未切换到 version 的段位行，走 idx_user_ranking_rating_version。
func (m *RankingModel) HasRankingsBeforeRatingVersion(tx *gorm.DB, version int) (bool, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return false, err
	}
	var ids []int64
	err = db.Model(&UserRanking{}).
		Where("rating_version < ?", version).
		Order("rating_version ASC, id ASC").
		Limit(1).
		Pluck("id", &ids).Error
	return len(ids) > 0, err
}

// ListRankingsBeforeRatingVersionForUpdateWithTx 加锁读取一批未切换到 version 的段位行。
func (m *RankingModel) ListRankingsBeforeRatingVersionForUpdateWithTx(tx *gorm.DB, version, limit int) ([]UserRanking, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return nil, err
	}
	var rankings []UserRanking
	err = db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("rating_version < ?", version).
		Order("rating_version ASC, id ASC").
		Limit(limit).
		Find(&rankings).Error
	return rankings, err
}

// SwitchRatingVersionWithTx 把仍低于目标版本的段位行切换为新分数；已切换的行不受影响并返回 false。
func (m *RankingModel) SwitchRatingVersionWithTx(tx *gorm.DB, rankingId int64, version, rankScore, rankLevel, ratedMatches int) (bool, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return false, err
	}
	result := db.Model(&UserRanking{}).
		Where("id = ? AND rating_version < ?", rankingId, version).
		Updates(map[string]interface{}{
			"rank_score":     rankScore,
			"rank_level":     rankLevel,
			"rating_version": version,
			"rated_matches":  ratedMatches,
		})
	return result.RowsAffected > 0, result.Error
}

// CreateRankChangeLogsIfAbsent 按唯一键幂等写入段位变更记录，返回实际写入条数。
func (m *RankingModel) CreateRankChangeLogsIfAbsent(tx *gorm.DB, logs []RankChangeLog) (int64, error) {
	if len(logs) == 0 {
		return 0, nil
	}
	db, err := m.resolveDB(tx)
	if err != nil {
		return 0, err
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(&logs, 500)
	return result.RowsAffected, result.Error
}

// FindRatingSeedsWithTx 读取指定段位行对应的 V2 初始分，按 (user_id, game_type) 索引。
func (m *RankingModel) FindRatingSeedsWithTx(tx *gorm.DB, rankings []UserRanking) (map[[2]int64]RankRatingSeed, error) {
	result := make(map[[2]int64]RankRatingSeed, len(rankings))
	if len(rankings) == 0 {
		return result, nil
	}
	db, err := m.resolveDB(tx)
	if err != nil {
		return nil, err
	}
	userIds := make([]int64, 0, len(rankings))
	seen := make(map[int64]bool, len(rankings))
	for _, ranking := range rankings {
		if !seen[ranking.UserId] {
			seen[ranking.UserId] = true
			userIds = append(userIds, ranking.UserId)
		}
	}
	var seeds []RankRatingSeed
	if err := db.Where("user_id IN ?", userIds).Find(&seeds).Error; err != nil {
		return nil, err
	}
	for _, seed := range seeds {
		result[[2]int64{seed.UserId, int64(seed.GameType)}] = seed
	}
	return result, nil
}

// ReplaceRatingSeedsWithTx 整表替换 V2 初始分。
func (m *RankingModel) ReplaceRatingSeedsWithTx(tx *gorm.DB, seeds []RankRatingSeed) error {
	db, err := m.resolveDB(tx)
	if err != nil {
		return err
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RankRatingSeed{}).Error; err != nil {
		return err
	}
	if len(seeds) == 0 {
		return nil
	}
	return db.CreateInBatches(&seeds, 500).Error
}

// ListRatingResetLogsWithTx 读取 V2 切换记录，供审计恢复每个段位行的起点。
func (m *RankingModel) ListRatingResetLogsWithTx(tx *gorm.DB) ([]RankChangeLog, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return nil, err
	}
	var logs []RankChangeLog
	err = db.Where("match_id = 0 AND change_type = ?", RankChangeTypeRatingReset).Order("id ASC").Find(&logs).Error
	return logs, err
}

// ListMatchRankChangesByMatchIdsWithTx 批量读取指定比赛的对局段位记录，供离线审计逐场比较。
func (m *RankingModel) ListMatchRankChangesByMatchIdsWithTx(tx *gorm.DB, matchIds []int64) ([]RankChangeLog, error) {
	if len(matchIds) == 0 {
		return []RankChangeLog{}, nil
	}
	db, err := m.resolveDB(tx)
	if err != nil {
		return nil, err
	}
	var logs []RankChangeLog
	err = db.Where("match_id IN ? AND change_type = ?", matchIds, rankChangeTypeMatchResult).Order("id ASC").Find(&logs).Error
	return logs, err
}

// ListRankingsPageWithTx 按主键游标分批读取段位行，供离线种子统计与审计使用。
func (m *RankingModel) ListRankingsPageWithTx(tx *gorm.DB, afterId int64, limit int) ([]UserRanking, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	db, err := m.resolveDB(tx)
	if err != nil {
		return nil, err
	}
	var rankings []UserRanking
	err = db.Where("id > ?", afterId).Order("id ASC").Limit(limit).Find(&rankings).Error
	return rankings, err
}

// DeleteAllRankings 清空全部段位快照
func (m *RankingModel) DeleteAllRankings(tx *gorm.DB) error {
	db, err := m.resolveDB(tx)
	if err != nil {
		return err
	}
	return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&UserRanking{}).Error
}

// CalculateLevel 根据排位分计算段位等级
func (m *RankingModel) CalculateLevel(score int) int {
	return RankLevelForScore(score)
}

// RankLevelForScore 根据排位分计算段位等级；阈值需与 rank_config.min_score 保持一致。
func RankLevelForScore(score int) int {
	switch {
	case score >= 2500:
		return 6 // 王者
	case score >= 2000:
		return 5 // 钻石
	case score >= 1500:
		return 4 // 铂金大师
	case score >= 1000:
		return 3 // 黄金球手
	case score >= 500:
		return 2 // 白银球手
	default:
		return 1 // 青铜球手
	}
}

// CreateRankChangeLogs 批量创建段位变更明细
func (m *RankingModel) CreateRankChangeLogs(tx *gorm.DB, logs []RankChangeLog) error {
	if len(logs) == 0 {
		return nil
	}

	db, err := m.resolveDB(tx)
	if err != nil {
		return err
	}

	supportsGameType, err := m.rankChangeLogSupportsGameType(tx)
	if err != nil {
		return err
	}
	if !supportsGameType {
		return db.Table((RankChangeLog{}).TableName()).Create(buildLegacyRankChangeLogRows(logs)).Error
	}

	return db.Create(&logs).Error
}

// DeleteAllRankChangeLogs 清空全部段位变更日志
func (m *RankingModel) DeleteAllRankChangeLogs(tx *gorm.DB) error {
	db, err := m.resolveDB(tx)
	if err != nil {
		return err
	}
	return db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RankChangeLog{}).Error
}

// FindMatchRankChangeByUser 获取某场比赛下当前用户的段位变更记录
func (m *RankingModel) FindMatchRankChangeByUser(matchId, userId int64) (*RankChangeLog, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}

	var log RankChangeLog
	err = db.Where("match_id = ? AND user_id = ? AND change_type = ?", matchId, userId, rankChangeTypeMatchResult).First(&log).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &log, err
}

// FindMatchRankChangeByUserAndGameType 获取某场比赛下当前用户指定球种的段位变更记录
func (m *RankingModel) FindMatchRankChangeByUserAndGameType(matchId, userId int64, gameType int) (*RankChangeLog, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}
	supportsGameType, err := m.rankChangeLogSupportsGameType(nil)
	if err != nil {
		return nil, err
	}

	var log RankChangeLog
	err = applyRankChangeLogGameTypeFilter(
		db.Where("match_id = ? AND user_id = ? AND change_type = ?", matchId, userId, rankChangeTypeMatchResult),
		supportsGameType,
		gameType,
	).First(&log).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &log, err
}

// ListRankChangesByMatch 获取某场比赛关联的全部段位变更记录
func (m *RankingModel) ListRankChangesByMatch(matchId int64) ([]RankChangeLog, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}

	var logs []RankChangeLog
	err = db.Where("match_id = ? AND change_type = ?", matchId, rankChangeTypeMatchResult).
		Order("effective_at ASC, id ASC").
		Find(&logs).Error
	return logs, err
}

// ListRankChangesByMatchAndGameType 获取某场比赛指定球种关联的全部段位变更记录
func (m *RankingModel) ListRankChangesByMatchAndGameType(matchId int64, gameType int) ([]RankChangeLog, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}
	supportsGameType, err := m.rankChangeLogSupportsGameType(nil)
	if err != nil {
		return nil, err
	}

	var logs []RankChangeLog
	err = applyRankChangeLogGameTypeFilter(
		db.Where("match_id = ? AND change_type = ?", matchId, rankChangeTypeMatchResult),
		supportsGameType,
		gameType,
	).
		Order("effective_at ASC, id ASC").
		Find(&logs).Error
	return logs, err
}

// ListRankChangesByUser 获取用户最近的段位变更记录
func (m *RankingModel) ListRankChangesByUser(userId int64, limit int) ([]RankChangeLog, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 30
	}

	var logs []RankChangeLog
	err = db.Where("user_id = ? AND change_type = ?", userId, rankChangeTypeMatchResult).
		Order("effective_at DESC, id DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// ListRankChangesByUserAndGameType 获取用户某球种最近的段位变更记录
func (m *RankingModel) ListRankChangesByUserAndGameType(userId int64, gameType int, limit int) ([]RankChangeLog, error) {
	return m.ListRankChangesByUserAndGameTypeSince(userId, gameType, time.Time{}, limit)
}

// ListRankChangesByUserAndGameTypeSince 获取用户某球种在 since 之后最近的段位变更记录；since 为零值时不限。
func (m *RankingModel) ListRankChangesByUserAndGameTypeSince(userId int64, gameType int, since time.Time, limit int) ([]RankChangeLog, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}
	supportsGameType, err := m.rankChangeLogSupportsGameType(nil)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	query := db.Where("user_id = ? AND change_type = ?", userId, rankChangeTypeMatchResult)
	if !since.IsZero() {
		query = query.Where("effective_at >= ?", since)
	}
	var logs []RankChangeLog
	err = applyRankChangeLogGameTypeFilter(query, supportsGameType, gameType).
		Order("effective_at DESC, id DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// MaxAnalysisRankChangeRows 是会员表现分析一次最多读取的对局结算记录数。
const MaxAnalysisRankChangeRows = 600

// ListMatchRankChangesForAnalysis 读取用户某球种自 since 起的对局结算记录（最新在前），只取分析需要的列。
// 走 (user_id, effective_at) 索引，行数上限为 MaxAnalysisRankChangeRows，不扫描完整历史。
func (m *RankingModel) ListMatchRankChangesForAnalysis(userId int64, gameType int, since time.Time, limit int) ([]RankChangeLog, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}
	supportsGameType, err := m.rankChangeLogSupportsGameType(nil)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > MaxAnalysisRankChangeRows {
		limit = MaxAnalysisRankChangeRows
	}
	query := db.Select("id", "match_id", "result", "before_score", "after_score", "remark", "effective_at").
		Where("user_id = ? AND change_type = ? AND effective_at >= ?", userId, rankChangeTypeMatchResult, since)
	var logs []RankChangeLog
	err = applyRankChangeLogGameTypeFilter(query, supportsGameType, gameType).
		Order("effective_at DESC, id DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// OpponentRatingRef 是同一场对局里对手的结算前排位分。
type OpponentRatingRef struct {
	MatchId     int64
	UserId      int64
	BeforeScore int
}

// ListOpponentBeforeScores 按对局 id 一次批量读取同场对手（非 userId 的一方）的结算前排位分，走 match_id 索引。
// 对手没有结算记录的对局不在结果中。调用方应保证 matchIds 有界。
func (m *RankingModel) ListOpponentBeforeScores(matchIds []int64, userId int64, gameType int) ([]OpponentRatingRef, error) {
	if len(matchIds) == 0 {
		return []OpponentRatingRef{}, nil
	}
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}
	supportsGameType, err := m.rankChangeLogSupportsGameType(nil)
	if err != nil {
		return nil, err
	}
	query := db.Model(&RankChangeLog{}).
		Select("match_id", "user_id", "before_score").
		Where("match_id IN ? AND change_type = ? AND user_id <> ?", matchIds, rankChangeTypeMatchResult, userId)
	var rows []OpponentRatingRef
	err = applyRankChangeLogGameTypeFilter(query, supportsGameType, gameType).Scan(&rows).Error
	return rows, err
}

// ListRankChangesByUserBetween 获取用户在时间范围内的段位变更记录
func (m *RankingModel) ListRankChangesByUserBetween(userId int64, start, end time.Time) ([]RankChangeLog, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}

	var logs []RankChangeLog
	err = db.Where("user_id = ? AND change_type = ? AND effective_at >= ? AND effective_at <= ?", userId, rankChangeTypeMatchResult, start, end).
		Order("effective_at ASC, id ASC").
		Find(&logs).Error
	return logs, err
}

// ListRankChangesByUserAndGameTypeBetween 获取用户某球种在时间范围内的段位变更记录
func (m *RankingModel) ListRankChangesByUserAndGameTypeBetween(userId int64, gameType int, start, end time.Time) ([]RankChangeLog, error) {
	return m.ListRankChangesByUserAndGameTypeBetweenWithTx(nil, userId, gameType, start, end)
}

func (m *RankingModel) ListRankChangesByUserAndGameTypeBetweenWithTx(tx *gorm.DB, userId int64, gameType int, start, end time.Time) ([]RankChangeLog, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return nil, err
	}
	supportsGameType, err := m.rankChangeLogSupportsGameType(tx)
	if err != nil {
		return nil, err
	}

	var logs []RankChangeLog
	err = applyRankChangeLogGameTypeFilter(
		db.Where("user_id = ? AND change_type = ? AND effective_at >= ? AND effective_at <= ?", userId, rankChangeTypeMatchResult, start, end),
		supportsGameType,
		gameType,
	).
		Order("effective_at ASC, id ASC").
		Find(&logs).Error
	return logs, err
}

// ListRankChangesByUserAndGameTypeBetweenHalfOpen returns logs in [start, endExclusive).
func (m *RankingModel) ListRankChangesByUserAndGameTypeBetweenHalfOpen(userId int64, gameType int, start, endExclusive time.Time) ([]RankChangeLog, error) {
	return m.ListRankChangesByUserAndGameTypeBetweenHalfOpenWithTx(nil, userId, gameType, start, endExclusive)
}

func (m *RankingModel) ListRankChangesByUserAndGameTypeBetweenHalfOpenWithTx(tx *gorm.DB, userId int64, gameType int, start, endExclusive time.Time) ([]RankChangeLog, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return nil, err
	}
	supportsGameType, err := m.rankChangeLogSupportsGameType(tx)
	if err != nil {
		return nil, err
	}

	var logs []RankChangeLog
	err = applyRankChangeLogGameTypeFilter(
		db.Where("user_id = ? AND change_type = ? AND effective_at >= ? AND effective_at < ?", userId, rankChangeTypeMatchResult, start, endExclusive),
		supportsGameType,
		gameType,
	).
		Order("effective_at ASC, id ASC").
		Find(&logs).Error
	return logs, err
}

// SumPositiveRankChangesByUserAndGameTypeBetween 汇总时间范围内的正向涨分
func (m *RankingModel) SumPositiveRankChangesByUserAndGameTypeBetween(
	tx *gorm.DB,
	userId int64,
	gameType int,
	start, end time.Time,
) (int, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return 0, err
	}
	supportsGameType, err := m.rankChangeLogSupportsGameType(tx)
	if err != nil {
		return 0, err
	}

	var total int
	query := db.Model(&RankChangeLog{}).
		Select("COALESCE(SUM(CASE WHEN final_change > 0 THEN final_change ELSE 0 END), 0)").
		Where("user_id = ? AND change_type = ? AND effective_at >= ? AND effective_at < ?",
			userId,
			rankChangeTypeMatchResult,
			start,
			end,
		)
	query = applyRankChangeLogGameTypeFilter(query, supportsGameType, gameType)
	err = query.Scan(&total).Error
	return total, err
}

// SumPositiveAchievementRankChangesByUserAndGameTypeBetween 汇总时间范围内会员特殊战绩正向涨分
func (m *RankingModel) SumPositiveAchievementRankChangesByUserAndGameTypeBetween(
	tx *gorm.DB,
	userId int64,
	gameType int,
	start, end time.Time,
) (int, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return 0, err
	}
	supportsGameType, err := m.rankChangeLogSupportsGameType(tx)
	if err != nil {
		return 0, err
	}

	var total int
	query := db.Model(&RankChangeLog{}).
		Select("COALESCE(SUM(CASE WHEN achievement_score > 0 THEN achievement_score ELSE 0 END), 0)").
		Where("user_id = ? AND change_type = ? AND effective_at >= ? AND effective_at < ?",
			userId,
			rankChangeTypeMatchResult,
			start,
			end,
		)
	query = applyRankChangeLogGameTypeFilter(query, supportsGameType, gameType)
	err = query.Scan(&total).Error
	return total, err
}

// FindLatestRankChangeBefore 获取某个时间点之前最近的一条段位变更记录
func (m *RankingModel) FindLatestRankChangeBefore(userId int64, at time.Time) (*RankChangeLog, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}

	var log RankChangeLog
	err = db.Where("user_id = ? AND change_type = ? AND effective_at < ?", userId, rankChangeTypeMatchResult, at).
		Order("effective_at DESC, id DESC").
		First(&log).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &log, err
}

// FindLatestRankChangeBeforeByGameType 获取某个时间点之前最近的一条指定球种段位变更记录
func (m *RankingModel) FindLatestRankChangeBeforeByGameType(userId int64, gameType int, at time.Time) (*RankChangeLog, error) {
	return m.FindLatestRankChangeBeforeByGameTypeWithTx(nil, userId, gameType, at)
}

func (m *RankingModel) FindLatestRankChangeBeforeByGameTypeWithTx(tx *gorm.DB, userId int64, gameType int, at time.Time) (*RankChangeLog, error) {
	db, err := m.resolveDB(tx)
	if err != nil {
		return nil, err
	}
	supportsGameType, err := m.rankChangeLogSupportsGameType(tx)
	if err != nil {
		return nil, err
	}

	var log RankChangeLog
	err = applyRankChangeLogGameTypeFilter(
		db.Where("user_id = ? AND change_type = ? AND effective_at < ?", userId, rankChangeTypeMatchResult, at),
		supportsGameType,
		gameType,
	).
		Order("effective_at DESC, id DESC").
		First(&log).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &log, err
}

// ========== RankConfig 操作 ==========

// GetAllRankConfigs 获取所有段位配置
func (m *RankingModel) GetAllRankConfigs() ([]RankConfig, error) {
	var configs []RankConfig
	err := m.db.Order("level ASC").Find(&configs).Error
	return configs, err
}

// GetRankConfigByLevel 根据等级获取段位配置
func (m *RankingModel) GetRankConfigByLevel(level int) (*RankConfig, error) {
	var config RankConfig
	err := m.db.Where("level = ?", level).First(&config).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &config, err
}

// GetAchievementRewardConfigs 获取指定球种的特殊战绩奖励配置
func (m *RankingModel) GetAchievementRewardConfigs(gameType int) ([]AchievementRewardConfig, error) {
	db, err := m.resolveDB(nil)
	if err != nil {
		return nil, err
	}

	var configs []AchievementRewardConfig
	err = db.Where("game_type = ?", gameType).Order("id ASC").Find(&configs).Error
	return configs, err
}

// ========== 排行榜操作 ==========

// LeaderboardEntry 排行榜条目（用于JOIN查询结果）
type LeaderboardEntry struct {
	Rank        int
	UserId      int64
	Nickname    string
	Avatar      string
	RankLevel   int
	RankScore   int
	TotalWins   int
	TotalLosses int
}

// GetLeaderboard 获取排行榜
func (m *RankingModel) GetLeaderboard(offset, limit int) ([]LeaderboardEntry, error) {
	return m.GetLeaderboardByGameType(defaultRankingGameType, offset, limit, LeaderboardScope{})
}

// GetLeaderboardByGameType 获取指定球种排行榜
func (m *RankingModel) GetLeaderboardByGameType(gameType, offset, limit int, scope LeaderboardScope) ([]LeaderboardEntry, error) {
	var entries []LeaderboardEntry
	err := m.db.Table("user_ranking ur").
		Select("ur.user_id, u.nickname, u.avatar, ur.rank_level, ur.rank_score, ur.total_wins, ur.total_losses").
		Joins("LEFT JOIN users u ON ur.user_id = u.id").
		Where("ur.game_type = ?", normalizeRankingGameType(gameType)).
		Where(leaderboardEligibilityCondition("ur", scope)).
		Order(leaderboardOrderClause("ur")).
		Offset(offset).
		Limit(limit).
		Scan(&entries).Error
	return entries, err
}

// GetTopThree 获取前三名
func (m *RankingModel) GetTopThree() ([]LeaderboardEntry, error) {
	return m.GetLeaderboard(0, 3)
}

// GetTopThreeByGameType 获取指定球种前三名
func (m *RankingModel) GetTopThreeByGameType(gameType int, scope LeaderboardScope) ([]LeaderboardEntry, error) {
	return m.GetLeaderboardByGameType(gameType, 0, 3, scope)
}

// GetLeaderboardCount 获取排行榜总人数
func (m *RankingModel) GetLeaderboardCount() (int64, error) {
	return m.GetLeaderboardCountByGameType(defaultRankingGameType, LeaderboardScope{})
}

// GetLeaderboardCountByGameType 获取指定球种排行榜总人数
func (m *RankingModel) GetLeaderboardCountByGameType(gameType int, scope LeaderboardScope) (int64, error) {
	var count int64
	err := m.db.Model(&UserRanking{}).
		Where("game_type = ?", normalizeRankingGameType(gameType)).
		Where(leaderboardEligibilityCondition("", scope)).
		Count(&count).Error
	return count, err
}

// FindRankLevelsByUserIds 批量读取用户在指定球种的当前段位等级，只查必要列；没有排位记录的用户不在结果中。
func (m *RankingModel) FindRankLevelsByUserIds(userIds []int64, gameType int) (map[int64]int, error) {
	levels := make(map[int64]int, len(userIds))
	const batchSize = 500
	for start := 0; start < len(userIds); start += batchSize {
		end := start + batchSize
		if end > len(userIds) {
			end = len(userIds)
		}
		var list []UserRanking
		if err := m.db.Select("user_id", "rank_level").
			Where("game_type = ? AND user_id IN ?", gameType, userIds[start:end]).
			Find(&list).Error; err != nil {
			return nil, err
		}
		for _, ranking := range list {
			levels[ranking.UserId] = ranking.RankLevel
		}
	}
	return levels, nil
}

// GetUserRanking 获取用户的排名
func (m *RankingModel) GetUserRanking(userId int64) (int, *LeaderboardEntry, error) {
	return m.GetUserRankingByGameType(userId, defaultRankingGameType, LeaderboardScope{})
}

// GetUserRankingByGameType 获取用户在指定球种下的排名
func (m *RankingModel) GetUserRankingByGameType(userId int64, gameType int, scope LeaderboardScope) (int, *LeaderboardEntry, error) {
	// 获取用户的段位信息
	ranking, err := m.FindByUserIdAndGameType(userId, gameType)
	if err != nil {
		return 0, nil, err
	}
	if !isLeaderboardEligible(ranking, scope) {
		return 0, nil, nil
	}

	// 计算排名：统计排位分比当前用户高的人数 + 1
	var rank int64
	condition, args := higherRankingCondition(ranking)
	err = m.db.Model(&UserRanking{}).
		Where(leaderboardEligibilityCondition("", scope)).
		Where(condition, args...).
		Count(&rank).Error
	if err != nil {
		return 0, nil, err
	}

	// 获取用户信息
	var entry LeaderboardEntry
	err = m.db.Table("user_ranking ur").
		Select("ur.user_id, u.nickname, u.avatar, ur.rank_level, ur.rank_score, ur.total_wins, ur.total_losses").
		Joins("LEFT JOIN users u ON ur.user_id = u.id").
		Where("ur.user_id = ? AND ur.game_type = ?", userId, normalizeRankingGameType(gameType)).
		Where(leaderboardEligibilityCondition("ur", scope)).
		Scan(&entry).Error
	if err != nil {
		return 0, nil, err
	}

	return int(rank) + 1, &entry, nil
}

const defaultRankingGameType = 3

func normalizeRankingGameType(gameType int) int {
	if gameType <= 0 {
		return defaultRankingGameType
	}
	return gameType
}
