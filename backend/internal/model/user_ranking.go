package model

import (
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)

// UserRanking 用户段位记录
type UserRanking struct {
	Id            int64     `gorm:"primarykey" json:"id"`
	UserId        int64     `gorm:"uniqueIndex;not null" json:"user_id"`
	GameType      int       `gorm:"uniqueIndex;not null;default:3" json:"game_type"`
	RankScore     int       `gorm:"not null;default:0" json:"rank_score"`
	RankLevel     int       `gorm:"not null;default:1" json:"rank_level"`
	TotalWins     int       `gorm:"not null;default:0" json:"total_wins"`
	TotalLosses   int       `gorm:"not null;default:0" json:"total_losses"`
	CurrentStreak int       `gorm:"not null;default:0" json:"current_streak"`
	MaxStreak     int       `gorm:"not null;default:0" json:"max_streak"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserRanking) TableName() string {
	return "user_ranking"
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

// RankingModel 段位相关数据库操作
type RankingModel struct {
	db                        *gorm.DB
	rankChangeLogGameTypeOnce sync.Once
	rankChangeLogHasGameType  bool
}

func NewRankingModel(db *gorm.DB) *RankingModel {
	return &RankingModel{db: db}
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

	m.rankChangeLogGameTypeOnce.Do(func() {
		m.rankChangeLogHasGameType = db.Migrator().HasColumn(&RankChangeLog{}, "game_type")
	})

	return m.rankChangeLogHasGameType, nil
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

func leaderboardEligibilityCondition(alias string) string {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	return "(" + prefix + "total_wins + " + prefix + "total_losses) > 0"
}

func isLeaderboardEligible(ranking *UserRanking) bool {
	if ranking == nil {
		return false
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

// UpdateAfterMatch 对局结束后更新段位信息
func (m *RankingModel) UpdateAfterMatch(userId int64, isWin bool, achievementScores int) error {
	ranking, err := m.FindOrCreate(userId)
	if err != nil {
		return err
	}

	// 基础段位分变化
	scoreChange := 0
	if isWin {
		scoreChange = 20 + achievementScores // 胜利 +20 + 特殊战绩奖励
		ranking.TotalWins++
		if ranking.CurrentStreak >= 0 {
			ranking.CurrentStreak++
		} else {
			ranking.CurrentStreak = 1
		}
		if ranking.CurrentStreak > ranking.MaxStreak {
			ranking.MaxStreak = ranking.CurrentStreak
		}
	} else {
		// 失败 -10，但如果已经是0分则不减
		if ranking.RankScore > 0 {
			scoreChange = -10
		}
		ranking.TotalLosses++
		if ranking.CurrentStreak <= 0 {
			ranking.CurrentStreak--
		} else {
			ranking.CurrentStreak = -1
		}
	}

	ranking.RankScore += scoreChange
	if ranking.RankScore < 0 {
		ranking.RankScore = 0
	}

	// 更新段位等级
	ranking.RankLevel = m.CalculateLevel(ranking.RankScore)

	return m.Update(ranking)
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
	switch {
	case score >= 2000:
		return 5 // 钻石王者
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

	var logs []RankChangeLog
	err = applyRankChangeLogGameTypeFilter(
		db.Where("user_id = ? AND change_type = ?", userId, rankChangeTypeMatchResult),
		supportsGameType,
		gameType,
	).
		Order("effective_at DESC, id DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
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
	return m.GetLeaderboardByGameType(defaultRankingGameType, offset, limit)
}

// GetLeaderboardByGameType 获取指定球种排行榜
func (m *RankingModel) GetLeaderboardByGameType(gameType, offset, limit int) ([]LeaderboardEntry, error) {
	var entries []LeaderboardEntry
	err := m.db.Table("user_ranking ur").
		Select("ur.user_id, u.nickname, u.avatar, ur.rank_level, ur.rank_score, ur.total_wins, ur.total_losses").
		Joins("LEFT JOIN users u ON ur.user_id = u.id").
		Where("ur.game_type = ?", normalizeRankingGameType(gameType)).
		Where(leaderboardEligibilityCondition("ur")).
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
func (m *RankingModel) GetTopThreeByGameType(gameType int) ([]LeaderboardEntry, error) {
	return m.GetLeaderboardByGameType(gameType, 0, 3)
}

// GetLeaderboardCount 获取排行榜总人数
func (m *RankingModel) GetLeaderboardCount() (int64, error) {
	return m.GetLeaderboardCountByGameType(defaultRankingGameType)
}

// GetLeaderboardCountByGameType 获取指定球种排行榜总人数
func (m *RankingModel) GetLeaderboardCountByGameType(gameType int) (int64, error) {
	var count int64
	err := m.db.Model(&UserRanking{}).
		Where("game_type = ?", normalizeRankingGameType(gameType)).
		Where(leaderboardEligibilityCondition("")).
		Count(&count).Error
	return count, err
}

// GetUserRanking 获取用户的排名
func (m *RankingModel) GetUserRanking(userId int64) (int, *LeaderboardEntry, error) {
	return m.GetUserRankingByGameType(userId, defaultRankingGameType)
}

// GetUserRankingByGameType 获取用户在指定球种下的排名
func (m *RankingModel) GetUserRankingByGameType(userId int64, gameType int) (int, *LeaderboardEntry, error) {
	// 获取用户的段位信息
	ranking, err := m.FindByUserIdAndGameType(userId, gameType)
	if err != nil {
		return 0, nil, err
	}
	if !isLeaderboardEligible(ranking) {
		return 0, nil, nil
	}

	// 计算排名：统计排位分比当前用户高的人数 + 1
	var rank int64
	condition, args := higherRankingCondition(ranking)
	err = m.db.Model(&UserRanking{}).
		Where(leaderboardEligibilityCondition("")).
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
		Where(leaderboardEligibilityCondition("ur")).
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
