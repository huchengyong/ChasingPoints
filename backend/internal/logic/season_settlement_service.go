package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	seasonx "chasing_points/internal/season"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const (
	seasonRolloverNotificationType     = "season_rollover"
	seasonSettlementPhaseRecords       = "records"
	seasonSettlementPhaseChallenges    = "challenges"
	seasonSettlementPhaseTitles        = "titles"
	seasonSettlementPhaseNotifications = "notifications"
	seasonSettlementPhasePublish       = "publish"
)

type SeasonSettlementSummary struct {
	SeasonId           int64
	NextSeasonId       *int64
	SeasonRecords      int
	SeasonTitles       int
	ChallengeSnapshots int
	Notifications      int
	Skipped            bool
}

type seasonRolloverNotificationData struct {
	FromSeasonId                int64  `json:"from_season_id"`
	FromSeasonName              string `json:"from_season_name"`
	ToSeasonId                  int64  `json:"to_season_id"`
	ToSeasonName                string `json:"to_season_name"`
	ChallengeArchivedCount      int    `json:"challenge_archived_count"`
	ChallengeCompletedCount     int    `json:"challenge_completed_count"`
	SeasonTitleCount            int    `json:"season_title_count"`
	CareerAchievementsPreserved bool   `json:"career_achievements_preserved"`
	Intermission                bool   `json:"intermission"`
	Url                         string `json:"url"`
}

type seasonRolloverRealtimeNotice struct {
	UserId    int64
	PushToken string
	Title     string
	Content   string
	Data      seasonRolloverNotificationData
}

type SeasonSettlementOptions struct {
	Notify bool
}

type SeasonSettlementService struct {
	svcCtx       *svc.ServiceContext
	sendRealtime func(seasonRolloverRealtimeNotice) error
	phaseHook    func(string) error
}

func NewSeasonSettlementService(svcCtx *svc.ServiceContext) *SeasonSettlementService {
	service := &SeasonSettlementService{svcCtx: svcCtx}
	service.sendRealtime = service.defaultSendRealtime
	return service
}

func (s *SeasonSettlementService) SettleSeason(ctx context.Context, seasonId int64) (*SeasonSettlementSummary, error) {
	return s.SettleSeasonAt(ctx, seasonId, time.Now())
}

func (s *SeasonSettlementService) SettleSeasonAt(ctx context.Context, seasonId int64, now time.Time) (*SeasonSettlementSummary, error) {
	return s.SettleSeasonWithOptionsAt(ctx, seasonId, now, SeasonSettlementOptions{Notify: true})
}

func (s *SeasonSettlementService) SettleSeasonWithOptionsAt(ctx context.Context, seasonId int64, now time.Time, options SeasonSettlementOptions) (*SeasonSettlementSummary, error) {
	if now.IsZero() {
		now = time.Now()
	}
	if err := s.validate(); err != nil {
		return nil, err
	}

	summary := &SeasonSettlementSummary{SeasonId: seasonId}
	skipped, nextSeasonID, err := s.beginSettlementAttempt(seasonId, now)
	if err != nil {
		return nil, err
	}
	if skipped {
		summary.Skipped = true
		summary.NextSeasonId = nextSeasonID
		return summary, nil
	}
	fail := func(cause error) (*SeasonSettlementSummary, error) {
		s.markSettlementFailed(seasonId, now, cause)
		return nil, cause
	}

	records, err := s.finalizeSeasonRecords(seasonId, now)
	if err != nil {
		return fail(err)
	}
	if err := s.runPhaseHook(seasonSettlementPhaseRecords); err != nil {
		return fail(err)
	}
	snapshots, err := s.archiveSeasonChallenges(seasonId, now)
	if err != nil {
		return fail(err)
	}
	if err := s.runPhaseHook(seasonSettlementPhaseChallenges); err != nil {
		return fail(err)
	}
	seasonTitles, err := s.grantSeasonTitles(seasonId, now)
	if err != nil {
		return fail(err)
	}
	if err := s.runPhaseHook(seasonSettlementPhaseTitles); err != nil {
		return fail(err)
	}
	notificationCount, realtimeNotices, err := s.persistRolloverNotificationPhase(seasonId, now, options.Notify)
	if err != nil {
		return fail(err)
	}
	if err := s.runPhaseHook(seasonSettlementPhaseNotifications); err != nil {
		return fail(err)
	}
	nextSeasonID, err = s.publishSeasonSettlement(seasonId, now)
	if err != nil {
		return fail(err)
	}
	invalidateSeasonInfoCache(ctx, s.svcCtx)

	summary.NextSeasonId = nextSeasonID
	summary.SeasonRecords = len(records)
	summary.SeasonTitles = seasonTitles
	summary.ChallengeSnapshots = len(snapshots)
	summary.Notifications = notificationCount
	for _, notice := range realtimeNotices {
		if s.sendRealtime != nil {
			if err := s.sendRealtime(notice); err != nil {
				logx.WithContext(ctx).Errorf("发送换季实时提醒失败: userId=%d err=%v", notice.UserId, err)
			}
		}
	}
	return summary, nil
}

func (s *SeasonSettlementService) beginSettlementAttempt(seasonID int64, now time.Time) (bool, *int64, error) {
	var skipped bool
	var nextSeasonID *int64
	err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		settlement, err := s.svcCtx.SeasonSettlementModel.FindOrCreateForUpdateWithTx(tx, seasonID)
		if err != nil {
			return err
		}
		if settlement.Status == model.SeasonSettlementStatusCompleted {
			skipped = true
			nextSeasonID = settlement.NextSeasonId
			return nil
		}
		season, err := s.svcCtx.SeasonModel.FindByIdWithTx(tx, seasonID, true)
		if err != nil {
			return err
		}
		if season == nil {
			return fmt.Errorf("season %d not found", seasonID)
		}
		_, endExclusive, err := seasonx.BoundsForConfig(s.svcCtx.Config.SeasonLifecycle, season)
		if err != nil {
			return err
		}
		if now.Before(endExclusive) {
			return fmt.Errorf("season %d has not ended", seasonID)
		}
		settlement.Status = model.SeasonSettlementStatusRunning
		settlement.Attempts++
		settlement.StartedAt = &now
		settlement.CompletedAt = nil
		settlement.LastError = ""
		return s.svcCtx.SeasonSettlementModel.SaveWithTx(tx, settlement)
	})
	return skipped, nextSeasonID, err
}

func (s *SeasonSettlementService) finalizeSeasonRecords(seasonID int64, now time.Time) ([]model.SeasonRecord, error) {
	var records []model.SeasonRecord
	err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		settlement, err := s.svcCtx.SeasonSettlementModel.FindBySeasonIdWithTx(tx, seasonID, true)
		if err != nil || settlement == nil {
			return err
		}
		season, err := s.svcCtx.SeasonModel.FindByIdWithTx(tx, seasonID, false)
		if err != nil {
			return err
		}
		if season == nil {
			return fmt.Errorf("season %d not found", seasonID)
		}
		records, err = s.buildSeasonRecordsWithTx(tx, season)
		if err != nil || settlement.RecordsCompletedAt != nil {
			return err
		}
		if err := s.svcCtx.SeasonRecordModel.UpsertBatchWithTx(tx, records); err != nil {
			return err
		}
		settlement.RecordsCompletedAt = &now
		return s.svcCtx.SeasonSettlementModel.SaveWithTx(tx, settlement)
	})
	return records, err
}

func (s *SeasonSettlementService) archiveSeasonChallenges(seasonID int64, now time.Time) ([]model.SeasonChallengeSnapshot, error) {
	var snapshots []model.SeasonChallengeSnapshot
	err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		settlement, err := s.svcCtx.SeasonSettlementModel.FindBySeasonIdWithTx(tx, seasonID, true)
		if err != nil || settlement == nil {
			return err
		}
		if settlement.ChallengesCompletedAt != nil {
			snapshots, err = s.svcCtx.SeasonChallengeSnapshotModel.ListBySeasonWithTx(tx, seasonID)
			return err
		}
		season, err := s.svcCtx.SeasonModel.FindByIdWithTx(tx, seasonID, false)
		if err != nil {
			return err
		}
		if season == nil {
			return fmt.Errorf("season %d not found", seasonID)
		}
		snapshots, err = achievementx.NewSeasonChallengeService(s.svcCtx).ArchiveSeasonWithTx(tx, season, now)
		if err != nil {
			return err
		}
		settlement.ChallengesCompletedAt = &now
		return s.svcCtx.SeasonSettlementModel.SaveWithTx(tx, settlement)
	})
	return snapshots, err
}

func (s *SeasonSettlementService) grantSeasonTitles(seasonID int64, now time.Time) (int, error) {
	count := 0
	err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		settlement, err := s.svcCtx.SeasonSettlementModel.FindBySeasonIdWithTx(tx, seasonID, true)
		if err != nil || settlement == nil {
			return err
		}
		if settlement.TitlesCompletedAt != nil {
			var existing int64
			if err := tx.Model(&model.UserTitle{}).Where("source_type = ? AND source_ref_id = ?", achievementx.SourceTypeSeason, seasonID).Count(&existing).Error; err != nil {
				return err
			}
			count = int(existing)
			return nil
		}
		season, err := s.svcCtx.SeasonModel.FindByIdWithTx(tx, seasonID, false)
		if err != nil {
			return err
		}
		if season == nil {
			return fmt.Errorf("season %d not found", seasonID)
		}
		records, err := s.buildSeasonRecordsWithTx(tx, season)
		if err != nil {
			return err
		}
		count, err = achievementx.GrantSeasonTitlesWithTx(s.svcCtx, tx, []model.Season{*season}, records)
		if err != nil {
			return err
		}
		settlement.TitlesCompletedAt = &now
		return s.svcCtx.SeasonSettlementModel.SaveWithTx(tx, settlement)
	})
	return count, err
}

func (s *SeasonSettlementService) persistRolloverNotificationPhase(seasonID int64, now time.Time, notify bool) (int, []seasonRolloverRealtimeNotice, error) {
	count := 0
	var notices []seasonRolloverRealtimeNotice
	err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		settlement, err := s.svcCtx.SeasonSettlementModel.FindBySeasonIdWithTx(tx, seasonID, true)
		if err != nil || settlement == nil {
			return err
		}
		if settlement.NotificationsCompletedAt != nil {
			if !notify {
				return nil
			}
			toSeasonID := int64(0)
			if settlement.NextSeasonId != nil {
				toSeasonID = *settlement.NextSeasonId
			}
			var total int64
			err := tx.Model(&model.Notification{}).
				Where("type = ? AND dedupe_key = ?", seasonRolloverNotificationType, seasonRolloverDedupeKey(seasonID, toSeasonID)).
				Count(&total).Error
			count = int(total)
			return err
		}
		season, err := s.svcCtx.SeasonModel.FindByIdWithTx(tx, seasonID, false)
		if err != nil {
			return err
		}
		if season == nil {
			return fmt.Errorf("season %d not found", seasonID)
		}
		nextSeason, err := s.findNextSeasonCandidateWithTx(tx, season, now)
		if err != nil {
			return err
		}
		if nextSeason != nil {
			nextID := nextSeason.Id
			settlement.NextSeasonId = &nextID
		} else {
			settlement.NextSeasonId = nil
		}
		if notify {
			records, err := s.buildSeasonRecordsWithTx(tx, season)
			if err != nil {
				return err
			}
			snapshots, err := s.svcCtx.SeasonChallengeSnapshotModel.ListBySeasonWithTx(tx, seasonID)
			if err != nil {
				return err
			}
			notices, err = s.persistRolloverNotificationsWithTx(tx, *season, nextSeason, records, snapshots)
			if err != nil {
				return err
			}
			toSeasonID := int64(0)
			if nextSeason != nil {
				toSeasonID = nextSeason.Id
			}
			var total int64
			if err := tx.Model(&model.Notification{}).
				Where("type = ? AND dedupe_key = ?", seasonRolloverNotificationType, seasonRolloverDedupeKey(seasonID, toSeasonID)).
				Count(&total).Error; err != nil {
				return err
			}
			count = int(total)
		}
		settlement.NotificationsCompletedAt = &now
		return s.svcCtx.SeasonSettlementModel.SaveWithTx(tx, settlement)
	})
	return count, notices, err
}

func (s *SeasonSettlementService) findNextSeasonCandidateWithTx(tx *gorm.DB, settled *model.Season, now time.Time) (*model.Season, error) {
	if settled == nil {
		return nil, nil
	}
	next, err := s.svcCtx.SeasonModel.FindNextAfterStartWithTx(tx, settled.StartDate, false)
	if err != nil || next == nil {
		return next, err
	}
	startAt, _, err := seasonx.BoundsForConfig(s.svcCtx.Config.SeasonLifecycle, next)
	if err != nil || now.Before(startAt) {
		return nil, err
	}
	return next, nil
}

func (s *SeasonSettlementService) publishSeasonSettlement(seasonID int64, now time.Time) (*int64, error) {
	var nextSeasonID *int64
	err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		settlement, err := s.svcCtx.SeasonSettlementModel.FindBySeasonIdWithTx(tx, seasonID, true)
		if err != nil || settlement == nil {
			return err
		}
		if settlement.Status == model.SeasonSettlementStatusCompleted {
			nextSeasonID = settlement.NextSeasonId
			return nil
		}
		if settlement.RecordsCompletedAt == nil || settlement.ChallengesCompletedAt == nil || settlement.TitlesCompletedAt == nil || settlement.NotificationsCompletedAt == nil {
			return fmt.Errorf("season %d settlement phases are incomplete", seasonID)
		}
		season, err := s.svcCtx.SeasonModel.FindByIdWithTx(tx, seasonID, true)
		if err != nil {
			return err
		}
		if season == nil {
			return fmt.Errorf("season %d not found", seasonID)
		}
		if _, err := s.svcCtx.SeasonModel.UpdateStatusToWithTx(tx, season.Id, 2); err != nil {
			return err
		}
		season.Status = 2
		nextSeason, err := s.resolveNextSeasonWithTx(tx, season, now)
		if err != nil {
			return err
		}
		if err := s.runPhaseHook(seasonSettlementPhasePublish); err != nil {
			return err
		}
		settlement.Status = model.SeasonSettlementStatusCompleted
		settlement.CompletedAt = &now
		settlement.LastError = ""
		if nextSeason != nil {
			nextID := nextSeason.Id
			settlement.NextSeasonId = &nextID
			nextSeasonID = &nextID
		} else {
			settlement.NextSeasonId = nil
			nextSeasonID = nil
		}
		return s.svcCtx.SeasonSettlementModel.SaveWithTx(tx, settlement)
	})
	return nextSeasonID, err
}

func (s *SeasonSettlementService) runPhaseHook(phase string) error {
	if s == nil || s.phaseHook == nil {
		return nil
	}
	return s.phaseHook(phase)
}

func (s *SeasonSettlementService) ActivateReadySeason(ctx context.Context) (*model.Season, error) {
	return s.ActivateReadySeasonAt(ctx, time.Now())
}

func (s *SeasonSettlementService) ActivateReadySeasonAt(_ context.Context, now time.Time) (*model.Season, error) {
	if now.IsZero() {
		now = time.Now()
	}
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil || s.svcCtx.SeasonModel == nil {
		return nil, fmt.Errorf("season activation service is unavailable")
	}
	var activated *model.Season
	err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		activated, err = s.activateReadySeasonWithTx(tx, now, time.Time{})
		return err
	})
	return activated, err
}

func (s *SeasonSettlementService) validate() error {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil ||
		s.svcCtx.SeasonModel == nil || s.svcCtx.SeasonRecordModel == nil ||
		s.svcCtx.UserTitleModel == nil || s.svcCtx.AchievementProgressEventModel == nil ||
		s.svcCtx.SeasonChallengeSnapshotModel == nil || s.svcCtx.SeasonSettlementModel == nil ||
		s.svcCtx.NotificationModel == nil {
		return fmt.Errorf("season settlement service is unavailable")
	}
	return nil
}

func (s *SeasonSettlementService) buildSeasonRecordsWithTx(tx *gorm.DB, season *model.Season) ([]model.SeasonRecord, error) {
	records, err := s.svcCtx.SeasonRecordModel.ListForSettlementWithTx(tx, season.Id)
	if err != nil {
		return nil, err
	}
	rankByGameType := make(map[int]int)
	for i := range records {
		rankByGameType[records[i].GameType]++
		records[i].FinalRank = rankByGameType[records[i].GameType]
	}
	return records, nil
}

func (s *SeasonSettlementService) resolveNextSeasonWithTx(tx *gorm.DB, settled *model.Season, now time.Time) (*model.Season, error) {
	if settled == nil {
		return nil, nil
	}
	next, err := s.svcCtx.SeasonModel.FindNextAfterStartWithTx(tx, settled.StartDate, true)
	if err != nil || next == nil {
		return next, err
	}
	startAt, _, err := seasonx.BoundsForConfig(s.svcCtx.Config.SeasonLifecycle, next)
	if err != nil || now.Before(startAt) {
		return nil, err
	}
	if next.Status != 0 {
		return next, nil
	}
	current, err := s.svcCtx.SeasonModel.FindCurrentWithTx(tx, true)
	if err != nil || current != nil {
		return next, err
	}
	activated, err := s.svcCtx.SeasonModel.UpdateStatusWithTx(tx, next.Id, 0, 1)
	if err != nil || !activated {
		return nil, err
	}
	next.Status = 1
	return next, nil
}

func (s *SeasonSettlementService) activateReadySeasonWithTx(tx *gorm.DB, now time.Time, after time.Time) (*model.Season, error) {
	current, err := s.svcCtx.SeasonModel.FindCurrentWithTx(tx, true)
	if err != nil || current != nil {
		return nil, err
	}
	candidate, err := s.svcCtx.SeasonModel.FindReadyUpcomingWithTx(tx, now, after, true)
	if err != nil || candidate == nil {
		return candidate, err
	}
	activated, err := s.svcCtx.SeasonModel.UpdateStatusWithTx(tx, candidate.Id, 0, 1)
	if err != nil || !activated {
		return nil, err
	}
	candidate.Status = 1
	return candidate, nil
}

func (s *SeasonSettlementService) persistRolloverNotificationsWithTx(
	tx *gorm.DB,
	from model.Season,
	to *model.Season,
	records []model.SeasonRecord,
	snapshots []model.SeasonChallengeSnapshot,
) ([]seasonRolloverRealtimeNotice, error) {
	challengeArchivedByUser := make(map[int64]int)
	challengeCompletedByUser := make(map[int64]int)
	for _, snapshot := range snapshots {
		challengeArchivedByUser[snapshot.UserId]++
		if snapshot.Completed == 1 {
			challengeCompletedByUser[snapshot.UserId]++
		}
	}
	titleCountByUser := make(map[int64]int)
	userSet := make(map[int64]struct{})
	for _, record := range records {
		userSet[record.UserId] = struct{}{}
		if record.FinalRank >= 1 && record.FinalRank <= 10 {
			titleCountByUser[record.UserId]++
		}
	}
	for _, snapshot := range snapshots {
		userSet[snapshot.UserId] = struct{}{}
	}
	userIds := make([]int64, 0, len(userSet))
	for userId := range userSet {
		userIds = append(userIds, userId)
	}
	sort.Slice(userIds, func(i, j int) bool { return userIds[i] < userIds[j] })

	toId := int64(0)
	toName := ""
	if to != nil {
		toId = to.Id
		toName = to.Name
	}
	dedupe := seasonRolloverDedupeKey(from.Id, toId)
	title := "赛季结算完成"
	content := fmt.Sprintf("%s 已结束并归档，生涯成就与历史荣誉已保留。", from.Name)
	if to != nil {
		content = fmt.Sprintf("%s 已归档，%s 已开启，生涯成就保持不变。", from.Name, to.Name)
	}
	users := make(map[int64]model.User)
	if s.svcCtx.UserModel != nil {
		var err error
		users, err = s.svcCtx.UserModel.FindByIdsWithTx(tx, userIds)
		if err != nil {
			return nil, err
		}
	}
	notices := make([]seasonRolloverRealtimeNotice, 0, len(userIds))
	notifications := make([]model.Notification, 0, len(userIds))
	for _, userId := range userIds {
		payload := seasonRolloverNotificationData{
			FromSeasonId:                from.Id,
			FromSeasonName:              from.Name,
			ToSeasonId:                  toId,
			ToSeasonName:                toName,
			ChallengeArchivedCount:      challengeArchivedByUser[userId],
			ChallengeCompletedCount:     challengeCompletedByUser[userId],
			SeasonTitleCount:            titleCountByUser[userId],
			CareerAchievementsPreserved: true,
			Intermission:                to == nil,
			Url:                         "/subPages/achievement/index?tab=season",
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		data := string(raw)
		dedupeKey := dedupe
		notifications = append(notifications, model.Notification{
			UserId: userId, Type: seasonRolloverNotificationType, DedupeKey: &dedupeKey,
			Title: title, Content: content, Data: &data, IsRead: 0,
		})
		notices = append(notices, seasonRolloverRealtimeNotice{
			UserId: userId, PushToken: users[userId].PushToken, Title: title, Content: content, Data: payload,
		})
	}
	if _, err := s.svcCtx.NotificationModel.CreateBatchIfAbsentWithTx(tx, notifications, 500); err != nil {
		return nil, err
	}
	return notices, nil
}

func (s *SeasonSettlementService) markSettlementFailed(seasonId int64, now time.Time, cause error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil || s.svcCtx.SeasonSettlementModel == nil {
		return
	}
	_ = s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		settlement, err := s.svcCtx.SeasonSettlementModel.FindBySeasonIdWithTx(tx, seasonId, true)
		if err != nil || settlement == nil || settlement.Status == model.SeasonSettlementStatusCompleted {
			return err
		}
		settlement.Status = model.SeasonSettlementStatusFailed
		settlement.CompletedAt = nil
		settlement.LastError = truncateSeasonSettlementError(cause.Error())
		return s.svcCtx.SeasonSettlementModel.SaveWithTx(tx, settlement)
	})
}

func (s *SeasonSettlementService) defaultSendRealtime(notice seasonRolloverRealtimeNotice) error {
	if s == nil || s.svcCtx == nil {
		return nil
	}
	if notice.PushToken != "" && s.svcCtx.PushService != nil {
		s.svcCtx.PushService.SendPush(notice.PushToken, notice.Title, notice.Content, map[string]interface{}{
			"url": notice.Data.Url,
		})
	}
	if ws.GlobalHub != nil {
		ws.GlobalHub.SendToUser(notice.UserId, &ws.Message{
			Type: "notification_update",
			Data: map[string]interface{}{"category": seasonRolloverNotificationType},
		})
	}
	return nil
}

func seasonRolloverDedupeKey(fromSeasonId, toSeasonId int64) string {
	return fmt.Sprintf("season_rollover:%d:%d", fromSeasonId, toSeasonId)
}

func truncateSeasonSettlementError(message string) string {
	if len(message) <= 500 {
		return message
	}
	return message[:500]
}
