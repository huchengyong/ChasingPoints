package model

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrMatchRevisionConflict = errors.New("match revision conflict")

const (
	MatchModePractice = "practice"
	MatchModeRanked   = "ranked"

	MatchVisibilityPrivate = "private"
	MatchVisibilityPublic  = "public"

	FinishStateNone                = "none"
	FinishStatePendingConfirmation = "pending_confirmation"
	FinishRequestTTL               = 24 * time.Hour

	CompletionSourceReferee         = "referee"
	CompletionSourcePlayerDirect    = "player_direct"
	CompletionSourcePlayerConfirmed = "player_confirmed"
	CompletionSourceUnknown         = "unknown"
)

func NormalizeMatchMode(mode string) string {
	if mode == MatchModePractice {
		return MatchModePractice
	}
	return MatchModeRanked
}

func NormalizeMatchVisibility(visibility, mode string) string {
	if visibility == MatchVisibilityPublic {
		return MatchVisibilityPublic
	}
	if visibility == MatchVisibilityPrivate && NormalizeMatchMode(mode) == MatchModePractice {
		return MatchVisibilityPrivate
	}
	return MatchVisibilityPublic
}

func NormalizeFinishState(state string) string {
	if state == FinishStatePendingConfirmation {
		return FinishStatePendingConfirmation
	}
	return FinishStateNone
}

// Match 对局记录
type Match struct {
	Id                         int64          `gorm:"primarykey" json:"id"`
	UserId                     int64          `gorm:"not null;index" json:"user_id"`
	OpponentId                 *int64         `gorm:"index" json:"opponent_id"`
	OpponentName               string         `gorm:"size:50;not null" json:"opponent_name"`
	GameType                   int            `gorm:"not null" json:"game_type"` // 1=斯诺克 2=九球追分 3=中式八球 4=美式九球
	GameMode                   string         `gorm:"size:20" json:"game_mode"`  // 比赛模式
	MatchMode                  string         `gorm:"size:20;not null;index" json:"match_mode"`
	Visibility                 string         `gorm:"size:20;not null;index" json:"visibility"`
	FinishConfirmationRequired bool           `gorm:"not null;default:false" json:"finish_confirmation_required"`
	FinishState                string         `gorm:"size:32;not null;index" json:"finish_state"`
	FinishRequestedBy          *int64         `gorm:"index" json:"finish_requested_by"`
	FinishRequestedAt          *time.Time     `json:"finish_requested_at"`
	FinishRequestRevision      int64          `gorm:"not null;default:0" json:"finish_request_revision"`
	RefereeUserId              *int64         `gorm:"index" json:"referee_user_id"`
	RefereeJoinedAt            *time.Time     `json:"referee_joined_at"`
	CompletedByUserId          *int64         `gorm:"index" json:"completed_by_user_id"`
	CompletionSource           string         `gorm:"size:20;not null;default:unknown" json:"completion_source"` // referee/player_direct/player_confirmed/unknown
	MyScore                    int            `gorm:"not null;default:0" json:"my_score"`
	OpponentScore              int            `gorm:"not null;default:0" json:"opponent_score"`
	CurrentFrameMyScore        int            `gorm:"not null;default:0" json:"current_frame_my_score"`
	CurrentFrameOpponentScore  int            `gorm:"not null;default:0" json:"current_frame_opponent_score"`
	CurrentFrameStarted        bool           `gorm:"not null;default:true" json:"current_frame_started"`
	SyncRevision               int64          `gorm:"not null;default:0" json:"sync_revision"`
	Status                     int            `gorm:"not null;default:1" json:"status"` // 1=进行中 2=已完成 3=已取消
	Result                     *int           `json:"result"`                           // 1=胜利 2=失败 3=平局
	MatchTime                  time.Time      `gorm:"not null" json:"match_time"`
	EndTime                    *time.Time     `json:"end_time"`
	Remark                     string         `gorm:"size:500" json:"remark"`
	CreatedAt                  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt                  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt                  gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Match) TableName() string {
	return "matches"
}

// MatchRound 局记录
type MatchRound struct {
	Id            int64     `gorm:"primarykey" json:"id"`
	MatchId       int64     `gorm:"not null;index" json:"match_id"`
	RoundNo       int       `gorm:"not null" json:"round_no"`
	MyScore       int       `gorm:"not null;default:0" json:"my_score"`
	OpponentScore int       `gorm:"not null;default:0" json:"opponent_score"`
	Winner        *int      `json:"winner"` // 1=我 2=对手
	WinType       string    `gorm:"size:20" json:"win_type"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (MatchRound) TableName() string {
	return "match_rounds"
}

// MatchAction 操作日志
type MatchAction struct {
	Id             int64     `gorm:"primarykey" json:"id"`
	MatchId        int64     `gorm:"not null;index;uniqueIndex:idx_match_client_action,priority:1" json:"match_id"`
	RoundNo        int       `gorm:"not null" json:"round_no"`
	ActionType     string    `gorm:"size:20;not null" json:"action_type"` // score/foul/win
	Actor          int       `gorm:"not null" json:"actor"`               // 1=我 2=对手
	ScoreChange    int       `gorm:"not null;default:0" json:"score_change"`
	ClientActionId *string   `gorm:"size:64;uniqueIndex:idx_match_client_action,priority:2" json:"client_action_id"`
	BaseRevision   int64     `gorm:"not null;default:0" json:"base_revision"`
	ServerRevision int64     `gorm:"not null;default:0" json:"server_revision"`
	ExtraData      *string   `gorm:"type:json" json:"extra_data"`
	IsUndone       int       `gorm:"not null;default:0;index" json:"is_undone"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (MatchAction) TableName() string {
	return "match_actions"
}

// MatchAchievement 特殊成绩
type MatchAchievement struct {
	Id              int64     `gorm:"primarykey" json:"id"`
	MatchId         int64     `gorm:"not null;index;uniqueIndex:uk_match_actor_achievement,priority:1" json:"match_id"`
	Actor           int       `gorm:"not null;default:1;uniqueIndex:uk_match_actor_achievement,priority:2" json:"actor"`
	AchievementType string    `gorm:"size:20;not null;uniqueIndex:uk_match_actor_achievement,priority:3" json:"achievement_type"`
	Count           int       `gorm:"not null;default:0" json:"count"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (MatchAchievement) TableName() string {
	return "match_achievements"
}

// Opponent 对手
type Opponent struct {
	Id           int64          `gorm:"primarykey" json:"id"`
	UserId       int64          `gorm:"not null;index" json:"user_id"`
	Name         string         `gorm:"size:50;not null" json:"name"`
	Avatar       string         `gorm:"size:255" json:"avatar"`
	LinkedUserId *int64         `gorm:"index" json:"linked_user_id"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Opponent) TableName() string {
	return "opponents"
}

// MatchModel 对局相关数据库操作
type MatchModel struct {
	db *gorm.DB
}

func NewMatchModel(db *gorm.DB) *MatchModel {
	return &MatchModel{db: db}
}

// ========== Match 操作 ==========

// Create 创建对局
func (m *MatchModel) Create(match *Match) error {
	return m.db.Create(match).Error
}

func (m *MatchModel) CreateWithTx(tx *gorm.DB, match *Match) error {
	if tx != nil {
		return tx.Create(match).Error
	}
	return m.Create(match)
}

// FindById 根据ID查找对局
func (m *MatchModel) FindById(id int64) (*Match, error) {
	var match Match
	err := m.db.First(&match, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &match, err
}

func (m *MatchModel) FindByIdForUpdateWithTx(tx *gorm.DB, id int64) (*Match, error) {
	var match Match
	db := m.db
	if tx != nil {
		db = tx
	}
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&match, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &match, err
}

func (m *MatchModel) ExpireStaleFinishRequest(matchId int64) (*Match, bool, int64, error) {
	var refreshed *Match
	var expiredRevision int64
	err := m.db.Transaction(func(tx *gorm.DB) error {
		locked, err := m.FindByIdForUpdateWithTx(tx, matchId)
		if err != nil {
			return err
		}
		if locked == nil {
			return gorm.ErrRecordNotFound
		}
		if locked.FinishState != FinishStatePendingConfirmation || locked.FinishRequestedAt == nil || time.Since(*locked.FinishRequestedAt) < FinishRequestTTL {
			refreshed = locked
			return nil
		}

		requestRevision := locked.FinishRequestRevision
		locked.FinishState = FinishStateNone
		locked.FinishRequestedBy = nil
		locked.FinishRequestedAt = nil
		locked.FinishRequestRevision = 0
		revision, err := m.BumpMatchRevisionWithTx(tx, locked)
		if err != nil {
			return err
		}
		expiredRevision = revision
		if err := m.CreateActionWithRevisionWithTx(tx, &MatchAction{
			MatchId:      locked.Id,
			RoundNo:      0,
			ActionType:   "finish_expired",
			Actor:        0,
			BaseRevision: requestRevision,
		}, revision); err != nil {
			return err
		}
		refreshed = locked
		return nil
	})
	return refreshed, expiredRevision > 0, expiredRevision, err
}

// FindCurrentByUserId 查找用户进行中的对局
func (m *MatchModel) FindCurrentByUserId(userId int64) (*Match, error) {
	return m.FindCurrentByUserIdWithTx(nil, userId)
}

func (m *MatchModel) FindCurrentByUserIdWithTx(tx *gorm.DB, userId int64) (*Match, error) {
	db := m.db
	if tx != nil {
		db = tx
	}

	var match Match
	query := db.Where("(user_id = ? OR opponent_id = ? OR referee_user_id = ?) AND status = 1", userId, userId, userId)
	err := query.
		Order("match_time DESC").
		First(&match).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &match, err
}

// ListCompletedForRankingReplay 获取用于段位历史回放的已完成对局
func (m *MatchModel) ListCompletedForRankingReplay() ([]Match, error) {
	var matches []Match
	err := m.db.
		Where("status = ? AND deleted_at IS NULL", 2).
		Where("match_mode = ? OR match_mode = '' OR match_mode IS NULL", MatchModeRanked).
		Order("CASE WHEN end_time IS NULL THEN 1 ELSE 0 END ASC").
		Order("end_time ASC").
		Order("match_time ASC").
		Order("id ASC").
		Find(&matches).Error
	return matches, err
}

func (m *MatchModel) CountCompletedMatchesBetweenUsersByGameTypeBetween(
	tx *gorm.DB,
	userA, userB int64,
	gameType int,
	start, end time.Time,
	excludeMatchId int64,
) (int64, error) {
	db := m.db
	if tx != nil {
		db = tx
	}

	query := db.Model(&Match{}).
		Where("status = ? AND game_type = ?", 2, gameType).
		Where("match_mode = ? OR match_mode = '' OR match_mode IS NULL", MatchModeRanked).
		Where("((user_id = ? AND opponent_id = ?) OR (user_id = ? AND opponent_id = ?))", userA, userB, userB, userA).
		Where("((end_time IS NOT NULL AND end_time >= ? AND end_time < ?) OR (end_time IS NULL AND match_time >= ? AND match_time < ?))",
			start, end, start, end)
	if excludeMatchId > 0 {
		query = query.Where("id <> ?", excludeMatchId)
	}

	var count int64
	err := query.Count(&count).Error
	return count, err
}

// Update 更新对局
func (m *MatchModel) Update(match *Match) error {
	return m.db.Save(match).Error
}

func (m *MatchModel) UpdateWithTx(tx *gorm.DB, match *Match) error {
	if tx != nil {
		return tx.Save(match).Error
	}
	return m.Update(match)
}

// MatchWithOpponentAvatar 带对手头像的对局记录
type MatchWithOpponentAvatar struct {
	Match
	OpponentAvatar string `json:"opponent_avatar"`
	IsAsOpponent   bool   `json:"-"` // 标记用户是否作为对手参与（用于调整胜负和分数）
}

// ListByUserId 获取用户对局列表（支持双向查询：用户可能是 user_id 或 opponent_id）
func (m *MatchModel) ListByUserId(
	userId int64,
	gameType, result int,
	offset, limit int,
) ([]MatchWithOpponentAvatar, int64, error) {
	// 基础条件：用户作为发起方(user_id)或对手方(opponent_id)
	baseWhere := "(matches.user_id = ? OR matches.opponent_id = ?) AND matches.status = 2 AND matches.deleted_at IS NULL"
	args := []interface{}{userId, userId}

	if gameType > 0 {
		baseWhere += " AND matches.game_type = ?"
		args = append(args, gameType)
	}
	// result 筛选需要特殊处理，因为作为对手时胜负是反的
	// 先不在SQL中筛选，后面在代码中处理

	// 查询总数（暂时不筛选result，后面代码中处理）
	var totalList []struct {
		UserId int64
		Result *int
	}
	totalWhere := "(matches.user_id = ? OR matches.opponent_id = ?) AND matches.status = 2 AND matches.deleted_at IS NULL"
	totalArgs := []interface{}{userId, userId}
	if gameType > 0 {
		totalWhere += " AND matches.game_type = ?"
		totalArgs = append(totalArgs, gameType)
	}
	if err := m.db.Table("matches").Select("user_id, result").Where(totalWhere, totalArgs...).Scan(&totalList).Error; err != nil {
		return nil, 0, err
	}

	// 计算真正的总数（考虑result筛选）
	var total int64
	for _, item := range totalList {
		if result == 0 {
			total++
			continue
		}
		isAsCreator := item.UserId == userId
		if item.Result == nil {
			continue
		}
		actualResult := *item.Result
		if !isAsCreator {
			// 作为对手时，胜负反转
			if actualResult == 1 {
				actualResult = 2
			} else if actualResult == 2 {
				actualResult = 1
			}
		}
		if actualResult == result {
			total++
		}
	}

	// 查询列表
	var rawList []struct {
		Match
		OpponentAvatar string `json:"opponent_avatar"`
		CreatorName    string `json:"creator_name"`
		CreatorAvatar  string `json:"creator_avatar"`
	}
	err := m.db.Table("matches").
		Select(`matches.*,
			COALESCE(u.avatar, o.avatar, '') as opponent_avatar,
			COALESCE(u_creator.nickname, '') as creator_name,
			COALESCE(u_creator.avatar, '') as creator_avatar`).
		Joins("LEFT JOIN users u ON matches.opponent_id = u.id").
		Joins("LEFT JOIN opponents o ON o.user_id = matches.user_id AND o.name = matches.opponent_name").
		Joins("LEFT JOIN users u_creator ON matches.user_id = u_creator.id").
		Where(baseWhere, args...).
		Order("matches.match_time DESC").
		Scan(&rawList).Error
	if err != nil {
		return nil, 0, err
	}

	// 处理数据：根据用户角色调整胜负和分数
	var list []MatchWithOpponentAvatar
	for _, item := range rawList {
		isAsCreator := item.UserId == userId
		record := MatchWithOpponentAvatar{
			Match:        item.Match,
			IsAsOpponent: !isAsCreator,
		}

		if !isAsCreator {
			// 用户是作为对手参与的，需要调整数据
			// 对手名和头像应该是发起方的信息
			record.OpponentName = item.CreatorName
			record.OpponentAvatar = item.CreatorAvatar
			// 分数互换
			record.MyScore = item.OpponentScore
			record.OpponentScore = item.MyScore
			// 胜负反转
			if item.Result != nil {
				newResult := *item.Result
				if newResult == 1 {
					newResult = 2
				} else if newResult == 2 {
					newResult = 1
				}
				record.Result = &newResult
			}
		} else {
			record.OpponentAvatar = item.OpponentAvatar
		}

		// result 筛选
		if result > 0 {
			if record.Result == nil || *record.Result != result {
				continue
			}
		}

		list = append(list, record)
	}

	// 分页处理
	if offset >= len(list) {
		return []MatchWithOpponentAvatar{}, total, nil
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}

	return list[offset:end], total, nil
}

func (m *MatchModel) ListCompletedWithPerspectiveByUserId(userId int64, gameType int) ([]MatchWithOpponentAvatar, error) {
	baseWhere := "(matches.user_id = ? OR matches.opponent_id = ?) AND matches.status = 2 AND matches.deleted_at IS NULL"
	args := []interface{}{userId, userId}
	if gameType > 0 {
		baseWhere += " AND matches.game_type = ?"
		args = append(args, gameType)
	}

	var rawList []struct {
		Match
		OpponentAvatar string `json:"opponent_avatar"`
		CreatorName    string `json:"creator_name"`
		CreatorAvatar  string `json:"creator_avatar"`
	}
	err := m.db.Table("matches").
		Select(`matches.*,
			COALESCE(u.avatar, o.avatar, '') as opponent_avatar,
			COALESCE(u_creator.nickname, '') as creator_name,
			COALESCE(u_creator.avatar, '') as creator_avatar`).
		Joins("LEFT JOIN users u ON matches.opponent_id = u.id").
		Joins("LEFT JOIN opponents o ON o.user_id = matches.user_id AND o.name = matches.opponent_name").
		Joins("LEFT JOIN users u_creator ON matches.user_id = u_creator.id").
		Where(baseWhere, args...).
		Order("matches.match_time DESC").
		Scan(&rawList).Error
	if err != nil {
		return nil, err
	}

	list := make([]MatchWithOpponentAvatar, 0, len(rawList))
	for _, item := range rawList {
		isAsCreator := item.UserId == userId
		record := MatchWithOpponentAvatar{
			Match:        item.Match,
			IsAsOpponent: !isAsCreator,
		}

		if !isAsCreator {
			record.OpponentName = item.CreatorName
			record.OpponentAvatar = item.CreatorAvatar
			record.MyScore = item.OpponentScore
			record.OpponentScore = item.MyScore
			if item.Result != nil {
				newResult := *item.Result
				if newResult == 1 {
					newResult = 2
				} else if newResult == 2 {
					newResult = 1
				}
				record.Result = &newResult
			}
		} else {
			record.OpponentAvatar = item.OpponentAvatar
		}

		list = append(list, record)
	}

	return list, nil
}

// H2HMatchRecord 交锋记录（已处理双向数据）
type H2HMatchRecord struct {
	Id            int64
	GameType      int
	OpponentName  string
	MyScore       int
	OpponentScore int
	Result        int
	MatchTime     time.Time
}

// ListByOpponentId 获取与特定对手用户的交锋列表（支持双向查询）
// opponentUserId 是对手的用户ID（users表中的ID）
// 优化版本：使用 JOIN 查询替代循环内的单独查询
func (m *MatchModel) ListByOpponentId(
	userId int64,
	opponentUserId int64,
	result int,
	offset, limit int,
	startTime, endTime *time.Time,
) ([]H2HMatchRecord, int64, error) {
	// 查询条件：
	// 1. 用户作为发起方(user_id)，对手是 opponent_id
	// 2. 用户作为对手(opponent_id)，发起方是 user_id
	// 同时JOIN用户表获取发起方昵称
	var rawList []struct {
		Match
		CreatorNickname string
	}
	query := m.db.Table("matches").
		Select("matches.*, COALESCE(u.nickname, '') as creator_nickname").
		Joins("LEFT JOIN users u ON matches.user_id = u.id").
		Where(`(
			(matches.user_id = ? AND matches.opponent_id = ?) OR
			(matches.user_id = ? AND matches.opponent_id = ?)
		) AND matches.status = 2 AND matches.deleted_at IS NULL`,
			userId, opponentUserId, opponentUserId, userId)
	if startTime != nil {
		query = query.Where("matches.match_time >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("matches.match_time < ?", *endTime)
	}

	err := query.Order("matches.match_time DESC").Scan(&rawList).Error
	if err != nil {
		return nil, 0, err
	}

	// 处理数据：根据用户角色调整胜负和分数
	var records []H2HMatchRecord
	for _, item := range rawList {
		isAsCreator := item.UserId == userId
		record := H2HMatchRecord{
			Id:        item.Id,
			GameType:  item.GameType,
			MatchTime: item.MatchTime,
		}

		if isAsCreator {
			// 用户是发起方
			record.OpponentName = item.OpponentName
			record.MyScore = item.MyScore
			record.OpponentScore = item.OpponentScore
			if item.Result != nil {
				record.Result = *item.Result
			}
		} else {
			// 用户是对手方，使用已经 JOIN 的发起方信息
			if item.CreatorNickname != "" {
				record.OpponentName = item.CreatorNickname
			} else {
				record.OpponentName = "玩家"
			}
			// 分数互换
			record.MyScore = item.OpponentScore
			record.OpponentScore = item.MyScore
			// 胜负反转
			if item.Result != nil {
				if *item.Result == 1 {
					record.Result = 2
				} else if *item.Result == 2 {
					record.Result = 1
				} else {
					record.Result = *item.Result
				}
			}
		}

		// 结果筛选
		if result > 0 && record.Result != result {
			continue
		}

		records = append(records, record)
	}

	total := int64(len(records))

	// 分页处理
	if offset >= len(records) {
		return []H2HMatchRecord{}, total, nil
	}
	end := offset + limit
	if end > len(records) {
		end = len(records)
	}

	return records[offset:end], total, nil
}

// ListByOpponentName 获取与匿名对手的交锋列表。
// 仅在没有稳定对手用户 ID 时作为兜底方案使用。
func (m *MatchModel) ListByOpponentName(
	userId int64,
	opponentName string,
	result int,
	offset, limit int,
	startTime, endTime *time.Time,
) ([]H2HMatchRecord, int64, error) {
	if opponentName == "" {
		return []H2HMatchRecord{}, 0, nil
	}

	var rawList []struct {
		Id            int64
		GameType      int
		OpponentName  string
		MyScore       int
		OpponentScore int
		Result        *int
		MatchTime     time.Time
	}
	query := m.db.Model(&Match{}).
		Select("id, game_type, opponent_name, my_score, opponent_score, result, match_time").
		Where("user_id = ? AND opponent_name = ? AND status = 2 AND deleted_at IS NULL", userId, opponentName)
	if startTime != nil {
		query = query.Where("match_time >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("match_time < ?", *endTime)
	}

	err := query.Order("match_time DESC").Find(&rawList).Error
	if err != nil {
		return nil, 0, err
	}

	records := make([]H2HMatchRecord, 0, len(rawList))
	for _, item := range rawList {
		record := H2HMatchRecord{
			Id:            item.Id,
			GameType:      item.GameType,
			OpponentName:  item.OpponentName,
			MyScore:       item.MyScore,
			OpponentScore: item.OpponentScore,
			MatchTime:     item.MatchTime,
		}
		if item.Result != nil {
			record.Result = *item.Result
		}
		if result > 0 && record.Result != result {
			continue
		}
		records = append(records, record)
	}

	total := int64(len(records))
	if offset >= len(records) {
		return []H2HMatchRecord{}, total, nil
	}

	end := offset + limit
	if end > len(records) {
		end = len(records)
	}

	return records[offset:end], total, nil
}

func (m *MatchModel) GetH2HStats(userId int64, opponentName string) (total, myWins, oppWins int, avgDiff float64, err error) {
	total, myWins, oppWins, avgDiff, _, err = m.GetH2HStatsByOpponent(userId, 0, opponentName)
	return total, myWins, oppWins, avgDiff, err
}

// ListByRefereeUserId 获取裁判执裁历史（已完成和已取消）
func (m *MatchModel) ListByRefereeUserId(refereeUserId int64, offset, limit int) ([]Match, int64, error) {
	var total int64
	err := m.db.Model(&Match{}).
		Where("referee_user_id = ? AND status IN (2, 3) AND deleted_at IS NULL", refereeUserId).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var matches []Match
	err = m.db.Where("referee_user_id = ? AND status IN (2, 3) AND deleted_at IS NULL", refereeUserId).
		Order("COALESCE(end_time, updated_at) DESC").
		Offset(offset).
		Limit(limit).
		Find(&matches).Error
	if err != nil {
		return nil, 0, err
	}

	return matches, total, nil
}

// GetH2HStatsByOpponent 获取交锋统计。
// 当对手是注册用户时优先按用户 ID 双向统计，保证与交锋历史列表口径一致。
// 当对手没有用户 ID 时，再退回到基于历史名称的统计。
func (m *MatchModel) GetH2HStatsByOpponent(userId int64, opponentUserId int64, opponentName string) (total, myWins, oppWins int, avgDiff float64, maxWinStreak int, err error) {
	type h2hStatRow struct {
		UserId        int64
		MyScore       int
		OpponentScore int
		Result        *int
		MatchTime     time.Time
	}

	calculateStats := func(rows []h2hStatRow) (int, int, int, float64, int) {
		if len(rows) == 0 {
			return 0, 0, 0, 0, 0
		}

		sort.Slice(rows, func(i, j int) bool {
			if rows[i].MatchTime.Equal(rows[j].MatchTime) {
				return rows[i].UserId < rows[j].UserId
			}
			return rows[i].MatchTime.Before(rows[j].MatchTime)
		})

		scoreDiffTotal := 0
		currentWinStreak := 0
		totalMatches := 0
		totalMyWins := 0
		totalOppWins := 0
		longestWinStreak := 0

		for _, match := range rows {
			totalMatches++

			actualResult := 0
			if match.Result != nil {
				actualResult = *match.Result
			}

			if match.UserId == userId {
				scoreDiffTotal += match.MyScore - match.OpponentScore
			} else {
				scoreDiffTotal += match.OpponentScore - match.MyScore
				if actualResult == 1 {
					actualResult = 2
				} else if actualResult == 2 {
					actualResult = 1
				}
			}

			if actualResult == 1 {
				totalMyWins++
				currentWinStreak++
				if currentWinStreak > longestWinStreak {
					longestWinStreak = currentWinStreak
				}
			} else if actualResult == 2 {
				totalOppWins++
				currentWinStreak = 0
			} else {
				currentWinStreak = 0
			}
		}

		return totalMatches, totalMyWins, totalOppWins, float64(scoreDiffTotal) / float64(totalMatches), longestWinStreak
	}

	if opponentUserId > 0 {
		var matches []h2hStatRow
		err = m.db.Model(&Match{}).
			Select("user_id, my_score, opponent_score, result, match_time").
			Where(`(
				(user_id = ? AND opponent_id = ?) OR
				(user_id = ? AND opponent_id = ?)
			) AND status = 2 AND deleted_at IS NULL`, userId, opponentUserId, opponentUserId, userId).
			Find(&matches).Error
		if err != nil {
			return 0, 0, 0, 0, 0, err
		}

		total, myWins, oppWins, avgDiff, maxWinStreak = calculateStats(matches)
		return total, myWins, oppWins, avgDiff, maxWinStreak, nil
	}

	if opponentName == "" {
		return 0, 0, 0, 0, 0, nil
	}

	var matches []h2hStatRow
	err = m.db.Model(&Match{}).
		Select("user_id, my_score, opponent_score, result, match_time").
		Where("user_id = ? AND opponent_name = ? AND status = 2 AND deleted_at IS NULL", userId, opponentName).
		Find(&matches).Error
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}

	total, myWins, oppWins, avgDiff, maxWinStreak = calculateStats(matches)
	return total, myWins, oppWins, avgDiff, maxWinStreak, nil
}

// ========== MatchRound 操作 ==========

// CreateRound 创建局记录
func (m *MatchModel) CreateRound(round *MatchRound) error {
	return m.CreateRoundWithTx(nil, round)
}

func (m *MatchModel) CreateRoundWithTx(tx *gorm.DB, round *MatchRound) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Create(round).Error
}

// GetRoundCount 获取对局的局数
func (m *MatchModel) GetRoundCount(matchId int64) (int64, error) {
	var count int64
	err := m.db.Model(&MatchRound{}).
		Where("match_id = ? AND winner IS NOT NULL AND win_type <> ?", matchId, "start").
		Count(&count).Error
	return count, err
}

func (m *MatchModel) GetLastRoundWithTx(tx *gorm.DB, matchId int64) (*MatchRound, error) {
	var round MatchRound
	db := m.db
	if tx != nil {
		db = tx
	}
	err := db.Where("match_id = ? AND winner IS NOT NULL AND win_type <> ?", matchId, "start").
		Order("id DESC").
		First(&round).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &round, err
}

func (m *MatchModel) DeleteRoundWithTx(tx *gorm.DB, roundId int64) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Delete(&MatchRound{}, roundId).Error
}

// ListCompletedRounds 获取已完成局记录
func (m *MatchModel) ListCompletedRounds(matchId int64) ([]MatchRound, error) {
	return m.ListCompletedRoundsWithTx(nil, matchId)
}

func (m *MatchModel) ListCompletedRoundsWithTx(tx *gorm.DB, matchId int64) ([]MatchRound, error) {
	var rounds []MatchRound
	db := m.db
	if tx != nil {
		db = tx
	}
	err := db.
		Where("match_id = ? AND winner IS NOT NULL AND win_type <> ?", matchId, "start").
		Order("round_no ASC").
		Find(&rounds).Error
	return rounds, err
}

// ========== MatchAction 操作 ==========

// CreateAction 创建操作日志
func (m *MatchModel) CreateAction(action *MatchAction) error {
	return m.CreateActionWithTx(nil, action)
}

func (m *MatchModel) CreateActionWithTx(tx *gorm.DB, action *MatchAction) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Create(action).Error
}

func (m *MatchModel) CreateActionWithRevisionWithTx(tx *gorm.DB, action *MatchAction, serverRevision int64) error {
	action.ServerRevision = serverRevision
	return m.CreateActionWithTx(tx, action)
}

func (m *MatchModel) FindActionByClientActionID(matchId int64, clientActionID string) (*MatchAction, error) {
	return m.FindActionByClientActionIDWithTx(nil, matchId, clientActionID)
}

func (m *MatchModel) FindActionByClientActionIDWithTx(tx *gorm.DB, matchId int64, clientActionID string) (*MatchAction, error) {
	var action MatchAction
	db := m.db
	if tx != nil {
		db = tx
	}
	result := db.Where("match_id = ? AND client_action_id = ?", matchId, clientActionID).Limit(1).Find(&action)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &action, nil
}

func (m *MatchModel) BumpMatchRevisionWithTx(tx *gorm.DB, match *Match) (int64, error) {
	if match == nil {
		return 0, nil
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	previousRevision := match.SyncRevision
	match.SyncRevision += 1

	result := db.Model(&Match{}).
		Where("id = ? AND sync_revision = ?", match.Id, previousRevision).
		Select("*").
		Omit("created_at").
		Updates(match)
	if result.Error != nil {
		match.SyncRevision = previousRevision
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		match.SyncRevision = previousRevision
		return 0, ErrMatchRevisionConflict
	}

	return match.SyncRevision, nil
}

// GetLastAction 获取最后一条操作记录
func (m *MatchModel) GetLastAction(matchId int64) (*MatchAction, error) {
	return m.GetLastActionWithTx(nil, matchId)
}

func (m *MatchModel) GetLastActionWithTx(tx *gorm.DB, matchId int64) (*MatchAction, error) {
	var action MatchAction
	db := m.db
	if tx != nil {
		db = tx
	}
	err := db.Where("match_id = ? AND is_undone = 0", matchId).Order("id DESC").First(&action).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &action, err
}

// CountScoreActions 统计指定局内未撤销的加分操作次数
func (m *MatchModel) CountScoreActions(matchId int64, roundNo int, scoreChange int) (int64, error) {
	var count int64
	err := m.db.Model(&MatchAction{}).
		Where("match_id = ? AND round_no = ? AND action_type = ? AND score_change = ? AND is_undone = 0",
			matchId, roundNo, "score", scoreChange).
		Count(&count).Error
	return count, err
}

// ListActiveActions 获取对局未撤销的操作日志
func (m *MatchModel) ListActiveActions(matchId int64) ([]MatchAction, error) {
	return m.ListActiveActionsWithTx(nil, matchId)
}

func (m *MatchModel) ListActiveActionsWithTx(tx *gorm.DB, matchId int64) ([]MatchAction, error) {
	var actions []MatchAction
	db := m.db
	if tx != nil {
		db = tx
	}
	err := db.
		Where("match_id = ? AND is_undone = 0", matchId).
		Order("round_no ASC, id ASC").
		Find(&actions).Error
	return actions, err
}

func (m *MatchModel) ListActiveActionsByMatchIDs(matchIds []int64) ([]MatchAction, error) {
	if len(matchIds) == 0 {
		return []MatchAction{}, nil
	}

	var actions []MatchAction
	err := m.db.
		Where("match_id IN ? AND is_undone = 0", matchIds).
		Order("match_id ASC, round_no ASC, id ASC").
		Find(&actions).Error
	return actions, err
}

// UndoAction 撤销操作
func (m *MatchModel) UndoAction(actionId int64) error {
	return m.UndoActionWithTx(nil, actionId)
}

func (m *MatchModel) UndoActionWithTx(tx *gorm.DB, actionId int64) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	return db.Model(&MatchAction{}).Where("id = ?", actionId).Update("is_undone", 1).Error
}

// ========== MatchAchievement 操作 ==========

// SaveAchievement 保存或更新特殊成绩
func (m *MatchModel) SaveAchievement(matchId int64, achievementType string, count int) error {
	return m.SaveAchievementWithTx(nil, matchId, achievementType, count)
}

func (m *MatchModel) SaveAchievementWithTx(tx *gorm.DB, matchId int64, achievementType string, count int, actor ...int) error {
	db := m.db
	if tx != nil {
		db = tx
	}
	achievementActor := 1
	if len(actor) > 0 {
		achievementActor = actor[0]
	}

	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "match_id"},
			{Name: "actor"},
			{Name: "achievement_type"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"count": gorm.Expr("count + ?", count),
		}),
	}).Create(&MatchAchievement{
		MatchId:         matchId,
		Actor:           achievementActor,
		AchievementType: achievementType,
		Count:           count,
	}).Error
}

// GetAchievements 获取对局的特殊成绩
func (m *MatchModel) GetAchievements(matchId int64) ([]MatchAchievement, error) {
	return m.GetAchievementsWithTx(nil, matchId)
}

func (m *MatchModel) GetAchievementsWithTx(tx *gorm.DB, matchId int64) ([]MatchAchievement, error) {
	var list []MatchAchievement
	db := m.db
	if tx != nil {
		db = tx
	}
	err := db.Where("match_id = ?", matchId).Find(&list).Error
	return list, err
}

// ========== Opponent 操作 ==========

// FindOrCreateOpponent 查找或创建对手
func (m *MatchModel) FindOrCreateOpponent(userId int64, name string, avatar string) (*Opponent, error) {
	return m.FindOrCreateOpponentWithTx(nil, userId, name, avatar)
}

func (m *MatchModel) FindOrCreateOpponentWithTx(tx *gorm.DB, userId int64, name string, avatar string) (*Opponent, error) {
	db := m.db
	if tx != nil {
		db = tx
	}

	var opponent Opponent
	err := db.Where("user_id = ? AND name = ?", userId, name).First(&opponent).Error
	if err == gorm.ErrRecordNotFound {
		opponent = Opponent{
			UserId: userId,
			Name:   name,
			Avatar: avatar,
		}
		if err := db.Create(&opponent).Error; err != nil {
			return nil, err
		}
		return &opponent, nil
	}
	// 如果对手已存在且传入了头像，更新头像
	if avatar != "" && opponent.Avatar != avatar {
		opponent.Avatar = avatar
		if err := db.Save(&opponent).Error; err != nil {
			return nil, err
		}
	}
	return &opponent, err
}

// SearchOpponents 搜索对手
func (m *MatchModel) SearchOpponents(userId int64, keyword string, limit int) ([]Opponent, error) {
	var list []Opponent
	err := m.db.Where("user_id = ? AND name LIKE ?", userId, "%"+keyword+"%").
		Limit(limit).Find(&list).Error
	return list, err
}

// GetOpponentMatchCount 获取与对手的对局次数
func (m *MatchModel) GetOpponentMatchCount(userId int64, opponentName string) (int64, error) {
	var count int64
	err := m.db.Model(&Match{}).
		Where("user_id = ? AND opponent_name = ? AND status = 2", userId, opponentName).
		Count(&count).Error
	return count, err
}

// DeleteLastRound 删除最后一局记录
func (m *MatchModel) DeleteLastRound(matchId int64) error {
	round, err := m.GetLastRoundWithTx(nil, matchId)
	if err != nil {
		return err
	}
	if round == nil {
		return gorm.ErrRecordNotFound
	}
	return m.DeleteRoundWithTx(nil, round.Id)
}

// UserStats 用户统计数据结构
type UserStats struct {
	TotalMatches int
	Wins         int
	Losses       int
	MaxWinStreak int
}

// GetUserStats 获取用户统计数据（支持双向查询）
func (m *MatchModel) GetUserStats(userId int64) (*UserStats, error) {
	// 查询用户参与的所有对局（作为发起方或对手方）
	var matches []struct {
		UserId int64
		Result *int
	}
	err := m.db.Model(&Match{}).
		Select("user_id, result").
		Where("(user_id = ? OR opponent_id = ?) AND status = 2", userId, userId).
		Where("match_mode = ? OR match_mode = '' OR match_mode IS NULL", MatchModeRanked).
		Scan(&matches).Error
	if err != nil {
		return nil, err
	}

	// 统计胜负
	total := len(matches)
	wins := 0
	losses := 0
	for _, match := range matches {
		if match.Result == nil {
			continue
		}
		isAsCreator := match.UserId == userId
		actualResult := *match.Result
		if !isAsCreator {
			// 作为对手时，胜负反转
			if actualResult == 1 {
				actualResult = 2
			} else if actualResult == 2 {
				actualResult = 1
			}
		}
		if actualResult == 1 {
			wins++
		} else if actualResult == 2 {
			losses++
		}
	}

	// 计算最高连胜
	maxWinStreak := m.calculateMaxWinStreak(userId)

	return &UserStats{
		TotalMatches: total,
		Wins:         wins,
		Losses:       losses,
		MaxWinStreak: maxWinStreak,
	}, nil
}

// calculateMaxWinStreak 计算最高连胜（支持双向查询）
func (m *MatchModel) calculateMaxWinStreak(userId int64) int {
	// 查询用户参与的所有对局，按时间排序
	var matches []struct {
		UserId    int64
		Result    *int
		MatchTime time.Time
	}
	err := m.db.Model(&Match{}).
		Select("user_id, result, match_time").
		Where("(user_id = ? OR opponent_id = ?) AND status = 2", userId, userId).
		Where("match_mode = ? OR match_mode = '' OR match_mode IS NULL", MatchModeRanked).
		Order("match_time ASC").
		Scan(&matches).Error
	if err != nil {
		return 0
	}

	maxStreak := 0
	currentStreak := 0
	for _, match := range matches {
		if match.Result == nil {
			currentStreak = 0
			continue
		}
		isAsCreator := match.UserId == userId
		actualResult := *match.Result
		if !isAsCreator {
			// 作为对手时，胜负反转
			if actualResult == 1 {
				actualResult = 2
			} else if actualResult == 2 {
				actualResult = 1
			}
		}
		if actualResult == 1 {
			currentStreak++
			if currentStreak > maxStreak {
				maxStreak = currentStreak
			}
		} else {
			currentStreak = 0
		}
	}
	return maxStreak
}

// OngoingMatch 正在进行的对局（带用户信息）
type OngoingMatch struct {
	Match
	Player1Name   string `json:"player1_name"`
	Player1Avatar string `json:"player1_avatar"`
	Player2Avatar string `json:"player2_avatar"`
}

type PublicMatchListOptions struct {
	Scope        string
	ViewerUserId int64
	Status       int
	GameType     int
	Offset       int
	Limit        int
}

type PublicMatchListRow struct {
	Match
	Player1Name   string `json:"player1_name"`
	Player1Avatar string `json:"player1_avatar"`
	Player2Name   string `json:"player2_name"`
	Player2Avatar string `json:"player2_avatar"`
}

// ListOngoingMatches 获取所有正在进行的对局列表
func (m *MatchModel) ListOngoingMatches(offset, limit int) ([]OngoingMatch, int64, error) {
	// 查询总数
	var total int64
	if err := m.db.Model(&Match{}).
		Where("status = 1 AND deleted_at IS NULL").
		Where("visibility = ? OR visibility = '' OR visibility IS NULL", MatchVisibilityPublic).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表（关联用户表获取玩家1和玩家2信息）
	var list []OngoingMatch
	err := m.db.Table("matches").
		Select(`matches.*, 
			COALESCE(u1.nickname, '玩家') as player1_name, 
			COALESCE(u1.avatar, '') as player1_avatar,
			COALESCE(u2.avatar, '') as player2_avatar`).
		Joins("LEFT JOIN users u1 ON matches.user_id = u1.id").
		Joins("LEFT JOIN users u2 ON matches.opponent_id = u2.id").
		Where("matches.status = 1 AND matches.deleted_at IS NULL").
		Where("matches.visibility = ? OR matches.visibility = '' OR matches.visibility IS NULL", MatchVisibilityPublic).
		Order("matches.match_time DESC").
		Offset(offset).
		Limit(limit).
		Scan(&list).Error

	return list, total, err
}

func (m *MatchModel) ListPublicMatches(options PublicMatchListOptions) ([]PublicMatchListRow, int64, error) {
	limit := options.Limit
	if limit <= 0 {
		limit = 20
	}

	base := m.db.Table("matches").
		Where("matches.deleted_at IS NULL")

	if options.Scope == "friends" {
		base = base.
			Where("matches.status IN ?", []int{1, 2}).
			Where(`EXISTS (
				SELECT 1 FROM friends
				WHERE friends.status = 1
				AND (
					(friends.user_id = ? AND (friends.friend_id = matches.user_id OR friends.friend_id = matches.opponent_id))
					OR
					(friends.friend_id = ? AND (friends.user_id = matches.user_id OR friends.user_id = matches.opponent_id))
				)
			) OR matches.user_id = ? OR matches.opponent_id = ? OR matches.referee_user_id = ?`,
				options.ViewerUserId, options.ViewerUserId,
				options.ViewerUserId, options.ViewerUserId, options.ViewerUserId)
		base = base.Where("matches.visibility = ? OR matches.visibility = '' OR matches.visibility IS NULL OR matches.user_id = ? OR matches.opponent_id = ? OR matches.referee_user_id = ?",
			MatchVisibilityPublic, options.ViewerUserId, options.ViewerUserId, options.ViewerUserId)
	} else {
		status := options.Status
		if status != 2 {
			status = 1
		}
		base = base.Where("matches.status = ?", status).
			Where("matches.visibility = ? OR matches.visibility = '' OR matches.visibility IS NULL", MatchVisibilityPublic)
	}

	if options.GameType > 0 {
		base = base.Where("matches.game_type = ?", options.GameType)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []PublicMatchListRow
	err := base.
		Select(`matches.*,
			COALESCE(u1.nickname, '玩家') as player1_name,
			COALESCE(u1.avatar, '') as player1_avatar,
			matches.opponent_name as player2_name,
			COALESCE(u2.avatar, '') as player2_avatar`).
		Joins("LEFT JOIN users u1 ON matches.user_id = u1.id").
		Joins("LEFT JOIN users u2 ON matches.opponent_id = u2.id").
		Order("matches.match_time DESC").
		Offset(options.Offset).
		Limit(limit).
		Scan(&list).Error

	return list, total, err
}

// OpponentStats 对手统计数据
type OpponentStats struct {
	OpponentId   int64     `json:"opponent_id"`
	OpponentName string    `json:"opponent_name"`
	Avatar       string    `json:"avatar"`
	TotalMatches int       `json:"total_matches"`
	Wins         int       `json:"wins"`
	Losses       int       `json:"losses"`
	LastMatchAt  time.Time `json:"last_match_at"`
}

type opponentProfile struct {
	Id       int64
	Nickname string
	Avatar   string
}

func buildOpponentStatsKey(opponentId int64, opponentName string) string {
	if opponentId > 0 {
		return fmt.Sprintf("user:%d", opponentId)
	}
	return "name:" + opponentName
}

// ListOpponentsWithStats 获取对手列表及统计数据（支持双向查询）
// 优化版本：消除 N+1 查询，使用批量查询
func (m *MatchModel) ListOpponentsWithStats(
	userId int64,
	keyword string,
	offset, limit int,
) ([]OpponentStats, int64, error) {
	// 1. 查询用户参与的所有对局，同时 JOIN 用户表获取发起方信息
	var matches []struct {
		UserId          int64
		OpponentId      *int64
		OpponentName    string
		Result          *int
		MatchTime       time.Time
		CreatorNickname string // 发起方昵称
	}
	err := m.db.Table("matches").
		Select("matches.user_id, matches.opponent_id, matches.opponent_name, matches.result, matches.match_time, COALESCE(u.nickname, '') as creator_nickname").
		Joins("LEFT JOIN users u ON matches.user_id = u.id").
		Where("(matches.user_id = ? OR matches.opponent_id = ?) AND matches.status = 2 AND matches.deleted_at IS NULL", userId, userId).
		Where("matches.match_mode = ? OR matches.match_mode = '' OR matches.match_mode IS NULL", MatchModeRanked).
		Scan(&matches).Error
	if err != nil {
		return nil, 0, err
	}

	// 2. 收集所有对手ID用于批量查询头像
	opponentIdSet := make(map[int64]bool)
	for _, match := range matches {
		isAsCreator := match.UserId == userId
		if isAsCreator {
			if match.OpponentId != nil {
				opponentIdSet[*match.OpponentId] = true
			}
		} else {
			opponentIdSet[match.UserId] = true
		}
	}

	// 3. 批量查询所有对手的资料
	userProfileMap := make(map[int64]opponentProfile)
	if len(opponentIdSet) > 0 {
		var opponentIds []int64
		for id := range opponentIdSet {
			opponentIds = append(opponentIds, id)
		}
		var users []opponentProfile
		if err := m.db.Table("users").Select("id, nickname, avatar").Where("id IN ?", opponentIds).Scan(&users).Error; err == nil {
			for _, u := range users {
				userProfileMap[u.Id] = u
			}
		}
	}

	// 4. 建立对手统计map
	opponentMap := make(map[string]*OpponentStats)
	for _, match := range matches {
		isAsCreator := match.UserId == userId
		var oppName string
		var oppId int64

		if isAsCreator {
			// 用户是发起方，对手是 opponent
			oppName = match.OpponentName
			if match.OpponentId != nil {
				oppId = *match.OpponentId
			}
		} else {
			// 用户是对手方，使用已经 JOIN 的发起方信息
			oppId = match.UserId
			oppName = match.CreatorNickname
			if oppName == "" {
				oppName = "玩家"
			}
		}

		if oppName == "" {
			continue
		}

		profile := userProfileMap[oppId]
		displayName := oppName
		displayAvatar := profile.Avatar
		if profile.Nickname != "" {
			displayName = profile.Nickname
		}

		// 关键词筛选
		if keyword != "" && !containsKeyword(displayName, keyword) {
			continue
		}

		key := buildOpponentStatsKey(oppId, oppName)
		if _, ok := opponentMap[key]; !ok {
			opponentMap[key] = &OpponentStats{
				OpponentId:   oppId,
				OpponentName: displayName,
				LastMatchAt:  match.MatchTime,
				Avatar:       displayAvatar,
			}
		}

		stats := opponentMap[key]
		stats.TotalMatches++
		if match.MatchTime.After(stats.LastMatchAt) {
			stats.LastMatchAt = match.MatchTime
			stats.OpponentName = displayName
			stats.Avatar = displayAvatar
		} else {
			if stats.OpponentName == "" {
				stats.OpponentName = displayName
			}
			if stats.Avatar == "" {
				stats.Avatar = displayAvatar
			}
		}

		if match.Result != nil {
			actualResult := *match.Result
			if !isAsCreator {
				// 作为对手时，胜负反转
				if actualResult == 1 {
					actualResult = 2
				} else if actualResult == 2 {
					actualResult = 1
				}
			}
			if actualResult == 1 {
				stats.Wins++
			} else if actualResult == 2 {
				stats.Losses++
			}
		}
	}

	// 5. 转换为切片
	var statsList []OpponentStats
	for _, stats := range opponentMap {
		statsList = append(statsList, *stats)
	}

	// 6. 按最后对局时间排序（使用更高效的排序）
	sort.Slice(statsList, func(i, j int) bool {
		return statsList[i].LastMatchAt.After(statsList[j].LastMatchAt)
	})

	total := int64(len(statsList))

	// 7. 分页
	if offset >= len(statsList) {
		return []OpponentStats{}, total, nil
	}
	end := offset + limit
	if end > len(statsList) {
		end = len(statsList)
	}

	return statsList[offset:end], total, nil
}

// containsKeyword 检查字符串是否包含关键词
func containsKeyword(s, keyword string) bool {
	return len(s) >= len(keyword) && (s == keyword || len(keyword) == 0 ||
		(len(s) > 0 && len(keyword) > 0 && findSubstring(s, keyword)))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// GetOverallOpponentStats 获取整体对手统计（总对手数、总胜场）（支持双向查询）
// 优化版本：使用 JOIN 查询替代循环内的单独查询
func (m *MatchModel) GetOverallOpponentStats(userId int64) (totalOpponents, totalWins int, err error) {
	// 查询用户参与的所有对局，同时 JOIN 用户表获取发起方信息
	var matches []struct {
		UserId          int64
		OpponentId      *int64
		OpponentName    string
		Result          *int
		CreatorNickname string
	}
	err = m.db.Table("matches").
		Select("matches.user_id, matches.opponent_id, matches.opponent_name, matches.result, COALESCE(u.nickname, '') as creator_nickname").
		Joins("LEFT JOIN users u ON matches.user_id = u.id").
		Where("(matches.user_id = ? OR matches.opponent_id = ?) AND matches.status = 2 AND matches.deleted_at IS NULL", userId, userId).
		Where("matches.match_mode = ? OR matches.match_mode = '' OR matches.match_mode IS NULL", MatchModeRanked).
		Scan(&matches).Error
	if err != nil {
		return 0, 0, err
	}

	// 统计对手数和胜场
	opponentSet := make(map[string]struct{})
	for _, match := range matches {
		isAsCreator := match.UserId == userId
		var oppName string
		var oppId int64

		if isAsCreator {
			oppName = match.OpponentName
			if match.OpponentId != nil {
				oppId = *match.OpponentId
			}
		} else {
			// 使用已经 JOIN 的发起方信息
			oppId = match.UserId
			oppName = match.CreatorNickname
		}

		if oppName != "" || oppId > 0 {
			opponentSet[buildOpponentStatsKey(oppId, oppName)] = struct{}{}
		}

		if match.Result != nil {
			actualResult := *match.Result
			if !isAsCreator {
				if actualResult == 1 {
					actualResult = 2
				} else if actualResult == 2 {
					actualResult = 1
				}
			}
			if actualResult == 1 {
				totalWins++
			}
		}
	}

	totalOpponents = len(opponentSet)
	return totalOpponents, totalWins, nil
}

// GetMaxSingleScore 获取用户的单杆最高得分
// 查询用户在所有对局中的单局最高得分（作为创建者时取 my_score，作为对手时取 opponent_score）
func (m *MatchModel) GetMaxSingleScore(userId int64) (int, error) {
	var maxScore int

	// 作为创建者(player1)时的最高得分
	var player1MaxScore int
	err := m.db.Table("match_rounds mr").
		Joins("INNER JOIN matches m ON mr.match_id = m.id").
		Where("m.user_id = ? AND m.status = 2 AND m.deleted_at IS NULL", userId).
		Where("m.match_mode = ? OR m.match_mode = '' OR m.match_mode IS NULL", MatchModeRanked).
		Select("COALESCE(MAX(mr.my_score), 0)").
		Scan(&player1MaxScore).Error
	if err != nil {
		return 0, err
	}

	// 作为对手(player2)时的最高得分
	var player2MaxScore int
	err = m.db.Table("match_rounds mr").
		Joins("INNER JOIN matches m ON mr.match_id = m.id").
		Where("m.opponent_id = ? AND m.status = 2 AND m.deleted_at IS NULL", userId).
		Where("m.match_mode = ? OR m.match_mode = '' OR m.match_mode IS NULL", MatchModeRanked).
		Select("COALESCE(MAX(mr.opponent_score), 0)").
		Scan(&player2MaxScore).Error
	if err != nil {
		return 0, err
	}

	// 返回两者中的最大值
	maxScore = player1MaxScore
	if player2MaxScore > maxScore {
		maxScore = player2MaxScore
	}

	return maxScore, nil
}

// CountTotal 获取对局总数
func (m *MatchModel) CountTotal() (int64, error) {
	var count int64
	err := m.db.Model(&Match{}).Count(&count).Error
	return count, err
}

// CountOngoing 获取进行中对局数
func (m *MatchModel) CountOngoing() (int64, error) {
	var count int64
	err := m.db.Model(&Match{}).Where("status = ?", 1).Count(&count).Error
	return count, err
}

// FindRecent 获取最近对局
func (m *MatchModel) FindRecent(limit int) ([]Match, error) {
	if limit <= 0 {
		limit = 10
	}
	var matches []Match
	err := m.db.Order("id DESC").Limit(limit).Find(&matches).Error
	return matches, err
}

// FindListForAdmin 管理员获取对局列表
func (m *MatchModel) FindListForAdmin(page, pageSize int, status, gameType int) ([]Match, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := m.db.Model(&Match{})
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	if gameType >= 0 {
		query = query.Where("game_type = ?", gameType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var matches []Match
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&matches).Error
	return matches, total, err
}
