package match

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	publiclogic "chasing_points/internal/logic/public"
	seasonlogic "chasing_points/internal/logic/season"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/types"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func TestRankedFinishProjectsCompetitiveReadModelsExactlyOnce(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(
		&model.MatchParticipantResult{},
		&model.UserCompetitiveStats{},
		&model.UserOpponentStats{},
		&model.UserOpponentStrengthBucket{},
		&model.UserNotificationPreference{},
	); err != nil {
		t.Fatalf("prepare competitive read schema: %v", err)
	}
	svcCtx.CompetitiveReadModel = model.NewCompetitiveReadModel(svcCtx.DB)
	svcCtx.UserNotificationPreferenceModel = model.NewUserNotificationPreferenceModel(svcCtx.DB)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: 2002, Nickname: "选手乙"},
	)
	for _, userID := range []int64{1001, 2002} {
		if err := svcCtx.UserNotificationPreferenceModel.Upsert(&model.UserNotificationPreference{
			UserId: userID, MatchResultEnabled: false,
		}); err != nil {
			t.Fatalf("disable match result notification: %v", err)
		}
	}
	previousHub := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = previousHub })

	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9101, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic,
		Status: 1, MyScore: 5, OpponentScore: 3, CurrentFrameStarted: true,
		MatchTime: time.Now().Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	req := &types.FinishMatchReq{MatchId: 9101, ClientActionId: "competitive-finish-9101", BaseRevision: 0}
	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(req)
	if err != nil || resp == nil || !resp.Success {
		t.Fatalf("finish match: resp=%#v err=%v", resp, err)
	}
	stored, err := svcCtx.MatchModel.FindById(9101)
	if err != nil || stored == nil || stored.Status != 2 || stored.CompletedAt == nil {
		t.Fatalf("completed match must persist completed_at: match=%+v err=%v", stored, err)
	}
	userDataEvents := map[int64]struct {
		CompetitiveRevision int64    `json:"competitive_revision"`
		Scopes              []string `json:"scopes"`
	}{}
	for i := 0; i < 4; i++ {
		message := <-hub.SendUser
		var payload struct {
			Type string `json:"type"`
			Data struct {
				CompetitiveRevision int64    `json:"competitive_revision"`
				Scopes              []string `json:"scopes"`
			} `json:"data"`
		}
		if err := json.Unmarshal(message.Message, &payload); err != nil {
			t.Fatalf("decode user event: %v", err)
		}
		if payload.Type == "user_data_updated" {
			userDataEvents[message.UserId] = payload.Data
		}
	}
	if len(userDataEvents) != 2 {
		t.Fatalf("closed match-result notification must not suppress competitive events: %+v", userDataEvents)
	}
	for userID, event := range userDataEvents {
		if event.CompetitiveRevision != 1 || !containsScope(event.Scopes, "rank") || !containsScope(event.Scopes, "stats") {
			t.Fatalf("unexpected competitive event for user %d: %+v", userID, event)
		}
	}
	for _, userID := range []int64{1001, 2002} {
		participant, findErr := svcCtx.CompetitiveReadModel.FindParticipantByMatchAndUser(9101, userID)
		if findErr != nil || participant == nil {
			t.Fatalf("missing user %d participant projection: result=%+v err=%v", userID, participant, findErr)
		}
		stats, statsErr := svcCtx.CompetitiveReadModel.FindStats(userID, 0)
		if statsErr != nil || stats == nil || stats.TotalMatches != 1 || stats.Revision != 1 {
			t.Fatalf("unexpected user %d competitive stats: stats=%+v err=%v", userID, stats, statsErr)
		}
	}

	replay, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(req)
	if err != nil || replay == nil || !replay.Success {
		t.Fatalf("replay finish: resp=%#v err=%v", replay, err)
	}
	stats, err := svcCtx.CompetitiveReadModel.FindStats(1001, 0)
	if err != nil || stats == nil || stats.TotalMatches != 1 || stats.Revision != 1 {
		t.Fatalf("replay must not double count competitive stats: stats=%+v err=%v", stats, err)
	}
	var finishActionCount int64
	if err := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", 9101, "match_end").Count(&finishActionCount).Error; err != nil || finishActionCount != 1 {
		t.Fatalf("replay must retain one finish action: count=%d err=%v", finishActionCount, err)
	}
}

func TestRankedFinishMaintainsCompetitiveProjectionAcrossCompletionSources(t *testing.T) {
	tests := []struct {
		name             string
		actorID          int64
		refereeID        int64
		confirmed        bool
		expectedSource   string
		expectedCloserID int64
	}{
		{name: "creator direct", actorID: 1001, expectedSource: model.CompletionSourcePlayerDirect, expectedCloserID: 1001},
		{name: "opponent direct", actorID: 2002, expectedSource: model.CompletionSourcePlayerDirect, expectedCloserID: 2002},
		{name: "referee direct", actorID: 3003, refereeID: 3003, expectedSource: model.CompletionSourceReferee, expectedCloserID: 3003},
		{name: "opponent confirms creator request", actorID: 2002, confirmed: true, expectedSource: model.CompletionSourcePlayerConfirmed, expectedCloserID: 2002},
	}
	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcCtx := newFinishMatchReputationTestSvc(t)
			if err := svcCtx.DB.AutoMigrate(&model.MatchParticipantResult{}, &model.UserCompetitiveStats{}, &model.UserOpponentStats{}, &model.UserOpponentStrengthBucket{}); err != nil {
				t.Fatalf("prepare competitive read schema: %v", err)
			}
			svcCtx.CompetitiveReadModel = model.NewCompetitiveReadModel(svcCtx.DB)
			seedFinishReputationUsers(t, svcCtx,
				model.User{Id: 1001, Nickname: "选手甲"},
				model.User{Id: 2002, Nickname: "选手乙"},
				model.User{Id: 3003, Nickname: "裁判"},
			)
			matchID := int64(9200 + index)
			opponentID := int64(2002)
			match := model.Match{
				Id: matchID, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
				MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic,
				Status: 1, MyScore: 5, OpponentScore: 3, CurrentFrameStarted: true,
				FinishConfirmationRequired: tt.confirmed, MatchTime: time.Now().Add(-10 * time.Minute),
			}
			if tt.refereeID > 0 {
				match.RefereeUserId = &tt.refereeID
			}
			if err := svcCtx.MatchModel.Create(&match); err != nil {
				t.Fatalf("create match: %v", err)
			}

			if tt.confirmed {
				request, err := NewRequestFinishMatchLogic(finishReputationCtx(1001), svcCtx).RequestFinishMatch(&types.FinishMatchActionReq{MatchId: matchID, ClientActionId: "request-finish", BaseRevision: 0})
				if err != nil || !request.Success {
					t.Fatalf("request finish: resp=%#v err=%v", request, err)
				}
				confirm, err := NewConfirmFinishMatchLogic(finishReputationCtx(tt.actorID), svcCtx).ConfirmFinishMatch(&types.FinishMatchActionReq{MatchId: matchID, ClientActionId: "confirm-finish", BaseRevision: request.ServerRevision})
				if err != nil || !confirm.Success {
					t.Fatalf("confirm finish: resp=%#v err=%v", confirm, err)
				}
			} else {
				resp, err := NewFinishMatchLogic(finishReputationCtx(tt.actorID), svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: matchID, ClientActionId: "direct-finish", BaseRevision: 0})
				if err != nil || !resp.Success {
					t.Fatalf("finish match: resp=%#v err=%v", resp, err)
				}
			}

			stored, err := svcCtx.MatchModel.FindById(matchID)
			if err != nil || stored == nil || stored.Status != 2 || stored.CompletedAt == nil || stored.CompletedByUserId == nil || *stored.CompletedByUserId != tt.expectedCloserID || stored.CompletionSource != tt.expectedSource {
				t.Fatalf("unexpected completion attribution: match=%+v err=%v", stored, err)
			}
			for _, userID := range []int64{1001, 2002} {
				participant, err := svcCtx.CompetitiveReadModel.FindParticipantByMatchAndUser(matchID, userID)
				if err != nil || participant == nil {
					t.Fatalf("missing participant projection for user %d: row=%+v err=%v", userID, participant, err)
				}
				stats, err := svcCtx.CompetitiveReadModel.FindStats(userID, 0)
				if err != nil || stats == nil || stats.TotalMatches != 1 || stats.Revision != 1 {
					t.Fatalf("unexpected competitive snapshot for user %d: stats=%+v err=%v", userID, stats, err)
				}
			}
			var rankLogCount int64
			if err := svcCtx.DB.Model(&model.RankChangeLog{}).Where("match_id = ?", matchID).Count(&rankLogCount).Error; err != nil || rankLogCount != 2 {
				t.Fatalf("completion source must persist both rank writes: count=%d err=%v", rankLogCount, err)
			}
		})
	}
}

func TestStaleMultiInstanceRankedFinishAppliesProjectionOnce(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.MatchParticipantResult{}, &model.UserCompetitiveStats{}, &model.UserOpponentStats{}, &model.UserOpponentStrengthBucket{}); err != nil {
		t.Fatalf("prepare competitive read schema: %v", err)
	}
	svcCtx.CompetitiveReadModel = model.NewCompetitiveReadModel(svcCtx.DB)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: 2002, Nickname: "选手乙"},
	)
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9301, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic,
		Status: 1, MyScore: 5, OpponentScore: 3, CurrentFrameStarted: true,
		MatchTime: time.Now().Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	instanceAMatch, err := svcCtx.MatchModel.FindById(9301)
	if err != nil || instanceAMatch == nil {
		t.Fatalf("load instance A snapshot: match=%+v err=%v", instanceAMatch, err)
	}
	instanceBMatch, err := svcCtx.MatchModel.FindById(9301)
	if err != nil || instanceBMatch == nil {
		t.Fatalf("load instance B snapshot: match=%+v err=%v", instanceBMatch, err)
	}
	if err := svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		_, settleErr := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).settleMatchWithTx(tx, instanceAMatch, 1001, &types.FinishMatchReq{MatchId: 9301, ClientActionId: "instance-a-finish", BaseRevision: 0})
		return settleErr
	}); err != nil {
		t.Fatalf("instance A settlement failed: %v", err)
	}
	secondErr := svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		_, settleErr := NewFinishMatchLogic(finishReputationCtx(2002), svcCtx).settleMatchWithTx(tx, instanceBMatch, 2002, &types.FinishMatchReq{MatchId: 9301, ClientActionId: "instance-b-finish", BaseRevision: 0})
		return settleErr
	})
	if !errors.Is(secondErr, errRevisionConflict) {
		t.Fatalf("stale instance must lose the optimistic write race: err=%v", secondErr)
	}
	var participantCount, finishActionCount, rankLogCount int64
	if err := svcCtx.DB.Model(&model.MatchParticipantResult{}).Where("match_id = ?", 9301).Count(&participantCount).Error; err != nil {
		t.Fatalf("count participant projections: %v", err)
	}
	if err := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", 9301, "match_end").Count(&finishActionCount).Error; err != nil {
		t.Fatalf("count finish actions: %v", err)
	}
	if err := svcCtx.DB.Model(&model.RankChangeLog{}).Where("match_id = ?", 9301).Count(&rankLogCount).Error; err != nil {
		t.Fatalf("count rank logs: %v", err)
	}
	if participantCount != 2 || finishActionCount != 1 || rankLogCount != 2 {
		t.Fatalf("multi-instance finish must commit one consistent dual write: participants=%d actions=%d rankLogs=%d", participantCount, finishActionCount, rankLogCount)
	}
	for _, userID := range []int64{1001, 2002} {
		stats, err := svcCtx.CompetitiveReadModel.FindStats(userID, 0)
		if err != nil || stats == nil || stats.TotalMatches != 1 || stats.Revision != 1 {
			t.Fatalf("user %d snapshot was applied more than once: stats=%+v err=%v", userID, stats, err)
		}
	}
}

func TestRankedFinishDoesNotRollbackWhenLeaderboardInvalidationFails(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.MatchParticipantResult{}, &model.UserCompetitiveStats{}, &model.UserOpponentStats{}, &model.UserOpponentStrengthBucket{}); err != nil {
		t.Fatalf("prepare competitive read schema: %v", err)
	}
	svcCtx.CompetitiveReadModel = model.NewCompetitiveReadModel(svcCtx.DB)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: 2002, Nickname: "选手乙"},
	)
	miniRedis := miniredis.RunT(t)
	svcCtx.Redis = redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = svcCtx.Redis.Close() })
	miniRedis.Close()
	publiclogic.ResetLeaderboardCacheForTest()
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{Id: 9351, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3, MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, Status: 1, MyScore: 5, OpponentScore: 3, CurrentFrameStarted: true, MatchTime: time.Now().Add(-time.Minute)}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: 9351, ClientActionId: "redis-failure-finish", BaseRevision: 0})
	if err != nil || !resp.Success {
		t.Fatalf("Redis invalidation failure must not fail finish: resp=%#v err=%v", resp, err)
	}
	stored, err := svcCtx.MatchModel.FindById(9351)
	stats, statsErr := svcCtx.CompetitiveReadModel.FindStats(1001, 0)
	if err != nil || stored == nil || stored.Status != 2 || statsErr != nil || stats == nil || stats.TotalMatches != 1 {
		t.Fatalf("committed match and snapshot must survive Redis failure: match=%+v matchErr=%v stats=%+v statsErr=%v", stored, err, stats, statsErr)
	}
	if publiclogic.LeaderboardCacheMetrics().WriteErrors == 0 {
		t.Fatal("Redis invalidation failure must be observable")
	}
}

func TestRankedFinishPublishesCurrentSeasonRecordAndBumpsVersions(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(
		&model.MatchParticipantResult{},
		&model.UserCompetitiveStats{},
		&model.UserOpponentStats{},
		&model.UserOpponentStrengthBucket{},
		&model.Season{},
		&model.SeasonRecord{},
	); err != nil {
		t.Fatalf("prepare season projection schema: %v", err)
	}
	svcCtx.CompetitiveReadModel = model.NewCompetitiveReadModel(svcCtx.DB)
	svcCtx.SeasonModel = model.NewSeasonModel(svcCtx.DB)
	svcCtx.SeasonRecordModel = model.NewSeasonRecordModel(svcCtx.DB)
	miniRedis := miniredis.RunT(t)
	svcCtx.Redis = redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	t.Cleanup(func() { _ = svcCtx.Redis.Close() })
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1001, Nickname: "选手甲"},
		model.User{Id: 2002, Nickname: "选手乙"},
	)
	now := time.Now()
	if err := svcCtx.SeasonModel.Create(&model.Season{Id: 1, Name: "当前赛季", StartDate: now.AddDate(0, -1, 0), EndDate: now.AddDate(0, 1, 0), Status: 1}); err != nil {
		t.Fatalf("seed current season: %v", err)
	}
	opponentID := int64(2002)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9401, UserId: 1001, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic,
		Status: 1, MyScore: 5, OpponentScore: 3, CurrentFrameStarted: true,
		MatchTime: now.Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}
	resp, err := NewFinishMatchLogic(finishReputationCtx(1001), svcCtx).FinishMatch(&types.FinishMatchReq{MatchId: 9401, ClientActionId: "season-live-finish", BaseRevision: 0})
	if err != nil || !resp.Success {
		t.Fatalf("finish match: resp=%#v err=%v", resp, err)
	}
	seasonResp, err := seasonlogic.NewGetSeasonOverviewLogic(finishReputationCtx(1001), svcCtx).GetSeasonOverview(&types.GetSeasonOverviewReq{GameType: 3})
	if err != nil || !seasonResp.Success || seasonResp.Record == nil || seasonResp.Record.MatchesPlayed != 1 || len(seasonResp.Leaderboard) != 2 {
		t.Fatalf("season record and leaderboard must be readable immediately: resp=%#v err=%v", seasonResp, err)
	}
	seasonVersion, err := seasonlogic.SeasonLeaderboardCacheVersion(finishReputationCtx(1001), svcCtx, 1, 3)
	if err != nil || seasonVersion != 1 {
		t.Fatalf("unexpected season leaderboard version: version=%d err=%v", seasonVersion, err)
	}
	leaderboardVersion, err := miniRedis.Get("leaderboard:version:3")
	if err != nil || leaderboardVersion != "1" {
		t.Fatalf("unexpected public leaderboard version: value=%q err=%v", leaderboardVersion, err)
	}
}

func containsScope(scopes []string, target string) bool {
	for _, scope := range scopes {
		if scope == target {
			return true
		}
	}
	return false
}

func TestRankedFinishRollsBackWhenCompetitiveProjectionWriteFails(t *testing.T) {
	svcCtx := newFinishMatchReputationTestSvc(t)
	svcCtx.CompetitiveReadModel = model.NewCompetitiveReadModel(svcCtx.DB)
	seedFinishReputationUsers(t, svcCtx,
		model.User{Id: 1101, Nickname: "选手甲"},
		model.User{Id: 2202, Nickname: "选手乙"},
	)
	opponentID := int64(2202)
	if err := svcCtx.MatchModel.Create(&model.Match{
		Id: 9102, UserId: 1101, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic,
		Status: 1, MyScore: 5, OpponentScore: 3, CurrentFrameStarted: true,
		MatchTime: time.Now().Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("create match: %v", err)
	}

	resp, err := NewFinishMatchLogic(finishReputationCtx(1101), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: 9102, ClientActionId: "competitive-failure-9102", BaseRevision: 0,
	})
	if err != nil || resp == nil || resp.Success {
		t.Fatalf("projection failure must reject finish: resp=%#v err=%v", resp, err)
	}
	stored, findErr := svcCtx.MatchModel.FindById(9102)
	if findErr != nil || stored == nil || stored.Status != 1 || stored.CompletedAt != nil {
		t.Fatalf("failed projection must roll back completed match state: match=%+v err=%v", stored, findErr)
	}
	var actionCount int64
	if countErr := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ?", 9102).Count(&actionCount).Error; countErr != nil || actionCount != 0 {
		t.Fatalf("failed projection must roll back finish action: count=%d err=%v", actionCount, countErr)
	}
}
