package match

import (
	"testing"
	"time"

	achievementx "chasing_points/internal/logic/achievement"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func TestGetMatchRewardSummaryReturnsParticipantSpecificRewards(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	opponentID := int64(2002)
	seedMatchAchievementUsers(t, svcCtx, 1001, opponentID)
	seedRewardSummaryAchievement(t, svcCtx, model.Achievement{
		Id: 101, Key: "first_match", Name: "初次登场", Description: "完成首场排位", Icon: "first.png",
		MetricKey: achievementx.MetricMatchesTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Status: 1,
		RewardTitleKey: "title_first_match", RewardTitleName: "初次登场",
	})
	seedRewardSummaryAchievement(t, svcCtx, model.Achievement{
		Id: 102, Key: "first_win", Name: "首战告捷", Description: "赢得首场排位", Icon: "win.png",
		MetricKey: achievementx.MetricWinsTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Status: 1,
		RewardTitleKey: "title_first_win", RewardTitleName: "首战告捷",
	})
	syncedAt := time.Now()
	seedRewardSummaryCompletedMatch(t, svcCtx, 8001, 1001, opponentID, &syncedAt)
	seedRewardSummaryUnlock(t, svcCtx, 1001, 101, 8001, "初次登场")
	seedRewardSummaryUnlock(t, svcCtx, opponentID, 102, 8001, "首战告捷")

	player1Resp, err := NewGetMatchRewardSummaryLogic(matchAchievementCtx(1001), svcCtx).
		GetMatchRewardSummary(&types.GetMatchRewardSummaryReq{MatchId: 8001})
	if err != nil {
		t.Fatalf("get player1 reward summary: %v", err)
	}
	if !player1Resp.Success || player1Resp.Status != "ready" || len(player1Resp.List) != 1 {
		t.Fatalf("unexpected player1 response: %+v", player1Resp)
	}
	if player1Resp.List[0].AchievementId != 101 || player1Resp.List[0].RewardTitleName != "初次登场" {
		t.Fatalf("unexpected player1 reward: %+v", player1Resp.List[0])
	}

	player2Resp, err := NewGetMatchRewardSummaryLogic(matchAchievementCtx(opponentID), svcCtx).
		GetMatchRewardSummary(&types.GetMatchRewardSummaryReq{MatchId: 8001})
	if err != nil {
		t.Fatalf("get player2 reward summary: %v", err)
	}
	if !player2Resp.Success || player2Resp.Status != "ready" || len(player2Resp.List) != 1 {
		t.Fatalf("unexpected player2 response: %+v", player2Resp)
	}
	if player2Resp.List[0].AchievementId != 102 || player2Resp.List[0].RewardTitleName != "首战告捷" {
		t.Fatalf("unexpected player2 reward: %+v", player2Resp.List[0])
	}
}

func TestGetMatchRewardSummaryRejectsNonParticipant(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	opponentID := int64(2002)
	seedMatchAchievementUsers(t, svcCtx, 1001, opponentID)
	syncedAt := time.Now()
	seedRewardSummaryCompletedMatch(t, svcCtx, 8002, 1001, opponentID, &syncedAt)

	resp, err := NewGetMatchRewardSummaryLogic(matchAchievementCtx(3003), svcCtx).
		GetMatchRewardSummary(&types.GetMatchRewardSummaryReq{MatchId: 8002})
	if err != nil {
		t.Fatalf("get non-participant reward summary: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected non-participant request rejected, got %+v", resp)
	}
}

func TestGetMatchRewardSummaryReturnsReadyEmptyList(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	opponentID := int64(2002)
	seedMatchAchievementUsers(t, svcCtx, 1001, opponentID)
	syncedAt := time.Now()
	seedRewardSummaryCompletedMatch(t, svcCtx, 8003, 1001, opponentID, &syncedAt)

	resp, err := NewGetMatchRewardSummaryLogic(matchAchievementCtx(1001), svcCtx).
		GetMatchRewardSummary(&types.GetMatchRewardSummaryReq{MatchId: 8003})
	if err != nil {
		t.Fatalf("get empty reward summary: %v", err)
	}
	if !resp.Success || resp.Status != "ready" || resp.List == nil || len(resp.List) != 0 {
		t.Fatalf("expected ready empty list, got %+v", resp)
	}
}

func TestGetMatchRewardSummaryRetriesPendingSyncAndStaysIdempotent(t *testing.T) {
	svcCtx := newMatchAchievementClosedLoopTestSvc(t)
	opponentID := int64(2002)
	seedMatchAchievementUsers(t, svcCtx, 1001, opponentID)
	seedRewardSummaryAchievement(t, svcCtx, model.Achievement{
		Id: 103, Key: "first_match", Name: "初次登场", Description: "完成首场排位", Icon: "first.png",
		MetricKey: achievementx.MetricMatchesTotal, ProgressMode: achievementx.ProgressModeSum, Threshold: 1, Status: 1,
		RewardTitleKey: "title_first_match", RewardTitleName: "初次登场",
	})
	seedRewardSummaryCompletedMatch(t, svcCtx, 8004, 1001, opponentID, nil)
	seedMatchAchievementRound(t, svcCtx, 8004, 1, 1)

	eventModel := svcCtx.AchievementProgressEventModel
	svcCtx.AchievementProgressEventModel = nil
	pendingResp, err := NewGetMatchRewardSummaryLogic(matchAchievementCtx(1001), svcCtx).
		GetMatchRewardSummary(&types.GetMatchRewardSummaryReq{MatchId: 8004})
	if err != nil {
		t.Fatalf("get pending reward summary: %v", err)
	}
	if !pendingResp.Success || pendingResp.Status != "pending" || len(pendingResp.List) != 0 {
		t.Fatalf("expected pending response, got %+v", pendingResp)
	}
	assertMatchAchievementSynced(t, svcCtx, 8004, false)

	svcCtx.AchievementProgressEventModel = eventModel
	readyResp, err := NewGetMatchRewardSummaryLogic(matchAchievementCtx(1001), svcCtx).
		GetMatchRewardSummary(&types.GetMatchRewardSummaryReq{MatchId: 8004})
	if err != nil {
		t.Fatalf("retry reward summary: %v", err)
	}
	if !readyResp.Success || readyResp.Status != "ready" || len(readyResp.List) != 1 {
		t.Fatalf("expected compensated ready response, got %+v", readyResp)
	}
	if readyResp.List[0].AchievementId != 103 || readyResp.List[0].RewardTitleName != "初次登场" {
		t.Fatalf("unexpected compensated reward: %+v", readyResp.List[0])
	}
	assertMatchAchievementSynced(t, svcCtx, 8004, true)

	repeatResp, err := NewGetMatchRewardSummaryLogic(matchAchievementCtx(1001), svcCtx).
		GetMatchRewardSummary(&types.GetMatchRewardSummaryReq{MatchId: 8004})
	if err != nil {
		t.Fatalf("repeat reward summary: %v", err)
	}
	if !repeatResp.Success || repeatResp.Status != "ready" || len(repeatResp.List) != 1 {
		t.Fatalf("unexpected repeated response: %+v", repeatResp)
	}
	if countProgressEvents(t, svcCtx, 1001, achievementx.MetricMatchesTotal) != 1 {
		t.Fatal("expected compensation retry to keep progress event idempotent")
	}
	var titleCount int64
	if err := svcCtx.DB.Model(&model.UserTitle{}).Where("user_id = ?", 1001).Count(&titleCount).Error; err != nil {
		t.Fatalf("count reward titles: %v", err)
	}
	if titleCount != 1 {
		t.Fatalf("expected one reward title after repeated queries, got %d", titleCount)
	}
}

func seedRewardSummaryAchievement(t *testing.T, svcCtx *svc.ServiceContext, achievement model.Achievement) {
	t.Helper()
	if err := svcCtx.DB.Create(&achievement).Error; err != nil {
		t.Fatalf("seed reward summary achievement %s: %v", achievement.Key, err)
	}
}

func seedRewardSummaryCompletedMatch(t *testing.T, svcCtx *svc.ServiceContext, matchID, userID, opponentID int64, syncedAt *time.Time) {
	t.Helper()
	result := 1
	endedAt := time.Now().Add(-time.Minute)
	seedMatchAchievementMatch(t, svcCtx, &model.Match{
		Id:                  matchID,
		UserId:              userID,
		OpponentId:          &opponentID,
		OpponentName:        "选手乙",
		GameType:            3,
		MatchMode:           model.MatchModeRanked,
		MyScore:             1,
		OpponentScore:       0,
		Status:              2,
		Result:              &result,
		MatchTime:           endedAt.Add(-10 * time.Minute),
		EndTime:             &endedAt,
		AchievementSyncedAt: syncedAt,
	})
}

func seedRewardSummaryUnlock(t *testing.T, svcCtx *svc.ServiceContext, userID, achievementID, matchID int64, titleName string) {
	t.Helper()
	unlockedAt := time.Now()
	if err := svcCtx.DB.Create(&model.UserAchievement{
		UserId:             userID,
		AchievementId:      achievementID,
		Progress:           1,
		Unlocked:           1,
		RewardGranted:      1,
		UnlockedAt:         &unlockedAt,
		UnlockedSourceType: achievementx.SourceTypeMatch,
		UnlockedSourceId:   matchID,
		RewardGrantedAt:    &unlockedAt,
	}).Error; err != nil {
		t.Fatalf("seed reward summary unlock: %v", err)
	}
	if err := svcCtx.DB.Create(&model.UserTitle{
		UserId:                 userID,
		TitleKey:               "title_" + titleName,
		TitleName:              titleName,
		Source:                 achievementx.SourceTypeAchievement,
		SourceType:             achievementx.SourceTypeAchievement,
		SourceRefId:            achievementID,
		SourceRefName:          titleName,
		GrantedByAchievementId: &achievementID,
		GrantedAt:              &unlockedAt,
	}).Error; err != nil {
		t.Fatalf("seed reward summary title: %v", err)
	}
}
