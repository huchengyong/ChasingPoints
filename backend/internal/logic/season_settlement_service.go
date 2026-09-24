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
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const seasonRolloverNotificationType = "season_rollover"

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
	UserId  int64
	Title   string
	Content string
	Data    seasonRolloverNotificationData
}

type SeasonSettlementService struct {
	svcCtx       *svc.ServiceContext
	sendRealtime func(seasonRolloverRealtimeNotice) error
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
	if now.IsZero() {
		now = time.Now()
	}
	if err := s.validate(); err != nil {
		return nil, err
	}

	summary := &SeasonSettlementSummary{SeasonId: seasonId}
	realtimeNotices := make([]seasonRolloverRealtimeNotice, 0)
	err := s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		settlement, err := s.svcCtx.SeasonSettlementModel.FindOrCreateForUpdateWithTx(tx, seasonId)
		if err != nil {
			return err
		}
		if settlement.Status == model.SeasonSettlementStatusCompleted {
			summary.Skipped = true
			summary.NextSeasonId = settlement.NextSeasonId
			return nil
		}

		season, err := s.svcCtx.SeasonModel.FindByIdWithTx(tx, seasonId, true)
		if err != nil {
			return err
		}
		if season == nil {
			return fmt.Errorf("season %d not found", seasonId)
		}
		if now.Before(seasonSettlementEndExclusive(season.EndDate)) {
			return fmt.Errorf("season %d has not ended", seasonId)
		}

		settlement.Status = model.SeasonSettlementStatusRunning
		settlement.Attempts++
		settlement.StartedAt = &now
		settlement.CompletedAt = nil
		settlement.LastError = ""
		if err := s.svcCtx.SeasonSettlementModel.SaveWithTx(tx, settlement); err != nil {
			return err
		}

		records, err := s.buildSeasonRecordsWithTx(tx, season)
		if err != nil {
			return err
		}
		if err := s.svcCtx.SeasonRecordModel.UpsertBatchWithTx(tx, records); err != nil {
			return err
		}
		if _, err := achievementx.GrantSeasonTitlesWithTx(s.svcCtx, tx, []model.Season{*season}, records); err != nil {
			return err
		}
		snapshots, err := achievementx.NewSeasonChallengeService(s.svcCtx).ArchiveSeasonWithTx(tx, season, now)
		if err != nil {
			return err
		}

		if season.Status == 1 {
			if _, err := s.svcCtx.SeasonModel.UpdateStatusWithTx(tx, season.Id, 1, 2); err != nil {
				return err
			}
		}
		nextSeason, err := s.activateReadySeasonWithTx(tx, now, season.StartDate)
		if err != nil {
			return err
		}

		notices, err := s.persistRolloverNotificationsWithTx(tx, *season, nextSeason, records, snapshots)
		if err != nil {
			return err
		}
		realtimeNotices = notices

		settlement.Status = model.SeasonSettlementStatusCompleted
		settlement.CompletedAt = &now
		settlement.LastError = ""
		if nextSeason != nil {
			nextId := nextSeason.Id
			settlement.NextSeasonId = &nextId
			summary.NextSeasonId = &nextId
		} else {
			settlement.NextSeasonId = nil
		}
		if err := s.svcCtx.SeasonSettlementModel.SaveWithTx(tx, settlement); err != nil {
			return err
		}

		summary.SeasonRecords = len(records)
		summary.SeasonTitles = countSeasonTitleRecords(records)
		summary.ChallengeSnapshots = len(snapshots)
		summary.Notifications = len(notices)
		return nil
	})
	if err != nil {
		s.markSettlementFailed(seasonId, now, err)
		return nil, err
	}

	for _, notice := range realtimeNotices {
		if s.sendRealtime != nil {
			if err := s.sendRealtime(notice); err != nil {
				logx.WithContext(ctx).Errorf("发送换季实时提醒失败: userId=%d err=%v", notice.UserId, err)
			}
		}
	}
	return summary, nil
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
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil || s.svcCtx.MatchModel == nil ||
		s.svcCtx.RankingModel == nil || s.svcCtx.SeasonModel == nil || s.svcCtx.SeasonRecordModel == nil ||
		s.svcCtx.UserTitleModel == nil || s.svcCtx.AchievementProgressEventModel == nil ||
		s.svcCtx.SeasonChallengeSnapshotModel == nil || s.svcCtx.SeasonSettlementModel == nil ||
		s.svcCtx.NotificationModel == nil {
		return fmt.Errorf("season settlement service is unavailable")
	}
	return nil
}

func (s *SeasonSettlementService) buildSeasonRecordsWithTx(tx *gorm.DB, season *model.Season) ([]model.SeasonRecord, error) {
	endExclusive := seasonSettlementEndExclusive(season.EndDate)
	matches, err := s.svcCtx.MatchModel.ListCompletedRankedBetweenWithTx(tx, season.StartDate, endExclusive)
	if err != nil {
		return nil, err
	}
	stats := make(map[rankUserGameKey]*seasonUserGameStats)
	for i := range matches {
		match := &matches[i]
		player1 := rankUserGameKey{UserId: match.UserId, GameType: match.GameType}
		player2 := rankUserGameKey{UserId: *match.OpponentId, GameType: match.GameType}
		if stats[player1] == nil {
			stats[player1] = &seasonUserGameStats{}
		}
		if stats[player2] == nil {
			stats[player2] = &seasonUserGameStats{}
		}
		stats[player1].MatchesPlayed++
		stats[player2].MatchesPlayed++
		if *match.Result == 1 {
			stats[player1].Wins++
		} else {
			stats[player2].Wins++
		}
	}

	records := make([]model.SeasonRecord, 0, len(stats))
	logEnd := endExclusive.Add(-time.Nanosecond)
	for pair, item := range stats {
		logs, err := s.svcCtx.RankingModel.ListRankChangesByUserAndGameTypeBetweenWithTx(tx, pair.UserId, pair.GameType, season.StartDate, logEnd)
		if err != nil {
			return nil, err
		}
		before, err := s.svcCtx.RankingModel.FindLatestRankChangeBeforeByGameTypeWithTx(tx, pair.UserId, pair.GameType, season.StartDate)
		if err != nil {
			return nil, err
		}
		startScore, endScore, peakScore := BuildSeasonSnapshotFromLogs(before, logs)
		records = append(records, model.SeasonRecord{
			SeasonId:       season.Id,
			UserId:         pair.UserId,
			GameType:       pair.GameType,
			StartRankScore: startScore,
			EndRankScore:   endScore,
			PeakRankScore:  peakScore,
			MatchesPlayed:  item.MatchesPlayed,
			Wins:           item.Wins,
		})
	}

	sort.SliceStable(records, func(i, j int) bool {
		if records[i].GameType != records[j].GameType {
			return records[i].GameType < records[j].GameType
		}
		if records[i].EndRankScore != records[j].EndRankScore {
			return records[i].EndRankScore > records[j].EndRankScore
		}
		if records[i].Wins != records[j].Wins {
			return records[i].Wins > records[j].Wins
		}
		return records[i].UserId < records[j].UserId
	})
	rankByGameType := make(map[int]int)
	for i := range records {
		rankByGameType[records[i].GameType]++
		records[i].FinalRank = rankByGameType[records[i].GameType]
	}
	return records, nil
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
	notices := make([]seasonRolloverRealtimeNotice, 0, len(userIds))
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
		created, err := s.svcCtx.NotificationModel.CreateIfAbsentWithTx(tx, &model.Notification{
			UserId:    userId,
			Type:      seasonRolloverNotificationType,
			DedupeKey: &dedupeKey,
			Title:     title,
			Content:   content,
			Data:      &data,
			IsRead:    0,
		})
		if err != nil {
			return nil, err
		}
		if created {
			notices = append(notices, seasonRolloverRealtimeNotice{UserId: userId, Title: title, Content: content, Data: payload})
		}
	}
	return notices, nil
}

func (s *SeasonSettlementService) markSettlementFailed(seasonId int64, now time.Time, cause error) {
	if s == nil || s.svcCtx == nil || s.svcCtx.DB == nil || s.svcCtx.SeasonSettlementModel == nil {
		return
	}
	_ = s.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		settlement, err := s.svcCtx.SeasonSettlementModel.FindOrCreateForUpdateWithTx(tx, seasonId)
		if err != nil || settlement.Status == model.SeasonSettlementStatusCompleted {
			return err
		}
		settlement.Status = model.SeasonSettlementStatusFailed
		settlement.Attempts++
		settlement.StartedAt = &now
		settlement.CompletedAt = nil
		settlement.LastError = truncateSeasonSettlementError(cause.Error())
		return s.svcCtx.SeasonSettlementModel.SaveWithTx(tx, settlement)
	})
}

func (s *SeasonSettlementService) defaultSendRealtime(notice seasonRolloverRealtimeNotice) error {
	if s == nil || s.svcCtx == nil {
		return nil
	}
	if s.svcCtx.UserModel != nil {
		user, err := s.svcCtx.UserModel.FindById(notice.UserId)
		if err != nil {
			return err
		}
		if user != nil && user.PushToken != "" && s.svcCtx.PushService != nil {
			s.svcCtx.PushService.SendPush(user.PushToken, notice.Title, notice.Content, map[string]interface{}{
				"url": notice.Data.Url,
			})
		}
	}
	if ws.GlobalHub != nil {
		ws.GlobalHub.SendToUser(notice.UserId, &ws.Message{
			Type: "notification_update",
			Data: map[string]interface{}{"category": seasonRolloverNotificationType},
		})
	}
	return nil
}

func seasonSettlementEndExclusive(endDate time.Time) time.Time {
	return time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location()).AddDate(0, 0, 1)
}

func seasonRolloverDedupeKey(fromSeasonId, toSeasonId int64) string {
	return fmt.Sprintf("season_rollover:%d:%d", fromSeasonId, toSeasonId)
}

func countSeasonTitleRecords(records []model.SeasonRecord) int {
	count := 0
	for _, record := range records {
		if record.FinalRank >= 1 && record.FinalRank <= 10 {
			count++
		}
	}
	return count
}

func truncateSeasonSettlementError(message string) string {
	if len(message) <= 500 {
		return message
	}
	return message[:500]
}
