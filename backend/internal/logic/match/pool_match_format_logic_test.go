package match

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	publiclogic "chasing_points/internal/logic/public"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPoolMatchFormatTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	svcCtx := newFinishMatchReputationTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(
		&model.MatchParticipantResult{},
		&model.UserCompetitiveStats{},
		&model.UserOpponentStats{},
		&model.UserOpponentStrengthBucket{},
		&model.Opponent{},
		&model.MatchAchievement{},
		&model.Achievement{},
		&model.UserAchievement{},
		&model.AchievementRewardConfig{},
		&model.AchievementProgressEvent{},
		&model.MemberGrowthProfile{},
		&model.MemberGrowthLog{},
		&model.MemberRightsConfig{},
	); err != nil {
		t.Fatalf("prepare pool match format schema: %v", err)
	}
	svcCtx.CompetitiveReadModel = model.NewCompetitiveReadModel(svcCtx.DB)
	svcCtx.AchievementModel = model.NewAchievementModel(svcCtx.DB)
	svcCtx.UserAchievementModel = model.NewUserAchievementModel(svcCtx.DB)
	svcCtx.AchievementProgressEventModel = model.NewAchievementProgressEventModel(svcCtx.DB)
	svcCtx.MemberGrowthProfileModel = model.NewMemberGrowthProfileModel(svcCtx.DB)
	svcCtx.MemberGrowthLogModel = model.NewMemberGrowthLogModel(svcCtx.DB)
	svcCtx.MemberRightsConfigModel = model.NewMemberRightsConfigModel(svcCtx.DB)
	return svcCtx
}

func newPoolMatchFormatConcurrentTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "pool-format-concurrency.db")
	db, err := gorm.Open(sqlite.Open("file:"+dbPath+"?_busy_timeout=5000&_journal_mode=WAL"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(2)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare concurrent pool match format schema: %v", err)
	}
	return &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}
}

func seedPoolFormatUsers(t *testing.T, svcCtx *svc.ServiceContext) {
	t.Helper()
	for _, user := range []model.User{
		{Id: 101, Nickname: "选手1", Status: 1},
		{Id: 202, Nickname: "选手2", Status: 1},
		{Id: 303, Nickname: "裁判", Status: 1},
	} {
		seed := user
		if err := svcCtx.DB.Where("id = ?", seed.Id).FirstOrCreate(&seed).Error; err != nil {
			t.Fatalf("seed user %d: %v", seed.Id, err)
		}
	}
}

func seedPoolFormatMatch(t *testing.T, svcCtx *svc.ServiceContext, id int64, mutate func(*model.Match)) *model.Match {
	t.Helper()
	seedPoolFormatUsers(t, svcCtx)
	opponentID := int64(202)
	match := &model.Match{
		Id:                  id,
		UserId:              101,
		OpponentId:          &opponentID,
		OpponentName:        "选手2",
		GameType:            3,
		MatchFormat:         model.MatchFormatFree,
		TargetWins:          0,
		MatchMode:           model.MatchModePractice,
		Visibility:          model.MatchVisibilityPrivate,
		FinishState:         model.FinishStateNone,
		Status:              1,
		CurrentFrameStarted: true,
		MatchTime:           time.Now().Add(-10 * time.Minute),
	}
	if mutate != nil {
		mutate(match)
	}
	if model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked {
		match.Visibility = model.MatchVisibilityPublic
		match.FinishConfirmationRequired = true
	}
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("seed pool match %d: %v", id, err)
	}
	return match
}

func int64Ref(value int64) *int64 {
	return &value
}

func TestPoolMatchFormatStartValidationAndCreation(t *testing.T) {
	svcCtx := newPoolMatchFormatTestSvc(t)
	seedPoolFormatUsers(t, svcCtx)

	resp, err := NewStartMatchLogic(matchLogicCtx(101), svcCtx).StartMatch(&types.StartMatchReq{
		GameType:      3,
		OpponentId:    202,
		OpponentName:  "选手2",
		MatchMode:     model.MatchModePractice,
		Visibility:    model.MatchVisibilityPrivate,
		MatchFormat:   model.MatchFormatFree,
		TargetWins:    0,
		StartingActor: 0,
	})
	if err != nil || !resp.Success || resp.Action != startMatchActionCreated || resp.Match == nil {
		t.Fatalf("start flexible pool match failed: resp=%+v err=%v", resp, err)
	}
	if resp.Match.MatchFormat != model.MatchFormatFree || resp.Match.TargetWins != 0 || !resp.Match.CanChangeMatchFormat {
		t.Fatalf("new pool match should expose editable free format: %+v", resp.Match)
	}

	for _, targetWins := range []int{1, 10, 65} {
		req := &types.StartMatchReq{GameType: 4, OpponentId: 202, MatchFormat: model.MatchFormatRaceTo, TargetWins: targetWins}
		if message := validateStartMatchReq(101, req); message != "" {
			t.Fatalf("race-to %d should be valid, got %q", targetWins, message)
		}
		if req.MatchFormat != model.MatchFormatRaceTo || req.TargetWins != targetWins {
			t.Fatalf("race-to %d normalized incorrectly: %+v", targetWins, req)
		}
	}

	legacyReq := &types.StartMatchReq{GameType: 3, OpponentId: 202}
	if message := validateStartMatchReq(101, legacyReq); message != "" {
		t.Fatalf("missing format should keep legacy compatibility, got %q", message)
	}
	if legacyReq.MatchFormat != model.MatchFormatLegacy || legacyReq.TargetWins != 0 {
		t.Fatalf("missing format should normalize to legacy/0, got %+v", legacyReq)
	}

	for _, tt := range []struct {
		name string
		req  types.StartMatchReq
	}{
		{name: "unknown format", req: types.StartMatchReq{GameType: 3, OpponentId: 202, MatchFormat: "custom"}},
		{name: "free target", req: types.StartMatchReq{GameType: 3, OpponentId: 202, MatchFormat: model.MatchFormatFree, TargetWins: 1}},
		{name: "race target zero", req: types.StartMatchReq{GameType: 3, OpponentId: 202, MatchFormat: model.MatchFormatRaceTo, TargetWins: 0}},
		{name: "race target 66", req: types.StartMatchReq{GameType: 4, OpponentId: 202, MatchFormat: model.MatchFormatRaceTo, TargetWins: 66}},
		{name: "nine-ball chasing", req: types.StartMatchReq{GameType: 2, OpponentId: 202, MatchFormat: model.MatchFormatFree}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req
			if message := validateStartMatchReq(101, &req); message == "" {
				t.Fatalf("expected invalid start match request: %+v", req)
			}
		})
	}
}

func TestPoolUpdateMatchFormatPermissionsLocksAndRevision(t *testing.T) {
	svcCtx := newPoolMatchFormatTestSvc(t)
	match := seedPoolFormatMatch(t, svcCtx, 710, nil)

	resp, err := NewUpdateMatchFormatLogic(matchLogicCtx(101), svcCtx).UpdateMatchFormat(&types.UpdateMatchFormatReq{
		MatchId: match.Id, MatchFormat: model.MatchFormatRaceTo, TargetWins: 10, BaseRevision: 0,
	})
	if err != nil || !resp.Success || !resp.Accepted || resp.ServerRevision != 1 || resp.Snapshot.MatchFormat != model.MatchFormatRaceTo || resp.Snapshot.TargetWins != 10 || !resp.Snapshot.CanChangeMatchFormat {
		t.Fatalf("creator should update format before scoring: resp=%+v err=%v", resp, err)
	}

	opponentResp, err := NewUpdateMatchFormatLogic(matchLogicCtx(202), svcCtx).UpdateMatchFormat(&types.UpdateMatchFormatReq{
		MatchId: match.Id, MatchFormat: model.MatchFormatFree, BaseRevision: 1,
	})
	if err != nil || opponentResp.Success || opponentResp.Accepted || opponentResp.ServerRevision != 1 || opponentResp.Snapshot.CanChangeMatchFormat {
		t.Fatalf("opponent update should be rejected: resp=%+v err=%v", opponentResp, err)
	}

	staleResp, err := NewUpdateMatchFormatLogic(matchLogicCtx(101), svcCtx).UpdateMatchFormat(&types.UpdateMatchFormatReq{
		MatchId: match.Id, MatchFormat: model.MatchFormatFree, BaseRevision: 0,
	})
	if err != nil || staleResp.Success || staleResp.Accepted || staleResp.ServerRevision != 1 {
		t.Fatalf("stale revision should be rejected: resp=%+v err=%v", staleResp, err)
	}

	legacy := seedPoolFormatMatch(t, svcCtx, 711, func(match *model.Match) {
		match.MatchFormat = model.MatchFormatLegacy
		match.TargetWins = 0
	})
	legacyResp, err := NewUpdateMatchFormatLogic(matchLogicCtx(101), svcCtx).UpdateMatchFormat(&types.UpdateMatchFormatReq{
		MatchId: legacy.Id, MatchFormat: model.MatchFormatRaceTo, TargetWins: 3, BaseRevision: 0,
	})
	if err != nil || legacyResp.Success || legacyResp.Accepted || legacyResp.Snapshot.MatchFormat != model.MatchFormatLegacy {
		t.Fatalf("legacy match must not convert to flexible format: resp=%+v err=%v", legacyResp, err)
	}

	for _, tt := range []struct {
		name   string
		id     int64
		mutate func(*model.Match)
		seed   func(*model.Match)
	}{
		{
			name: "action locks format",
			id:   712,
			seed: func(match *model.Match) {
				if err := svcCtx.DB.Create(&model.MatchAction{MatchId: match.Id, RoundNo: 1, ActionType: "score", Actor: 1}).Error; err != nil {
					t.Fatalf("seed action: %v", err)
				}
			},
		},
		{
			name: "round locks format",
			id:   713,
			seed: func(match *model.Match) {
				winner := 1
				if err := svcCtx.DB.Create(&model.MatchRound{MatchId: match.Id, RoundNo: 1, Winner: &winner}).Error; err != nil {
					t.Fatalf("seed round: %v", err)
				}
			},
		},
		{
			name: "referee locks format",
			id:   714,
			mutate: func(match *model.Match) {
				match.RefereeUserId = int64Ref(303)
			},
		},
		{
			name: "finish confirmation locks format",
			id:   715,
			mutate: func(match *model.Match) {
				match.FinishState = model.FinishStatePendingConfirmation
				match.FinishRequestedBy = int64Ref(101)
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			locked := seedPoolFormatMatch(t, svcCtx, tt.id, tt.mutate)
			if tt.seed != nil {
				tt.seed(locked)
			}
			lockedResp, err := NewUpdateMatchFormatLogic(matchLogicCtx(101), svcCtx).UpdateMatchFormat(&types.UpdateMatchFormatReq{
				MatchId: locked.Id, MatchFormat: model.MatchFormatRaceTo, TargetWins: 3, BaseRevision: 0,
			})
			if err != nil || lockedResp.Success || lockedResp.Accepted || lockedResp.Snapshot.CanChangeMatchFormat {
				t.Fatalf("format should be locked: resp=%+v err=%v", lockedResp, err)
			}
		})
	}
}

func TestPoolUpdateMatchFormatConcurrentRevisionRace(t *testing.T) {
	svcCtx := newPoolMatchFormatConcurrentTestSvc(t)
	match := seedPoolFormatMatch(t, svcCtx, 720, nil)

	start := make(chan struct{})
	responses := make(chan *types.UpdateMatchFormatResp, 2)
	var wg sync.WaitGroup
	for _, targetWins := range []int{3, 10} {
		targetWins := targetWins
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			resp, _ := NewUpdateMatchFormatLogic(matchLogicCtx(101), svcCtx).UpdateMatchFormat(&types.UpdateMatchFormatReq{
				MatchId: match.Id, MatchFormat: model.MatchFormatRaceTo, TargetWins: targetWins, BaseRevision: 0,
			})
			responses <- resp
		}()
	}
	close(start)
	wg.Wait()
	close(responses)

	accepted := 0
	for resp := range responses {
		if resp != nil && resp.Success && resp.Accepted {
			accepted++
		}
	}
	stored, err := svcCtx.MatchModel.FindById(match.Id)
	if err != nil {
		t.Fatalf("reload match: %v", err)
	}
	if accepted != 1 || stored.SyncRevision != 1 || stored.MatchFormat != model.MatchFormatRaceTo {
		t.Fatalf("expected exactly one concurrent update, accepted=%d stored=%+v", accepted, stored)
	}
}

func TestFlexiblePoolFreeFinishAndRoundLimit(t *testing.T) {
	t.Run("zero round finish rejected", func(t *testing.T) {
		svcCtx := newPoolMatchFormatTestSvc(t)
		match := seedPoolFormatMatch(t, svcCtx, 730, nil)
		resp, err := NewFinishMatchLogic(matchLogicCtx(101), svcCtx).FinishMatch(&types.FinishMatchReq{
			MatchId: match.Id, ClientActionId: "free-zero-finish", BaseRevision: 0,
		})
		if err != nil || resp.Success || resp.Accepted || resp.Snapshot.Status != 1 {
			t.Fatalf("zero-round free finish should be rejected: resp=%+v err=%v", resp, err)
		}
	})

	t.Run("tie can finish normally", func(t *testing.T) {
		svcCtx := newPoolMatchFormatTestSvc(t)
		match := seedPoolFormatMatch(t, svcCtx, 731, nil)
		first, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
			MatchId: match.Id, Winner: 1, WinType: "normal", Score: 1, ClientActionId: "free-tie-round-1", BaseRevision: 0,
		})
		if err != nil || !first.Success || !first.Accepted || first.Snapshot.Status != 1 {
			t.Fatalf("first free round failed: resp=%+v err=%v", first, err)
		}
		second, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
			MatchId: match.Id, Winner: 2, WinType: "normal", Score: 1, ClientActionId: "free-tie-round-2", BaseRevision: first.ServerRevision,
		})
		if err != nil || !second.Success || !second.Accepted || second.Snapshot.MyScore != 1 || second.Snapshot.OpponentScore != 1 {
			t.Fatalf("second free round failed: resp=%+v err=%v", second, err)
		}
		finish, err := NewFinishMatchLogic(matchLogicCtx(101), svcCtx).FinishMatch(&types.FinishMatchReq{
			MatchId: match.Id, ClientActionId: "free-tie-finish", BaseRevision: second.ServerRevision,
		})
		if err != nil || !finish.Success || !finish.Accepted || finish.Result != 3 || finish.Snapshot.Status != 2 {
			t.Fatalf("free tie finish failed: resp=%+v err=%v", finish, err)
		}
	})

	t.Run("round 129 auto finishes and round 130 is rejected", func(t *testing.T) {
		svcCtx := newPoolMatchFormatTestSvc(t)
		match := seedPoolFormatMatch(t, svcCtx, 732, nil)
		baseRevision := int64(0)
		var last *types.EndRoundResp
		for roundNo := 1; roundNo <= model.PoolMatchMaxCompletedRounds; roundNo++ {
			winner := 1
			if roundNo%2 == 0 {
				winner = 2
			}
			resp, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
				MatchId:        match.Id,
				Winner:         winner,
				WinType:        "normal",
				Score:          1,
				ClientActionId: fmt.Sprintf("free-max-round-%d", roundNo),
				BaseRevision:   baseRevision,
			})
			if err != nil || !resp.Success || !resp.Accepted {
				t.Fatalf("round %d failed: resp=%+v err=%v", roundNo, resp, err)
			}
			baseRevision = resp.ServerRevision
			last = resp
		}
		if last.Snapshot.Status != 2 || last.Snapshot.TotalRounds != model.PoolMatchMaxCompletedRounds || last.Snapshot.MyScore != 65 || last.Snapshot.OpponentScore != 64 {
			t.Fatalf("round 129 should finish at 65:64: %+v", last.Snapshot)
		}
		blocked, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
			MatchId: match.Id, Winner: 2, WinType: "normal", Score: 1, ClientActionId: "free-round-130", BaseRevision: last.ServerRevision,
		})
		if err != nil || blocked.Success || blocked.Accepted || blocked.Snapshot.TotalRounds != model.PoolMatchMaxCompletedRounds {
			t.Fatalf("round 130 should be rejected: resp=%+v err=%v", blocked, err)
		}
	})
}

func TestFlexiblePoolRaceToBoundaryAndIdempotency(t *testing.T) {
	svcCtx := newPoolMatchFormatTestSvc(t)
	match := seedPoolFormatMatch(t, svcCtx, 740, func(match *model.Match) {
		match.MatchFormat = model.MatchFormatRaceTo
		match.TargetWins = 3
	})

	baseRevision := int64(0)
	for roundNo := 1; roundNo <= 2; roundNo++ {
		resp, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
			MatchId:        match.Id,
			Winner:         1,
			WinType:        "normal",
			Score:          1,
			ClientActionId: fmt.Sprintf("race-to-round-%d", roundNo),
			BaseRevision:   baseRevision,
		})
		if err != nil || !resp.Success || !resp.Accepted || resp.Snapshot.Status != 1 {
			t.Fatalf("race-to round %d failed before target: resp=%+v err=%v", roundNo, resp, err)
		}
		baseRevision = resp.ServerRevision
	}
	earlyFinish, err := NewFinishMatchLogic(matchLogicCtx(101), svcCtx).FinishMatch(&types.FinishMatchReq{
		MatchId: match.Id, ClientActionId: "race-to-early-finish", BaseRevision: baseRevision,
	})
	if err != nil || earlyFinish.Success || earlyFinish.Accepted || earlyFinish.Snapshot.Status != 1 {
		t.Fatalf("race-to early finish should be rejected: resp=%+v err=%v", earlyFinish, err)
	}

	targetReq := &types.EndRoundReq{
		MatchId: match.Id, Winner: 1, WinType: "normal", Score: 1, ClientActionId: "race-to-target-round", BaseRevision: baseRevision,
	}
	target, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(targetReq)
	if err != nil || !target.Success || !target.Accepted || target.Snapshot.Status != 2 || target.Snapshot.MyScore != 3 || target.Snapshot.TargetWins != 3 {
		t.Fatalf("race-to target round should auto finish: resp=%+v err=%v", target, err)
	}
	replay, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(targetReq)
	if err != nil || !replay.Success || !replay.Accepted || replay.ServerRevision != target.ServerRevision {
		t.Fatalf("race-to target replay should be idempotent: first=%+v replay=%+v err=%v", target, replay, err)
	}
	blocked, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
		MatchId: match.Id, Winner: 1, WinType: "normal", Score: 1, ClientActionId: "race-to-extra-round", BaseRevision: target.ServerRevision,
	})
	if err != nil || blocked.Success || blocked.Accepted || blocked.Snapshot.MyScore != 3 {
		t.Fatalf("race-to extra round should be rejected: resp=%+v err=%v", blocked, err)
	}

	var winActionCount int64
	if err := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", match.Id, "win").Count(&winActionCount).Error; err != nil {
		t.Fatalf("count win actions: %v", err)
	}
	var endActionCount int64
	if err := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", match.Id, "match_end").Count(&endActionCount).Error; err != nil {
		t.Fatalf("count end actions: %v", err)
	}
	if winActionCount != 3 || endActionCount != 1 {
		t.Fatalf("expected idempotent side effects, win=%d end=%d", winActionCount, endActionCount)
	}
}

func TestFlexiblePoolRankedConfirmationDisputeAndRefereeFinish(t *testing.T) {
	t.Run("ranked target enters pending and dispute cannot continue beyond target", func(t *testing.T) {
		svcCtx := newPoolMatchFormatTestSvc(t)
		match := seedPoolFormatMatch(t, svcCtx, 750, func(match *model.Match) {
			match.MatchFormat = model.MatchFormatRaceTo
			match.TargetWins = 1
			match.MatchMode = model.MatchModeRanked
		})
		resp, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
			MatchId: match.Id, Winner: 1, WinType: "normal", Score: 1, ClientActionId: "ranked-target-round", BaseRevision: 0,
		})
		if err != nil || !resp.Success || !resp.Accepted || resp.Snapshot.Status != 1 || resp.Snapshot.FinishState != model.FinishStatePendingConfirmation {
			t.Fatalf("ranked target should enter pending confirmation: resp=%+v err=%v", resp, err)
		}
		replay, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
			MatchId: match.Id, Winner: 1, WinType: "normal", Score: 1, ClientActionId: "ranked-target-round", BaseRevision: 0,
		})
		if err != nil || replay.Success || replay.Accepted || replay.ServerRevision != resp.ServerRevision {
			t.Fatalf("pending confirmation replay should be rejected without advancing revision: first=%+v replay=%+v err=%v", resp, replay, err)
		}
		var winActionCount int64
		if err := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", match.Id, "win").Count(&winActionCount).Error; err != nil {
			t.Fatalf("count win actions: %v", err)
		}
		var finishRequestCount int64
		if err := svcCtx.DB.Model(&model.MatchAction{}).Where("match_id = ? AND action_type = ?", match.Id, "finish_request").Count(&finishRequestCount).Error; err != nil {
			t.Fatalf("count finish request actions: %v", err)
		}
		if winActionCount != 1 || finishRequestCount != 1 {
			t.Fatalf("pending replay should not duplicate side effects, win=%d finish_request=%d", winActionCount, finishRequestCount)
		}
		pendingBlocked, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
			MatchId: match.Id, Winner: 2, WinType: "normal", Score: 1, ClientActionId: "ranked-pending-extra", BaseRevision: resp.ServerRevision,
		})
		if err != nil || pendingBlocked.Success || pendingBlocked.Accepted || pendingBlocked.Snapshot.FinishState != model.FinishStatePendingConfirmation {
			t.Fatalf("pending confirmation should block scoring: resp=%+v err=%v", pendingBlocked, err)
		}
		dispute, err := NewDisputeFinishMatchLogic(matchLogicCtx(202), svcCtx).DisputeFinishMatch(&types.FinishMatchActionReq{
			MatchId: match.Id, ClientActionId: "ranked-target-dispute", BaseRevision: resp.ServerRevision,
		})
		if err != nil || !dispute.Success || !dispute.Accepted || dispute.FinishState != model.FinishStateNone {
			t.Fatalf("opponent should dispute pending finish: resp=%+v err=%v", dispute, err)
		}
		afterDisputeBlocked, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
			MatchId: match.Id, Winner: 2, WinType: "normal", Score: 1, ClientActionId: "ranked-after-dispute-extra", BaseRevision: dispute.ServerRevision,
		})
		if err != nil || afterDisputeBlocked.Success || afterDisputeBlocked.Accepted || afterDisputeBlocked.Snapshot.MyScore != 1 {
			t.Fatalf("dispute should not allow scoring beyond reached target: resp=%+v err=%v", afterDisputeBlocked, err)
		}
	})

	t.Run("referee-bound target completes directly", func(t *testing.T) {
		svcCtx := newPoolMatchFormatTestSvc(t)
		match := seedPoolFormatMatch(t, svcCtx, 751, func(match *model.Match) {
			match.MatchFormat = model.MatchFormatRaceTo
			match.TargetWins = 1
			match.MatchMode = model.MatchModeRanked
			match.RefereeUserId = int64Ref(303)
		})
		resp, err := NewEndRoundLogic(matchLogicCtx(303), svcCtx).EndRound(&types.EndRoundReq{
			MatchId: match.Id, Winner: 2, WinType: "normal", Score: 1, ClientActionId: "referee-target-round", BaseRevision: 0,
		})
		if err != nil || !resp.Success || !resp.Accepted || resp.Snapshot.Status != 2 || resp.Snapshot.FinishState != model.FinishStateNone {
			t.Fatalf("referee-bound target should complete directly: resp=%+v err=%v", resp, err)
		}
		stored, err := svcCtx.MatchModel.FindById(match.Id)
		if err != nil || stored == nil || stored.CompletedByUserId == nil || *stored.CompletedByUserId != 303 || stored.CompletionSource != model.CompletionSourceReferee {
			t.Fatalf("expected referee completion attribution: stored=%+v err=%v", stored, err)
		}
	})
}

func TestFlexiblePoolRejectsScoreAndFoulBypassesAndKeepsLegacy(t *testing.T) {
	svcCtx := newPoolMatchFormatTestSvc(t)
	flexible := seedPoolFormatMatch(t, svcCtx, 760, nil)

	badRound, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
		MatchId: flexible.Id, Winner: 1, WinType: "normal", Score: 2, ClientActionId: "pool-bad-round-score", BaseRevision: 0,
	})
	if err != nil || badRound.Success || badRound.Accepted || badRound.ServerRevision != 0 {
		t.Fatalf("flexible round score!=1 should be rejected: resp=%+v err=%v", badRound, err)
	}
	score, err := NewMatchScoreLogic(matchLogicCtx(101), svcCtx).MatchScore(&types.MatchScoreReq{
		MatchId: flexible.Id, Actor: 1, Score: 5, ClientActionId: "pool-bypass-score", BaseRevision: 0,
	})
	if err != nil || score.Success || score.Accepted || score.ServerRevision != 0 || score.MyScore != 0 {
		t.Fatalf("flexible score bypass should be rejected: resp=%+v err=%v", score, err)
	}
	foul, err := NewMatchFoulLogic(matchLogicCtx(101), svcCtx).MatchFoul(&types.MatchFoulReq{
		MatchId: flexible.Id, Actor: 1, Score: 1, ClientActionId: "pool-bypass-foul", BaseRevision: 0,
	})
	if err != nil || foul.Success || foul.Accepted || foul.ServerRevision != 0 || foul.OpponentScore != 0 {
		t.Fatalf("flexible foul bypass should be rejected: resp=%+v err=%v", foul, err)
	}

	legacy := seedPoolFormatMatch(t, svcCtx, 761, func(match *model.Match) {
		match.MatchFormat = model.MatchFormatLegacy
	})
	legacyScore, err := NewMatchScoreLogic(matchLogicCtx(101), svcCtx).MatchScore(&types.MatchScoreReq{
		MatchId: legacy.Id, Actor: 1, Score: 5, ClientActionId: "legacy-score", BaseRevision: 0,
	})
	if err != nil || !legacyScore.Success || !legacyScore.Accepted || legacyScore.MyScore != 5 {
		t.Fatalf("legacy pool score should keep old behavior: resp=%+v err=%v", legacyScore, err)
	}

	nineBallChasing := seedPoolFormatMatch(t, svcCtx, 762, func(match *model.Match) {
		match.GameType = 2
		match.MatchFormat = model.MatchFormatLegacy
	})
	nineBallScore, err := NewMatchScoreLogic(matchLogicCtx(101), svcCtx).MatchScore(&types.MatchScoreReq{
		MatchId: nineBallChasing.Id, Actor: 1, Score: 4, ClientActionId: "nine-ball-score", BaseRevision: 0,
	})
	if err != nil || !nineBallScore.Success || !nineBallScore.Accepted || nineBallScore.MyScore != 4 {
		t.Fatalf("nine-ball chasing score should keep old behavior: resp=%+v err=%v", nineBallScore, err)
	}
}

func TestFlexiblePoolSnapshotsExposeFormatAndViewerPermissions(t *testing.T) {
	svcCtx := newPoolMatchFormatTestSvc(t)
	match := seedPoolFormatMatch(t, svcCtx, 770, func(match *model.Match) {
		match.Visibility = model.MatchVisibilityPublic
		match.MatchMode = model.MatchModePractice
	})

	currentOwner, err := NewGetCurrentMatchLogic(matchLogicCtx(101), svcCtx).GetCurrentMatch()
	if err != nil || !currentOwner.Success || currentOwner.Match == nil || currentOwner.Match.MatchFormat != model.MatchFormatFree || !currentOwner.Match.CanChangeMatchFormat {
		t.Fatalf("owner current match should expose editable free format: resp=%+v err=%v", currentOwner, err)
	}
	currentOpponent, err := NewGetCurrentMatchLogic(matchLogicCtx(202), svcCtx).GetCurrentMatch()
	if err != nil || !currentOpponent.Success || currentOpponent.Match == nil || currentOpponent.Match.CanChangeMatchFormat {
		t.Fatalf("opponent current match should be read-only: resp=%+v err=%v", currentOpponent, err)
	}

	detailOwner, err := NewGetMatchDetailLogic(matchLogicCtx(101), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: match.Id})
	if err != nil || !detailOwner.Success || detailOwner.Match.MatchFormat != model.MatchFormatFree || !detailOwner.Match.CanChangeMatchFormat {
		t.Fatalf("owner detail should expose editable format: resp=%+v err=%v", detailOwner, err)
	}
	detailOpponent, err := NewGetMatchDetailLogic(matchLogicCtx(202), svcCtx).GetMatchDetail(&types.GetMatchDetailReq{MatchId: match.Id})
	if err != nil || !detailOpponent.Success || detailOpponent.Match.CanChangeMatchFormat || detailOpponent.Match.MatchFormat != model.MatchFormatFree {
		t.Fatalf("opponent detail should expose read-only format: resp=%+v err=%v", detailOpponent, err)
	}

	publicDetail, err := publiclogic.NewGetPublicMatchDetailLogic(context.Background(), svcCtx).GetPublicMatchDetail(&types.GetPublicMatchDetailReq{MatchId: match.Id})
	if err != nil || !publicDetail.Success || publicDetail.Match == nil || publicDetail.Match.MatchFormat != model.MatchFormatFree || publicDetail.Match.CanChangeMatchFormat {
		t.Fatalf("public detail should expose read-only format: resp=%+v err=%v", publicDetail, err)
	}

	ownerState, err := loadMatchWriteState(svcCtx, 101, match)
	if err != nil || ownerState.Snapshot.MatchFormat != model.MatchFormatFree || !ownerState.Snapshot.CanChangeMatchFormat {
		t.Fatalf("owner sync snapshot should expose editable format: state=%+v err=%v", ownerState, err)
	}
	opponentState, err := loadMatchWriteState(svcCtx, 202, match)
	if err != nil || opponentState.Snapshot.MatchFormat != model.MatchFormatFree || opponentState.Snapshot.CanChangeMatchFormat {
		t.Fatalf("opponent sync snapshot should expose read-only format: state=%+v err=%v", opponentState, err)
	}

	round, err := NewEndRoundLogic(matchLogicCtx(101), svcCtx).EndRound(&types.EndRoundReq{
		MatchId: match.Id, Winner: 1, WinType: "normal", Score: 1, ClientActionId: "snapshot-lock-round", BaseRevision: 0,
	})
	if err != nil || !round.Success || round.Snapshot.MatchFormat != model.MatchFormatFree || round.Snapshot.CanChangeMatchFormat {
		t.Fatalf("round response should lock format in authoritative snapshot: resp=%+v err=%v", round, err)
	}
}
