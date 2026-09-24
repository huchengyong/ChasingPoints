package model

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 约球状态：0待回应 1已接受(含等待) 2已拒绝 3已过期 4已取消 5已放弃 6已开局 7已完成 8对局取消结束 9互邀归并别名
const (
	ChallengeStatusPending        = 0
	ChallengeStatusAccepted       = 1
	ChallengeStatusRejected       = 2
	ChallengeStatusExpired        = 3
	ChallengeStatusCancelled      = 4
	ChallengeStatusAbandoned      = 5
	ChallengeStatusStarted        = 6
	ChallengeStatusCompleted      = 7
	ChallengeStatusMatchCancelled = 8
	ChallengeStatusMerged         = 9

	ChallengeCloseReasonRejected  = "rejected"
	ChallengeCloseReasonCancelled = "cancelled"
	ChallengeCloseReasonAbandoned = "abandoned"
	ChallengeCloseReasonExpired   = "expired"
	ChallengeCloseReasonSwitched  = "switched_to_other"
	ChallengeCloseReasonMerged    = "merged"
	ChallengeCloseReasonCompleted = "match_completed"
	ChallengeCloseReasonMatchGone = "match_cancelled"
	ChallengeCloseReasonDeleted   = "account_deleted"
)

type Challenge struct {
	Id                  int64      `gorm:"primarykey"`
	FromUserId          int64      `gorm:"not null;index"`
	ToUserId            int64      `gorm:"not null;index"`
	GameType            int        `gorm:"not null"`
	ScheduledDate       *time.Time `gorm:"type:date"`
	StartHour           *int       `gorm:"type:tinyint"`
	EndHour             *int       `gorm:"type:tinyint"`
	MatchMode           string     `gorm:"size:20"`
	Visibility          string     `gorm:"size:20"`
	MatchFormat         string     `gorm:"size:20"`
	TargetWins          int        `gorm:"not null;default:0"`
	SnookerRulesVersion int        `gorm:"not null;default:0"`
	BestOfFrames        int        `gorm:"not null;default:0"`
	SnookerFormat       string     `gorm:"size:20"`
	SnookerTargetWins   int        `gorm:"not null;default:0"`
	StartingActor       int        `gorm:"not null;default:0"`
	WaitingUserId       *int64     `gorm:"index"`
	WaitingEnteredAt    *time.Time
	MergedIntoId        *int64    `gorm:"index:idx_challenges_merged_into"`
	CloseReason         string    `gorm:"size:50"`
	Message             string    `gorm:"size:200"`
	Status              int       `gorm:"not null;default:0;index:idx_challenges_status_expires,priority:1"`
	MatchId             *int64    `gorm:"index"` // 旧字段：不再写入，权威关联见 matches.challenge_id
	ExpiresAt           time.Time `gorm:"not null;index:idx_challenges_status_expires,priority:2"`
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"`
}

func (Challenge) TableName() string {
	return "challenges"
}

// ChallengeListRow 列表/详情联表行。
type ChallengeListRow struct {
	Id                  int64      `gorm:"column:id"`
	FromUserId          int64      `gorm:"column:from_user_id"`
	ToUserId            int64      `gorm:"column:to_user_id"`
	GameType            int        `gorm:"column:game_type"`
	ScheduledDate       *time.Time `gorm:"column:scheduled_date"`
	StartHour           *int       `gorm:"column:start_hour"`
	EndHour             *int       `gorm:"column:end_hour"`
	MatchMode           string     `gorm:"column:match_mode"`
	Visibility          string     `gorm:"column:visibility"`
	MatchFormat         string     `gorm:"column:match_format"`
	TargetWins          int        `gorm:"column:target_wins"`
	SnookerRulesVersion int        `gorm:"column:snooker_rules_version"`
	BestOfFrames        int        `gorm:"column:best_of_frames"`
	SnookerFormat       string     `gorm:"column:snooker_format"`
	SnookerTargetWins   int        `gorm:"column:snooker_target_wins"`
	StartingActor       int        `gorm:"column:starting_actor"`
	WaitingUserId       *int64     `gorm:"column:waiting_user_id"`
	WaitingEnteredAt    *time.Time `gorm:"column:waiting_entered_at"`
	MergedIntoId        *int64     `gorm:"column:merged_into_id"`
	CloseReason         string     `gorm:"column:close_reason"`
	Message             string     `gorm:"column:message"`
	Status              int        `gorm:"column:status"`
	ExpiresAt           time.Time  `gorm:"column:expires_at"`
	CreatedAt           time.Time  `gorm:"column:created_at"`
	LinkedMatchId       int64      `gorm:"column:linked_match_id"`
	FromNickname        string     `gorm:"column:from_nickname"`
	FromAvatar          string     `gorm:"column:from_avatar"`
	ToNickname          string     `gorm:"column:to_nickname"`
	ToAvatar            string     `gorm:"column:to_avatar"`
}

const (
	challengeReadLimit      = 200
	challengeHistoryOpLimit = 50
)

type ChallengeModel struct {
	db *gorm.DB
}

func NewChallengeModel(db *gorm.DB) *ChallengeModel {
	return &ChallengeModel{db: db}
}

func (m *ChallengeModel) Create(challenge *Challenge) error {
	return m.db.Create(challenge).Error
}

func (m *ChallengeModel) CreateWithTx(tx *gorm.DB, challenge *Challenge) error {
	if tx != nil {
		return tx.Create(challenge).Error
	}
	return m.Create(challenge)
}

func (m *ChallengeModel) FindById(id int64) (*Challenge, error) {
	var challenge Challenge
	err := m.db.First(&challenge, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &challenge, err
}

func (m *ChallengeModel) FindByIdForUpdateWithTx(tx *gorm.DB, id int64) (*Challenge, error) {
	var challenge Challenge
	db := m.db
	if tx != nil {
		db = tx
	}
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&challenge, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &challenge, err
}

// FindMainById 互邀归并别名只指向主记录一层，这里顺带防御性跟随。
func (m *ChallengeModel) FindMainById(id int64) (*Challenge, error) {
	current := id
	for i := 0; i < 5; i++ {
		challenge, err := m.FindById(current)
		if err != nil || challenge == nil {
			return nil, err
		}
		if challenge.MergedIntoId == nil || *challenge.MergedIntoId <= 0 || *challenge.MergedIntoId == current {
			return challenge, nil
		}
		current = *challenge.MergedIntoId
	}
	return nil, nil
}

// FindByIdsForUpdateWithTx 按 ID 升序锁定邀约行，避免交叉锁顺序。
func (m *ChallengeModel) FindByIdsForUpdateWithTx(tx *gorm.DB, ids []int64) ([]*Challenge, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	db := m.db
	if tx != nil {
		db = tx
	}
	ordered := append([]int64(nil), ids...)
	for i := 1; i < len(ordered); i++ {
		for j := i; j > 0 && ordered[j] < ordered[j-1]; j-- {
			ordered[j], ordered[j-1] = ordered[j-1], ordered[j]
		}
	}
	var list []*Challenge
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", ordered).
		Order("id ASC").
		Find(&list).Error
	return list, err
}

// FindActivePendingByFromUserWithTx 发送方当前仍有效（未回应未过期）的待回应邀请。
func (m *ChallengeModel) FindActivePendingByFromUserWithTx(tx *gorm.DB, fromUserId int64, now time.Time) (*Challenge, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var challenge Challenge
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("from_user_id = ? AND status = ? AND expires_at > ?", fromUserId, ChallengeStatusPending, now).
		Order("id ASC").
		First(&challenge).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &challenge, err
}

// FindOpenJoinedByUserWithTx 用户尚未结束的已接受约球（有效已接受或已开局），excludeId 用于接受时排除本条。
func (m *ChallengeModel) FindOpenJoinedByUserWithTx(tx *gorm.DB, userId int64, excludeId int64, now time.Time) (*Challenge, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	query := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("(from_user_id = ? OR to_user_id = ?) AND ((status = ? AND expires_at > ?) OR status = ?)",
			userId, userId, ChallengeStatusAccepted, now, ChallengeStatusStarted)
	if excludeId > 0 {
		query = query.Where("id <> ?", excludeId)
	}
	var challenge Challenge
	err := query.Order("id ASC").First(&challenge).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &challenge, err
}

// FindMutualPendingWithTx 本人发给对手、同球种同比赛类型的有效待回应邀请（互邀合并用）。
func (m *ChallengeModel) FindMutualPendingWithTx(tx *gorm.DB, fromUserId, toUserId int64, gameType int, matchMode string, now time.Time) (*Challenge, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var challenge Challenge
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("from_user_id = ? AND to_user_id = ? AND game_type = ? AND match_mode = ? AND status = ? AND expires_at > ? AND merged_into_id IS NULL",
			fromUserId, toUserId, gameType, matchMode, ChallengeStatusPending, now).
		Order("id ASC").
		First(&challenge).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &challenge, err
}

// FindOpenWaitingByUserWithTx 本人正在等待进入的有效已接受约球（扫码开局时清除等待用）。
func (m *ChallengeModel) FindOpenWaitingByUserWithTx(tx *gorm.DB, userId int64, now time.Time) (*Challenge, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var challenge Challenge
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("(from_user_id = ? OR to_user_id = ?) AND status = ? AND waiting_user_id = ? AND expires_at > ?",
			userId, userId, ChallengeStatusAccepted, userId, now).
		Order("id ASC").
		First(&challenge).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &challenge, err
}

// SetWaitingWithTx 写入本人等待位；仅限已接受未开局。
func (m *ChallengeModel) SetWaitingWithTx(tx *gorm.DB, challengeId, userId int64, now time.Time) (bool, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	result := db.Model(&Challenge{}).
		Where("id = ? AND status = ? AND waiting_user_id IS NULL", challengeId, ChallengeStatusAccepted).
		Updates(map[string]interface{}{"waiting_user_id": userId, "waiting_entered_at": now})
	return result.RowsAffected > 0, result.Error
}

// ClearWaitingWithTx 退出等待：只清除本人等待态，保留约球。
func (m *ChallengeModel) ClearWaitingWithTx(tx *gorm.DB, challengeId, userId int64) (bool, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	result := db.Model(&Challenge{}).
		Where("id = ? AND status = ? AND waiting_user_id = ?", challengeId, ChallengeStatusAccepted, userId).
		Updates(map[string]interface{}{"waiting_user_id": nil, "waiting_entered_at": nil})
	return result.RowsAffected > 0, result.Error
}

// UpdateStatusWithTx 条件状态流转；extra 可携带 close_reason 等字段。
func (m *ChallengeModel) UpdateStatusWithTx(tx *gorm.DB, challengeId int64, fromStatuses []int, toStatus int, extra map[string]interface{}) (bool, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	updates := map[string]interface{}{"status": toStatus}
	for key, value := range extra {
		updates[key] = value
	}
	query := db.Model(&Challenge{}).Where("id = ?", challengeId)
	if len(fromStatuses) > 0 {
		query = query.Where("status IN ?", fromStatuses)
	}
	result := query.Updates(updates)
	return result.RowsAffected > 0, result.Error
}

// MarkCompletedByMatchIdWithTx 比赛完成：仅已开局约球流转为已完成。
func (m *ChallengeModel) MarkCompletedByMatchIdWithTx(tx *gorm.DB, challengeId int64) (bool, error) {
	return m.UpdateStatusWithTx(tx, challengeId, []int{ChallengeStatusStarted}, ChallengeStatusCompleted,
		map[string]interface{}{"close_reason": ChallengeCloseReasonCompleted})
}

// MarkMatchCancelledWithTx 比赛取消/作废同步结束约球。
func (m *ChallengeModel) MarkMatchCancelledWithTx(tx *gorm.DB, challengeId int64, reason string) (bool, error) {
	if reason == "" {
		reason = ChallengeCloseReasonMatchGone
	}
	return m.UpdateStatusWithTx(tx, challengeId, []int{ChallengeStatusStarted}, ChallengeStatusMatchCancelled,
		map[string]interface{}{"close_reason": reason})
}

// CountReceivedPending 收到的有效待回应邀请数量。
func (m *ChallengeModel) CountReceivedPending(userId int64, now time.Time) (int64, error) {
	var count int64
	err := m.db.Model(&Challenge{}).
		Where("to_user_id = ? AND status = ? AND expires_at > ?", userId, ChallengeStatusPending, now).
		Count(&count).Error
	return count, err
}

// FindCurrentSummaryByUser 当前权威约球摘要：已开局 > 有效已接受（含等待） > 本人发出的有效待回应。
func (m *ChallengeModel) FindCurrentSummaryByUser(userId int64, now time.Time) (*Challenge, error) {
	var challenge Challenge
	err := m.db.
		Where("(from_user_id = ? OR to_user_id = ?) AND (status = ? OR (status = ? AND expires_at > ?) OR (from_user_id = ? AND status = ? AND expires_at > ?))",
			userId, userId, ChallengeStatusStarted, ChallengeStatusAccepted, now, userId, ChallengeStatusPending, now).
		Order("CASE status WHEN 6 THEN 0 WHEN 1 THEN 1 ELSE 2 END, id DESC").
		First(&challenge).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &challenge, err
}

const challengeRowSelect = `c.id, c.from_user_id, c.to_user_id, c.game_type, c.scheduled_date, c.start_hour, c.end_hour,
	c.match_mode, c.visibility, c.match_format, c.target_wins, c.snooker_rules_version, c.best_of_frames,
	c.snooker_format, c.snooker_target_wins, c.starting_actor, c.waiting_user_id, c.waiting_entered_at,
	c.merged_into_id, c.close_reason, c.message, c.status, c.expires_at, c.created_at,
	m.id AS linked_match_id,
	f.nickname AS from_nickname, f.avatar AS from_avatar, t.nickname AS to_nickname, t.avatar AS to_avatar`

// ListActiveByUserWithRows 有效活动列表：收到待回应 + 已接受/已开局 + 本人发出待回应，有界。
func (m *ChallengeModel) ListActiveByUserWithRows(userId int64, now time.Time) ([]ChallengeListRow, error) {
	var rows []ChallengeListRow
	err := m.db.Table("challenges AS c").
		Select(challengeRowSelect).
		Joins("LEFT JOIN users AS f ON f.id = c.from_user_id").
		Joins("LEFT JOIN users AS t ON t.id = c.to_user_id").
		Joins("LEFT JOIN matches AS m ON m.challenge_id = c.id").
		Where("(c.from_user_id = ? OR c.to_user_id = ?) AND (c.status = ? OR (c.status = ? AND c.expires_at > ?) OR (c.status = ? AND c.expires_at > ?))",
			userId, userId, ChallengeStatusStarted, ChallengeStatusAccepted, now, ChallengeStatusPending, now).
		Order("c.id DESC").
		Limit(challengeReadLimit).
		Scan(&rows).Error
	return rows, err
}

// ListReceivedPendingWithRows 收到的有效待回应邀请（收到的邀请列表）。
func (m *ChallengeModel) ListReceivedPendingWithRows(userId int64, now time.Time) ([]ChallengeListRow, error) {
	var rows []ChallengeListRow
	err := m.db.Table("challenges AS c").
		Select(challengeRowSelect).
		Joins("LEFT JOIN users AS f ON f.id = c.from_user_id").
		Joins("LEFT JOIN users AS t ON t.id = c.to_user_id").
		Joins("LEFT JOIN matches AS m ON m.challenge_id = c.id").
		Where("c.to_user_id = ? AND c.status = ? AND c.expires_at > ? AND c.merged_into_id IS NULL",
			userId, ChallengeStatusPending, now).
		Order("c.id DESC").
		Limit(challengeReadLimit).
		Scan(&rows).Error
	return rows, err
}

// ListHistoryWithRows 主记录历史分页（id 游标倒序），别名归并不重复展示。
func (m *ChallengeModel) ListHistoryWithRows(userId int64, beforeId int64, limit int) ([]ChallengeListRow, error) {
	if limit <= 0 || limit > challengeHistoryOpLimit {
		limit = 20
	}
	query := m.db.Table("challenges AS c").
		Select(challengeRowSelect).
		Joins("LEFT JOIN users AS f ON f.id = c.from_user_id").
		Joins("LEFT JOIN users AS t ON t.id = c.to_user_id").
		Joins("LEFT JOIN matches AS m ON m.challenge_id = c.id").
		Where("(c.from_user_id = ? OR c.to_user_id = ?) AND c.merged_into_id IS NULL", userId, userId)
	if beforeId > 0 {
		query = query.Where("c.id < ?", beforeId)
	}
	var rows []ChallengeListRow
	err := query.Order("c.id DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

// FindRowById 联表读取单条约球行。
func (m *ChallengeModel) FindRowById(id int64) (*ChallengeListRow, error) {
	var row ChallengeListRow
	err := m.db.Table("challenges AS c").
		Select(challengeRowSelect).
		Joins("LEFT JOIN users AS f ON f.id = c.from_user_id").
		Joins("LEFT JOIN users AS t ON t.id = c.to_user_id").
		Joins("LEFT JOIN matches AS m ON m.challenge_id = c.id").
		Where("c.id = ?", id).
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &row, err
}

// FindActivePendingToOtherWithTx 本人发给他人（不含指定对象）的有效待回应邀请。
func (m *ChallengeModel) FindActivePendingToOtherWithTx(tx *gorm.DB, fromUserId int64, excludeToUserId int64, now time.Time) (*Challenge, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var challenge Challenge
	query := db.Where("from_user_id = ? AND status = ? AND expires_at > ? AND merged_into_id IS NULL", fromUserId, ChallengeStatusPending, now)
	if tx != nil {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if excludeToUserId > 0 {
		query = query.Where("to_user_id <> ?", excludeToUserId)
	}
	err := query.Order("id ASC").First(&challenge).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &challenge, err
}

// HasAnyMutualPendingWithTx 本人发给对手的任意有效待回应邀请（不限类型）。
func (m *ChallengeModel) HasAnyMutualPendingWithTx(tx *gorm.DB, fromUserId, toUserId int64, now time.Time) (bool, error) {
	db := m.db
	if tx != nil {
		db = tx
	}
	var count int64
	err := db.Model(&Challenge{}).
		Where("from_user_id = ? AND to_user_id = ? AND status = ? AND expires_at > ? AND merged_into_id IS NULL",
			fromUserId, toUserId, ChallengeStatusPending, now).
		Count(&count).Error
	return count > 0, err
}

// ListExpiredUnstarted 到期未开局（含已接受/等待），供 worker 落库。
func (m *ChallengeModel) ListExpiredUnstarted(limit int, now time.Time) ([]Challenge, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	var list []Challenge
	err := m.db.Where("status IN ? AND expires_at <= ?", []int{ChallengeStatusPending, ChallengeStatusAccepted}, now).
		Order("expires_at ASC, id ASC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

// MarkExpiredIfUnstarted 条件过期：不得改写已开局记录。
func (m *ChallengeModel) MarkExpiredIfUnstarted(challengeId int64, now time.Time) (bool, error) {
	return m.UpdateStatusWithTx(nil, challengeId, []int{ChallengeStatusPending, ChallengeStatusAccepted}, ChallengeStatusExpired,
		map[string]interface{}{"close_reason": ChallengeCloseReasonExpired})
}
