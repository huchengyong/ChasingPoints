package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const PersonalDataExportMaxBatchSize = 100

var personalDataExportCategories = []string{
	"profile",
	"authentication_connections",
	"notification_preferences",
	"friends",
	"friend_requests",
	"friend_blacklists",
	"follows",
	"challenges",
	"notifications",
	"social_posts",
	"social_comments",
	"social_likes",
	"matches",
	"opponent_records",
	"competitive_stats",
	"opponent_stats",
	"achievements",
	"titles",
	"achievement_progress",
	"rankings",
	"rank_changes",
	"reputation_profile",
	"reputation_logs",
	"venue_checkins",
	"member_subscriptions",
	"member_growth_profile",
	"member_growth_logs",
	"favorite_venue_rewards",
	"feedback_tickets",
	"tournament_participations",
	"season_records",
	"season_challenges",
}

type PersonalDataExportRecord struct {
	ID   int64
	Data map[string]any
}

type UserDataLifecycleModel struct {
	db *gorm.DB
}

type accountDeletionStatement struct {
	query string
	args  []any
}

func NewUserDataLifecycleModel(db *gorm.DB) *UserDataLifecycleModel {
	return &UserDataLifecycleModel{db: db}
}

func (m *UserDataLifecycleModel) DeleteAccountRelationsWithTx(tx *gorm.DB, userID int64, phone string) error {
	if m == nil || m.db == nil || tx == nil || userID <= 0 {
		return errors.New("invalid account deletion scope")
	}

	statements := []accountDeletionStatement{
		{"DELETE FROM social_post_likes WHERE user_id = ? OR post_id IN (SELECT id FROM social_posts WHERE user_id = ?)", []any{userID, userID}},
		{"DELETE FROM social_post_comments WHERE user_id = ? OR post_id IN (SELECT id FROM social_posts WHERE user_id = ?)", []any{userID, userID}},
		{"DELETE FROM social_posts WHERE user_id = ?", []any{userID}},
		{"DELETE FROM friends WHERE user_id = ? OR friend_id = ?", []any{userID, userID}},
		{"DELETE FROM friend_requests WHERE from_user_id = ? OR to_user_id = ?", []any{userID, userID}},
		{"DELETE FROM friend_blacklists WHERE user_id = ? OR blocked_user_id = ?", []any{userID, userID}},
		{"DELETE FROM follows WHERE follower_id = ? OR following_id = ?", []any{userID, userID}},
		{"DELETE FROM challenges WHERE from_user_id = ? OR to_user_id = ?", []any{userID, userID}},
		{"DELETE FROM notifications WHERE user_id = ?", []any{userID}},
		{"DELETE FROM user_notification_preferences WHERE user_id = ?", []any{userID}},
		{"DELETE FROM opponents WHERE user_id = ?", []any{userID}},
		{"DELETE FROM match_participant_results WHERE user_id = ?", []any{userID}},
		{"DELETE FROM user_competitive_stats WHERE user_id = ?", []any{userID}},
		{"DELETE FROM user_opponent_stats WHERE user_id = ?", []any{userID}},
		{"DELETE FROM user_opponent_strength_buckets WHERE user_id = ?", []any{userID}},
		{"DELETE FROM user_ranking WHERE user_id = ?", []any{userID}},
		{"DELETE FROM user_achievements WHERE user_id = ?", []any{userID}},
		{"DELETE FROM user_titles WHERE user_id = ?", []any{userID}},
		{"DELETE FROM achievement_progress_events WHERE user_id = ?", []any{userID}},
		{"DELETE FROM user_reputation_profiles WHERE user_id = ?", []any{userID}},
		{"DELETE FROM member_growth_profiles WHERE user_id = ?", []any{userID}},
		{"DELETE FROM member_growth_logs WHERE user_id = ?", []any{userID}},
		{"DELETE FROM favorite_venue_reward_records WHERE user_id = ?", []any{userID}},
		{"DELETE FROM venue_checkins WHERE user_id = ?", []any{userID}},
	}
	if strings.TrimSpace(phone) != "" {
		statements = append(statements, accountDeletionStatement{"DELETE FROM verification_codes WHERE phone = ?", []any{phone}})
	}
	return executeAccountDeletionStatements(tx, statements)
}

func (m *UserDataLifecycleModel) AnonymizeRetainedAccountFactsWithTx(tx *gorm.DB, userID int64) error {
	if m == nil || m.db == nil || tx == nil || userID <= 0 {
		return errors.New("invalid account anonymization scope")
	}

	deletedNameKey := fmt.Sprintf("deleted:%d", userID)
	statements := []accountDeletionStatement{
		{"UPDATE matches SET remark = '' WHERE (user_id = ? OR opponent_id = ? OR referee_user_id = ?) AND NOT (status = ? AND remark = ?)", []any{userID, userID, userID, 3, "account_deleted"}},
		{"UPDATE matches SET opponent_name = ? WHERE opponent_id = ?", []any{DeletedUserDisplayName, userID}},
		{"UPDATE matches SET referee_user_id = NULL WHERE referee_user_id = ?", []any{userID}},
		{"UPDATE matches SET completed_by_user_id = NULL WHERE completed_by_user_id = ?", []any{userID}},
		{"UPDATE matches SET finish_requested_by = NULL WHERE finish_requested_by = ?", []any{userID}},
		{"UPDATE opponents SET name = ?, avatar = '', linked_user_id = NULL WHERE linked_user_id = ?", []any{DeletedUserDisplayName, userID}},
		{"UPDATE match_participant_results SET opponent_name_key = ?, opponent_name = ?, opponent_avatar = '' WHERE opponent_user_id = ?", []any{deletedNameKey, DeletedUserDisplayName, userID}},
		{"UPDATE user_opponent_stats SET opponent_name_key = ?, opponent_name = ?, opponent_avatar = '' WHERE opponent_user_id = ?", []any{deletedNameKey, DeletedUserDisplayName, userID}},
		{"UPDATE rank_change_logs SET remark = '' WHERE user_id = ? OR operator_user_id = ?", []any{userID, userID}},
		{"UPDATE user_reputation_logs SET reason_detail = '' WHERE user_id = ?", []any{userID}},
		{"UPDATE feedback_tickets SET content = ?, contact = '' WHERE user_id = ?", []any{DeletedUserDisplayName, userID}},
	}
	return executeAccountDeletionStatements(tx, statements)
}

func executeAccountDeletionStatements(tx *gorm.DB, statements []accountDeletionStatement) error {
	for _, statement := range statements {
		if err := tx.Exec(statement.query, statement.args...).Error; err != nil {
			return err
		}
	}
	return nil
}

func PersonalDataExportCategories() []string {
	return append([]string(nil), personalDataExportCategories...)
}

func (m *UserDataLifecycleModel) ListExportRecords(userID int64, category string, afterID int64, snapshotAt time.Time, limit int) ([]PersonalDataExportRecord, bool, error) {
	if m == nil || m.db == nil {
		return nil, false, errors.New("user data lifecycle db is nil")
	}
	if userID <= 0 || afterID < 0 || snapshotAt.IsZero() {
		return nil, false, errors.New("invalid personal data export scope")
	}
	limit = normalizePersonalDataExportLimit(limit)

	switch category {
	case "profile":
		return m.listExportRows("users", "id, phone, nickname, avatar, member_expires_at, hide_match_record, created_at, updated_at", "id = ? AND deleted_at IS NULL", []any{userID}, afterID, snapshotAt, limit, nil)
	case "authentication_connections":
		return m.listExportRows("user_oauth", "id, provider, open_id, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, redactOAuthIdentifier)
	case "notification_preferences":
		return m.listExportRows("user_notification_preferences", "id, match_result_enabled, friend_request_enabled, challenge_enabled, tournament_enabled, follow_enabled, created_at, updated_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "friends":
		return m.listExportRows("friends", "id, friend_id, status, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "friend_requests":
		return m.listExportRows("friend_requests", "id, from_user_id, to_user_id, status, message, created_at, updated_at", "from_user_id = ? OR to_user_id = ?", []any{userID, userID}, afterID, snapshotAt, limit, func(row map[string]any) {
			if exportInt64(row["from_user_id"]) == userID {
				row["direction"] = "sent"
			} else {
				row["direction"] = "received"
			}
		})
	case "friend_blacklists":
		return m.listExportRows("friend_blacklists", "id, blocked_user_id, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "follows":
		return m.listExportRows("follows", "id, follower_id, following_id, created_at", "follower_id = ? OR following_id = ?", []any{userID, userID}, afterID, snapshotAt, limit, func(row map[string]any) {
			if exportInt64(row["follower_id"]) == userID {
				row["direction"] = "following"
			} else {
				row["direction"] = "follower"
			}
		})
	case "challenges":
		return m.listExportRows("challenges", "id, from_user_id, to_user_id, game_type, message, status, match_id, expires_at, created_at, updated_at", "from_user_id = ? OR to_user_id = ?", []any{userID, userID}, afterID, snapshotAt, limit, nil)
	case "notifications":
		return m.listExportRows("notifications", "id, type, title, content, is_read, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "social_posts":
		return m.listExportRows("social_posts", "id, content, images, post_type, match_id, likes_count, comments_count, status, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "social_comments":
		return m.listExportRows("social_post_comments", "id, post_id, content, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "social_likes":
		return m.listExportRows("social_post_likes", "id, post_id, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "matches":
		return m.listExportRows("matches", "id, game_type, game_mode, snooker_rules_version, best_of_frames, snooker_format, snooker_target_wins, match_format, target_wins, starting_actor, match_mode, visibility, my_score, opponent_score, status, result, match_time, end_time, completed_at, remark, created_at", "(user_id = ? OR opponent_id = ? OR referee_user_id = ?) AND deleted_at IS NULL", []any{userID, userID, userID}, afterID, snapshotAt, limit, nil)
	case "opponent_records":
		return m.listExportRows("opponents", "id, name, avatar, linked_user_id, created_at, updated_at", "user_id = ? AND deleted_at IS NULL", []any{userID}, afterID, snapshotAt, limit, nil)
	case "competitive_stats":
		return m.listExportRows("user_competitive_stats", "id, game_type, total_matches, wins, losses, draws, current_win_streak, max_win_streak, highest_score, highest_break, duration_count, duration_sum_seconds, duration_min_seconds, duration_max_seconds, last_match_id, last_match_at, revision, created_at, updated_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "opponent_stats":
		return m.listExportRows("user_opponent_stats", "id, opponent_user_id, opponent_name, opponent_avatar, game_type, total_matches, wins, losses, draws, score_diff_sum, current_win_streak, max_win_streak, last_match_id, last_match_at, created_at, updated_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "achievements":
		return m.listExportRows("user_achievements", "id, achievement_id, progress, unlocked, reward_granted, unlocked_at, unlocked_source_type, unlocked_source_id, reward_granted_at, created_at, updated_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "titles":
		return m.listExportRows("user_titles", "id, title_key, title_name, source, source_type, source_ref_id, source_ref_name, granted_by_achievement_id, equipped, equipped_at, granted_at, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "achievement_progress":
		return m.listExportRows("achievement_progress_events", "id, source_type, source_id, game_type, metric_key, metric_value, occurred_at, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "rankings":
		return m.listExportRows("user_ranking", "id, game_type, rank_score, rank_level, total_wins, total_losses, current_streak, max_streak, created_at, updated_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "rank_changes":
		return m.listExportRows("rank_change_logs", "id, match_id, change_type, game_type, result, base_score, achievement_score, final_change, before_score, after_score, before_level, after_level, remark, effective_at, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "reputation_profile":
		return m.listExportRows("user_reputation_profiles", "id, reputation_score, last_recovered_at, last_penalized_at, ban_until, total_penalty_count, total_abnormal_match_count, created_at, updated_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "reputation_logs":
		return m.listExportRows("user_reputation_logs", "id, match_id, change_type, change_score, before_score, after_score, reason_code, reason_detail, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "venue_checkins":
		return m.listExportRows("venue_checkins", "id, venue_id, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "member_subscriptions":
		return m.listExportRows("member_subscription_orders", "id, plan_code, plan_name, duration_days, amount_fen, pay_channel, status, paid_at, member_expires_at_before, member_expires_at_after, created_at, updated_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "member_growth_profile":
		return m.listExportRows("member_growth_profiles", "id, growth_points, growth_level, today_growth_count, today_growth_date, last_growth_at, created_at, updated_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "member_growth_logs":
		return m.listExportRows("member_growth_logs", "id, match_id, growth_points, source, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "favorite_venue_rewards":
		return m.listExportRows("favorite_venue_reward_records", "id, activity_key, venue_id, reward_days, member_expires_at_before, member_expires_at_after, granted_at, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "feedback_tickets":
		return m.listExportRows("feedback_tickets", "id, source, category, content, contact, status, created_at, updated_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "tournament_participations":
		return m.listExportRows("tournament_participants", "id, tournament_id, seed, status, final_rank, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "season_records":
		return m.listExportRows("season_records", "id, season_id, game_type, start_rank_score, end_rank_score, peak_rank_score, matches_played, wins, final_rank, rewards, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	case "season_challenges":
		return m.listExportRows("season_challenge_snapshots", "id, season_id, game_type, challenge_key, challenge_name, threshold, progress, completed, archived_at, created_at", "user_id = ?", []any{userID}, afterID, snapshotAt, limit, nil)
	default:
		return nil, false, fmt.Errorf("unsupported personal data export category: %s", category)
	}
}

func (m *UserDataLifecycleModel) listExportRows(table, columns, scope string, args []any, afterID int64, snapshotAt time.Time, limit int, transform func(map[string]any)) ([]PersonalDataExportRecord, bool, error) {
	var rows []map[string]any
	err := m.db.Table(table).
		Select(columns).
		Where(scope, args...).
		Where("id > ?", afterID).
		Where("created_at <= ?", snapshotAt).
		Order("id ASC").
		Limit(limit + 1).
		Find(&rows).Error
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	records := make([]PersonalDataExportRecord, 0, len(rows))
	for _, row := range rows {
		id := exportInt64(row["id"])
		if id <= 0 {
			return nil, false, errors.New("personal data export row missing id")
		}
		data := make(map[string]any, len(row)-1)
		for key, value := range row {
			if key != "id" {
				data[key] = exportValue(value)
			}
		}
		if transform != nil {
			transform(data)
		}
		records = append(records, PersonalDataExportRecord{ID: id, Data: data})
	}
	return records, hasMore, nil
}

func normalizePersonalDataExportLimit(limit int) int {
	if limit <= 0 {
		return PersonalDataExportMaxBatchSize
	}
	if limit > PersonalDataExportMaxBatchSize {
		return PersonalDataExportMaxBatchSize
	}
	return limit
}

func redactOAuthIdentifier(row map[string]any) {
	identifier := exportString(row["open_id"])
	delete(row, "open_id")
	if identifier == "" {
		row["identifier_hint"] = ""
		return
	}
	if len(identifier) <= 4 {
		row["identifier_hint"] = "****"
		return
	}
	row["identifier_hint"] = "****" + identifier[len(identifier)-4:]
}

func exportInt64(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case int32:
		return int64(typed)
	case uint64:
		return int64(typed)
	case uint:
		return int64(typed)
	case []byte:
		var result int64
		_, _ = fmt.Sscan(string(typed), &result)
		return result
	case string:
		var result int64
		_, _ = fmt.Sscan(typed, &result)
		return result
	default:
		return 0
	}
}

func exportString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func exportValue(value any) any {
	if bytes, ok := value.([]byte); ok {
		return string(bytes)
	}
	return value
}
