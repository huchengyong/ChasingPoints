package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MatchParticipantResult 是 completed match 的用户视角读投影。
type MatchParticipantResult struct {
	Id                 int64     `gorm:"primarykey" json:"id"`
	MatchId            int64     `gorm:"not null;uniqueIndex:uk_match_participant_result,priority:1" json:"match_id"`
	UserId             int64     `gorm:"not null;uniqueIndex:uk_match_participant_result,priority:2" json:"user_id"`
	OpponentUserId     int64     `gorm:"not null;default:0" json:"opponent_user_id"`
	OpponentNameKey    string    `gorm:"size:128;not null" json:"opponent_name_key"`
	OpponentName       string    `gorm:"size:128;not null" json:"opponent_name"`
	OpponentAvatar     string    `gorm:"size:512;not null" json:"opponent_avatar"`
	GameType           int       `gorm:"not null" json:"game_type"`
	MatchMode          string    `gorm:"size:20;not null" json:"match_mode"`
	Result             int       `gorm:"not null" json:"result"`
	MyScore            int       `gorm:"not null" json:"my_score"`
	OpponentScore      int       `gorm:"not null" json:"opponent_score"`
	CompletedAt        time.Time `gorm:"not null" json:"completed_at"`
	DurationSeconds    int64     `gorm:"not null" json:"duration_seconds"`
	MatchHighScore     int       `gorm:"not null" json:"match_high_score"`
	BestBreak          int       `gorm:"not null" json:"best_break"`
	OpponentRankBucket string    `gorm:"size:32;not null" json:"opponent_rank_bucket"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (MatchParticipantResult) TableName() string { return "match_participant_results" }

// UserCompetitiveStats 保存总体(game_type=0)与分球种快照。
type UserCompetitiveStats struct {
	Id                 int64      `gorm:"primarykey" json:"id"`
	UserId             int64      `gorm:"not null;uniqueIndex:uk_user_competitive_stats_user_game,priority:1" json:"user_id"`
	GameType           int        `gorm:"not null;uniqueIndex:uk_user_competitive_stats_user_game,priority:2" json:"game_type"`
	TotalMatches       int        `gorm:"not null" json:"total_matches"`
	Wins               int        `gorm:"not null" json:"wins"`
	Losses             int        `gorm:"not null" json:"losses"`
	Draws              int        `gorm:"not null" json:"draws"`
	CurrentWinStreak   int        `gorm:"not null" json:"current_win_streak"`
	MaxWinStreak       int        `gorm:"not null" json:"max_win_streak"`
	HighestScore       int        `gorm:"not null" json:"highest_score"`
	HighestBreak       int        `gorm:"not null" json:"highest_break"`
	DurationCount      int        `gorm:"not null" json:"duration_count"`
	DurationSumSeconds int64      `gorm:"not null" json:"duration_sum_seconds"`
	DurationMinSeconds int64      `gorm:"not null" json:"duration_min_seconds"`
	DurationMaxSeconds int64      `gorm:"not null" json:"duration_max_seconds"`
	LastMatchId        int64      `gorm:"not null" json:"last_match_id"`
	LastMatchAt        *time.Time `json:"last_match_at"`
	Revision           int64      `gorm:"not null" json:"revision"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserCompetitiveStats) TableName() string { return "user_competitive_stats" }

// UserOpponentStats 保存稳定对手身份的总体和分球种交锋快照。
type UserOpponentStats struct {
	Id               int64      `gorm:"primarykey" json:"id"`
	UserId           int64      `gorm:"not null;uniqueIndex:uk_user_opponent_stats_identity_game,priority:1" json:"user_id"`
	OpponentUserId   int64      `gorm:"not null;uniqueIndex:uk_user_opponent_stats_identity_game,priority:2" json:"opponent_user_id"`
	OpponentNameKey  string     `gorm:"size:128;not null;uniqueIndex:uk_user_opponent_stats_identity_game,priority:3" json:"opponent_name_key"`
	OpponentName     string     `gorm:"size:128;not null" json:"opponent_name"`
	OpponentAvatar   string     `gorm:"size:512;not null" json:"opponent_avatar"`
	GameType         int        `gorm:"not null;uniqueIndex:uk_user_opponent_stats_identity_game,priority:4" json:"game_type"`
	TotalMatches     int        `gorm:"not null" json:"total_matches"`
	Wins             int        `gorm:"not null" json:"wins"`
	Losses           int        `gorm:"not null" json:"losses"`
	Draws            int        `gorm:"not null" json:"draws"`
	ScoreDiffSum     int64      `gorm:"not null" json:"score_diff_sum"`
	CurrentWinStreak int        `gorm:"not null" json:"current_win_streak"`
	MaxWinStreak     int        `gorm:"not null" json:"max_win_streak"`
	LastMatchId      int64      `gorm:"not null" json:"last_match_id"`
	LastMatchAt      *time.Time `json:"last_match_at"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserOpponentStats) TableName() string { return "user_opponent_stats" }

type UserOpponentStrengthBucket struct {
	Id         int64     `gorm:"primarykey" json:"id"`
	UserId     int64     `gorm:"not null;uniqueIndex:uk_user_opponent_strength_bucket,priority:1" json:"user_id"`
	GameType   int       `gorm:"not null;uniqueIndex:uk_user_opponent_strength_bucket,priority:2" json:"game_type"`
	RankBucket string    `gorm:"size:32;not null;uniqueIndex:uk_user_opponent_strength_bucket,priority:3" json:"rank_bucket"`
	Matches    int       `gorm:"not null" json:"matches"`
	Wins       int       `gorm:"not null" json:"wins"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (UserOpponentStrengthBucket) TableName() string { return "user_opponent_strength_buckets" }

type CompetitiveReadModelRebuildCheckpoint struct {
	Id                    int64      `gorm:"primarykey" json:"id"`
	JobName               string     `gorm:"size:64;not null;uniqueIndex:uk_competitive_read_model_rebuild_job" json:"job_name"`
	CursorMatchId         int64      `gorm:"not null" json:"cursor_match_id"`
	CursorCompletedAt     *time.Time `json:"cursor_completed_at"`
	BackfillCursorMatchId int64      `gorm:"not null" json:"backfill_cursor_match_id"`
	BackfillCompleted     bool       `gorm:"not null" json:"backfill_completed"`
	Paused                bool       `gorm:"not null" json:"paused"`
	LastError             string     `gorm:"size:512;not null" json:"last_error"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	CreatedAt             time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (CompetitiveReadModelRebuildCheckpoint) TableName() string {
	return "competitive_read_model_rebuild_checkpoints"
}

type CompetitiveReadModel struct {
	db *gorm.DB
}

func NewCompetitiveReadModel(db *gorm.DB) *CompetitiveReadModel {
	return &CompetitiveReadModel{db: db}
}

func (m *CompetitiveReadModel) dbWith(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return m.db
}

func (m *CompetitiveReadModel) InsertParticipantIfAbsentWithTx(tx *gorm.DB, result *MatchParticipantResult) (bool, error) {
	if result == nil {
		return false, nil
	}
	created := m.dbWith(tx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "match_id"}, {Name: "user_id"}},
		DoNothing: true,
	}).Create(result)
	return created.RowsAffected > 0, created.Error
}

func (m *CompetitiveReadModel) FindParticipantByMatchAndUser(matchId, userId int64) (*MatchParticipantResult, error) {
	var result MatchParticipantResult
	err := m.db.Where("match_id = ? AND user_id = ?", matchId, userId).First(&result).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &result, err
}

type ParticipantMatchListRow struct {
	MatchId        int64
	OpponentUserId int64
	OpponentName   string
	OpponentAvatar string
	GameType       int
	MatchMode      string
	Result         int
	MyScore        int
	OpponentScore  int
	CompletedAt    time.Time
	MatchTime      time.Time
	Visibility     string
	FinishState    string
}

func (m *CompetitiveReadModel) ListParticipantMatchPage(userId int64, gameType, result, offset, limit int) ([]ParticipantMatchListRow, int64, error) {
	return m.ListParticipantMatchPageWithTx(nil, userId, gameType, result, offset, limit)
}

func (m *CompetitiveReadModel) ListParticipantMatchPageWithTx(tx *gorm.DB, userId int64, gameType, result, offset, limit int) ([]ParticipantMatchListRow, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	query := m.dbWith(tx).Table("match_participant_results AS p").
		Joins("JOIN matches AS m ON m.id = p.match_id").
		Where("p.user_id = ?", userId)
	if gameType > 0 {
		query = query.Where("p.game_type = ?", gameType)
	}
	if result > 0 {
		query = query.Where("p.result = ?", result)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []ParticipantMatchListRow
	err := query.Select(`p.match_id, p.opponent_user_id, p.opponent_name, p.opponent_avatar,
		p.game_type, p.match_mode, p.result, p.my_score, p.opponent_score, p.completed_at,
		m.match_time, m.visibility, m.finish_state`).
		Order("p.completed_at DESC, p.match_id DESC").
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error
	return rows, total, err
}

type ParticipantHighScoreRecord struct {
	MatchId        int64
	GameType       int
	OpponentName   string
	CompletedAt    time.Time
	MatchHighScore int
	BestBreak      int
}

func (m *CompetitiveReadModel) ListParticipantH2HPage(
	userId, opponentUserId int64,
	opponentNameKey string,
	gameType, result int,
	start, end *time.Time,
	offset, limit int,
) ([]MatchParticipantResult, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	query := m.db.Model(&MatchParticipantResult{}).Where("user_id = ? AND match_mode = ?", userId, MatchModeRanked)
	if opponentUserId > 0 {
		query = query.Where("opponent_user_id = ?", opponentUserId)
	} else {
		query = query.Where("opponent_user_id = 0 AND opponent_name_key = ?", opponentNameKey)
	}
	if gameType > 0 {
		query = query.Where("game_type = ?", gameType)
	}
	if result > 0 {
		query = query.Where("result = ?", result)
	}
	if start != nil {
		query = query.Where("completed_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("completed_at < ?", *end)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []MatchParticipantResult
	err := query.Order("completed_at DESC, match_id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (m *CompetitiveReadModel) ListRecentParticipantResults(userId int64, gameType, limit int) ([]MatchParticipantResult, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	query := m.db.Where("user_id = ? AND match_mode = ?", userId, MatchModeRanked)
	if gameType > 0 {
		query = query.Where("game_type = ?", gameType)
	}
	var results []MatchParticipantResult
	err := query.Order("completed_at DESC, match_id DESC").Limit(limit).Find(&results).Error
	return results, err
}

func (m *CompetitiveReadModel) ListParticipantHighScores(userId int64, gameType, limit int) ([]ParticipantHighScoreRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	query := m.db.Model(&MatchParticipantResult{}).
		Select("match_id, game_type, opponent_name, completed_at, match_high_score, best_break").
		Where("user_id = ? AND match_mode = ?", userId, MatchModeRanked)
	if gameType > 0 {
		query = query.Where("game_type = ?", gameType)
	}
	var records []ParticipantHighScoreRecord
	err := query.Order("CASE WHEN game_type = 1 THEN best_break ELSE match_high_score END DESC, completed_at DESC, match_id DESC").
		Limit(limit).
		Scan(&records).Error
	return records, err
}

type CompetitiveStatsDelta struct {
	UserId          int64
	GameType        int
	Result          int
	MatchId         int64
	CompletedAt     time.Time
	Score           int
	BestBreak       int
	DurationSeconds int64
}

type OpponentStatsDelta struct {
	UserId          int64
	OpponentUserId  int64
	OpponentNameKey string
	GameType        int
	Result          int
	MatchId         int64
	CompletedAt     time.Time
	OpponentName    string
	OpponentAvatar  string
	MyScore         int
	OpponentScore   int
}

type OpponentStrengthBucketDelta struct {
	UserId     int64
	GameType   int
	RankBucket string
	Result     int
}

func (m *CompetitiveReadModel) ApplyCompetitiveStatsWithTx(tx *gorm.DB, delta CompetitiveStatsDelta) error {
	wins, losses, draws := 0, 0, 0
	switch delta.Result {
	case 1:
		wins = 1
	case 2:
		losses = 1
	default:
		draws = 1
	}
	initialStreak := 0
	if delta.Result == 1 {
		initialStreak = 1
	}
	stats := UserCompetitiveStats{
		UserId:             delta.UserId,
		GameType:           delta.GameType,
		TotalMatches:       1,
		Wins:               wins,
		Losses:             losses,
		Draws:              draws,
		CurrentWinStreak:   initialStreak,
		MaxWinStreak:       initialStreak,
		HighestScore:       delta.Score,
		HighestBreak:       delta.BestBreak,
		DurationCount:      1,
		DurationSumSeconds: delta.DurationSeconds,
		DurationMinSeconds: delta.DurationSeconds,
		DurationMaxSeconds: delta.DurationSeconds,
		LastMatchId:        delta.MatchId,
		LastMatchAt:        &delta.CompletedAt,
		Revision:           1,
	}
	db := m.dbWith(tx)
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "game_type"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"total_matches":        gorm.Expr("total_matches + 1"),
			"wins":                 gorm.Expr("wins + ?", wins),
			"losses":               gorm.Expr("losses + ?", losses),
			"draws":                gorm.Expr("draws + ?", draws),
			"current_win_streak":   gorm.Expr("CASE WHEN ? = 1 THEN current_win_streak + 1 ELSE 0 END", delta.Result),
			"highest_score":        gorm.Expr("CASE WHEN highest_score >= ? THEN highest_score ELSE ? END", delta.Score, delta.Score),
			"highest_break":        gorm.Expr("CASE WHEN highest_break >= ? THEN highest_break ELSE ? END", delta.BestBreak, delta.BestBreak),
			"duration_count":       gorm.Expr("duration_count + 1"),
			"duration_sum_seconds": gorm.Expr("duration_sum_seconds + ?", delta.DurationSeconds),
			"duration_min_seconds": gorm.Expr("CASE WHEN duration_min_seconds = 0 OR duration_min_seconds > ? THEN ? ELSE duration_min_seconds END", delta.DurationSeconds, delta.DurationSeconds),
			"duration_max_seconds": gorm.Expr("CASE WHEN duration_max_seconds >= ? THEN duration_max_seconds ELSE ? END", delta.DurationSeconds, delta.DurationSeconds),
			"last_match_id":        gorm.Expr("CASE WHEN last_match_at IS NULL OR last_match_at <= ? THEN ? ELSE last_match_id END", delta.CompletedAt, delta.MatchId),
			"last_match_at":        gorm.Expr("CASE WHEN last_match_at IS NULL OR last_match_at <= ? THEN ? ELSE last_match_at END", delta.CompletedAt, delta.CompletedAt),
			"revision":             gorm.Expr("revision + 1"),
		}),
	}).Create(&stats).Error; err != nil {
		return err
	}
	return db.Model(&UserCompetitiveStats{}).
		Where("user_id = ? AND game_type = ?", delta.UserId, delta.GameType).
		UpdateColumn("max_win_streak", gorm.Expr("CASE WHEN max_win_streak >= current_win_streak THEN max_win_streak ELSE current_win_streak END")).Error
}

func (m *CompetitiveReadModel) ApplyOpponentStatsWithTx(tx *gorm.DB, delta OpponentStatsDelta) error {
	wins, losses, draws := 0, 0, 0
	switch delta.Result {
	case 1:
		wins = 1
	case 2:
		losses = 1
	default:
		draws = 1
	}
	initialStreak := 0
	if delta.Result == 1 {
		initialStreak = 1
	}
	scoreDiff := delta.MyScore - delta.OpponentScore
	stats := UserOpponentStats{
		UserId:           delta.UserId,
		OpponentUserId:   delta.OpponentUserId,
		OpponentNameKey:  delta.OpponentNameKey,
		OpponentName:     delta.OpponentName,
		OpponentAvatar:   delta.OpponentAvatar,
		GameType:         delta.GameType,
		TotalMatches:     1,
		Wins:             wins,
		Losses:           losses,
		Draws:            draws,
		ScoreDiffSum:     int64(scoreDiff),
		CurrentWinStreak: initialStreak,
		MaxWinStreak:     initialStreak,
		LastMatchId:      delta.MatchId,
		LastMatchAt:      &delta.CompletedAt,
	}
	db := m.dbWith(tx)
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "opponent_user_id"}, {Name: "opponent_name_key"}, {Name: "game_type"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"total_matches":      gorm.Expr("total_matches + 1"),
			"wins":               gorm.Expr("wins + ?", wins),
			"losses":             gorm.Expr("losses + ?", losses),
			"draws":              gorm.Expr("draws + ?", draws),
			"opponent_name":      delta.OpponentName,
			"opponent_avatar":    delta.OpponentAvatar,
			"score_diff_sum":     gorm.Expr("score_diff_sum + ?", scoreDiff),
			"current_win_streak": gorm.Expr("CASE WHEN ? = 1 THEN current_win_streak + 1 ELSE 0 END", delta.Result),
			"last_match_id":      gorm.Expr("CASE WHEN last_match_at IS NULL OR last_match_at <= ? THEN ? ELSE last_match_id END", delta.CompletedAt, delta.MatchId),
			"last_match_at":      gorm.Expr("CASE WHEN last_match_at IS NULL OR last_match_at <= ? THEN ? ELSE last_match_at END", delta.CompletedAt, delta.CompletedAt),
		}),
	}).Create(&stats).Error; err != nil {
		return err
	}
	return db.Model(&UserOpponentStats{}).
		Where("user_id = ? AND opponent_user_id = ? AND opponent_name_key = ? AND game_type = ?", delta.UserId, delta.OpponentUserId, delta.OpponentNameKey, delta.GameType).
		UpdateColumn("max_win_streak", gorm.Expr("CASE WHEN max_win_streak >= current_win_streak THEN max_win_streak ELSE current_win_streak END")).Error
}

type OpponentStatsListRow struct {
	OpponentUserId  int64
	OpponentNameKey string
	OpponentName    string
	OpponentAvatar  string
	RankName        string
	TotalMatches    int
	Wins            int
	Losses          int
	LastMatchAt     *time.Time
	LastMatchId     int64
}

func (m *CompetitiveReadModel) GetOpponentSummary(userId int64) (int64, int, int, error) {
	var row struct {
		TotalOpponents int64
		TotalMatches   int
		TotalWins      int
	}
	err := m.db.Model(&UserOpponentStats{}).
		Select("COUNT(*) AS total_opponents, COALESCE(SUM(total_matches), 0) AS total_matches, COALESCE(SUM(wins), 0) AS total_wins").
		Where("user_id = ? AND game_type = 0", userId).
		Scan(&row).Error
	return row.TotalOpponents, row.TotalMatches, row.TotalWins, err
}

func (m *CompetitiveReadModel) ListOpponentStatsPage(userId int64, keyword string, offset, limit int) ([]OpponentStatsListRow, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	query := m.db.Table("user_opponent_stats AS s").
		Joins("LEFT JOIN users AS u ON u.id = s.opponent_user_id AND s.opponent_user_id > 0").
		Where("s.user_id = ? AND s.game_type = 0", userId)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("s.opponent_name LIKE ? OR u.nickname LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []OpponentStatsListRow
	err := query.Select(`s.opponent_user_id, s.opponent_name_key,
		COALESCE(NULLIF(u.nickname, ''), s.opponent_name) AS opponent_name,
		COALESCE(NULLIF(u.avatar, ''), s.opponent_avatar) AS opponent_avatar,
		s.total_matches, s.wins, s.losses, s.last_match_at, s.last_match_id`).
		Order("s.last_match_at DESC, s.last_match_id DESC").
		Offset(offset).
		Limit(limit).
		Scan(&rows).Error
	return rows, total, err
}

func (m *CompetitiveReadModel) ListRecentOpponentStats(userId int64, gameType, limit int) ([]OpponentStatsListRow, error) {
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	if gameType < 0 {
		gameType = 0
	}
	var rows []OpponentStatsListRow
	err := m.db.Table("user_opponent_stats AS s").
		Select(`s.opponent_user_id, s.opponent_name_key,
			COALESCE(NULLIF(u.nickname, ''), s.opponent_name) AS opponent_name,
			COALESCE(NULLIF(u.avatar, ''), s.opponent_avatar) AS opponent_avatar,
			COALESCE(c.name, '') AS rank_name,
			s.total_matches, s.wins, s.losses, s.last_match_at, s.last_match_id`).
		Joins("LEFT JOIN users AS u ON u.id = s.opponent_user_id AND s.opponent_user_id > 0").
		Joins("LEFT JOIN user_ranking AS r ON r.user_id = s.opponent_user_id AND r.game_type = ?", defaultRankingGameType).
		Joins("LEFT JOIN rank_config AS c ON c.level = COALESCE(r.rank_level, 1)").
		Where("s.user_id = ? AND s.game_type = ?", userId, gameType).
		Order("s.last_match_at DESC, s.last_match_id DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (m *CompetitiveReadModel) ListOpponentStrengthBuckets(userId int64, gameType int) ([]UserOpponentStrengthBucket, error) {
	var buckets []UserOpponentStrengthBucket
	err := m.db.Where("user_id = ? AND game_type = ?", userId, gameType).
		Order("rank_bucket ASC").
		Find(&buckets).Error
	return buckets, err
}

func (m *CompetitiveReadModel) IncrementOpponentStrengthBucketWithTx(tx *gorm.DB, delta OpponentStrengthBucketDelta) error {
	wins := 0
	if delta.Result == 1 {
		wins = 1
	}
	bucket := UserOpponentStrengthBucket{
		UserId:     delta.UserId,
		GameType:   delta.GameType,
		RankBucket: delta.RankBucket,
		Matches:    1,
		Wins:       wins,
	}
	return m.dbWith(tx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "game_type"}, {Name: "rank_bucket"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"matches": gorm.Expr("matches + 1"),
			"wins":    gorm.Expr("wins + ?", wins),
		}),
	}).Create(&bucket).Error
}

func (m *CompetitiveReadModel) FindStats(userId int64, gameType int) (*UserCompetitiveStats, error) {
	return m.FindStatsWithTx(nil, userId, gameType)
}

func (m *CompetitiveReadModel) FindStatsWithTx(tx *gorm.DB, userId int64, gameType int) (*UserCompetitiveStats, error) {
	var stats UserCompetitiveStats
	err := m.dbWith(tx).Where("user_id = ? AND game_type = ?", userId, gameType).First(&stats).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &stats, err
}

func (m *CompetitiveReadModel) ListStats(userId int64) ([]UserCompetitiveStats, error) {
	return m.ListStatsWithTx(nil, userId)
}

func (m *CompetitiveReadModel) ListStatsWithTx(tx *gorm.DB, userId int64) ([]UserCompetitiveStats, error) {
	var stats []UserCompetitiveStats
	err := m.dbWith(tx).Where("user_id = ?", userId).Order("game_type ASC").Find(&stats).Error
	return stats, err
}

func (m *CompetitiveReadModel) FindOpponentStats(userId, opponentUserId int64, opponentNameKey string, gameType int) (*UserOpponentStats, error) {
	return m.FindOpponentStatsWithTx(nil, userId, opponentUserId, opponentNameKey, gameType)
}

func (m *CompetitiveReadModel) FindOpponentStatsWithTx(tx *gorm.DB, userId, opponentUserId int64, opponentNameKey string, gameType int) (*UserOpponentStats, error) {
	var stats UserOpponentStats
	err := m.dbWith(tx).Where("user_id = ? AND opponent_user_id = ? AND opponent_name_key = ? AND game_type = ?", userId, opponentUserId, opponentNameKey, gameType).First(&stats).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &stats, err
}

func (m *CompetitiveReadModel) FindCheckpoint(jobName string) (*CompetitiveReadModelRebuildCheckpoint, error) {
	return m.FindCheckpointWithTx(nil, jobName)
}

func (m *CompetitiveReadModel) FindCheckpointWithTx(tx *gorm.DB, jobName string) (*CompetitiveReadModelRebuildCheckpoint, error) {
	var checkpoint CompetitiveReadModelRebuildCheckpoint
	err := m.dbWith(tx).Where("job_name = ?", jobName).First(&checkpoint).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &checkpoint, err
}

func (m *CompetitiveReadModel) UpsertCheckpoint(checkpoint *CompetitiveReadModelRebuildCheckpoint) error {
	return m.UpsertCheckpointWithTx(nil, checkpoint)
}

func (m *CompetitiveReadModel) UpsertCheckpointWithTx(tx *gorm.DB, checkpoint *CompetitiveReadModelRebuildCheckpoint) error {
	if checkpoint == nil {
		return nil
	}
	return m.dbWith(tx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "job_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"cursor_match_id", "cursor_completed_at", "backfill_cursor_match_id", "backfill_completed", "paused", "last_error", "updated_at",
		}),
	}).Create(checkpoint).Error
}
