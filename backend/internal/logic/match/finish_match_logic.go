package match

import (
	"context"
	"fmt"
	"time"

	logicx "chasing_points/internal/logic"
	achievementx "chasing_points/internal/logic/achievement"
	publiclogic "chasing_points/internal/logic/public"
	seasonlogic "chasing_points/internal/logic/season"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type FinishMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 结束对局
func NewFinishMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FinishMatchLogic {
	return &FinishMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *FinishMatchLogic) FinishMatch(req *types.FinishMatchReq) (resp *types.FinishMatchResp, err error) {
	return l.finishMatchImmediately(req, false)
}

func (l *FinishMatchLogic) finishMatchImmediately(req *types.FinishMatchReq, force bool) (resp *types.FinishMatchResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.FinishMatchResp{Success: false}, nil
	}

	// 查询对局
	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		l.Logger.Errorf("对局不存在: %v", err)
		return &types.FinishMatchResp{Success: false}, nil
	}
	if isSnookerV2Match(match) {
		if _, authorityErr := validateMatchWriteAuthority(match, userId); authorityErr != nil {
			return &types.FinishMatchResp{Success: false, Accepted: false, Message: authorityErr.Error()}, nil
		}
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			return &types.FinishMatchResp{Success: false, Accepted: false, Message: "加载对局快照失败"}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.FinishMatchResp{
			Success:        false,
			Accepted:       false,
			Message:        "版本2斯诺克将按赛制自动结束，请使用认输或裁判判局",
			ClientActionId: req.ClientActionId,
			ServerRevision: view.Snapshot.ServerRevision,
			Snapshot:       view.Snapshot,
			MyScore:        scoreView.MyScore,
			OpponentScore:  scoreView.OpponentScore,
		}, nil
	}
	if !force && shouldRequestRankedFinish(match, userId) {
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			return &types.FinishMatchResp{Success: false, Accepted: false, Message: "加载对局快照失败"}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		return &types.FinishMatchResp{
			Success:        false,
			Accepted:       false,
			Message:        "本场排位赛需要双方确认，请使用结束确认流程",
			Result:         3,
			ClientActionId: req.ClientActionId,
			ServerRevision: view.Snapshot.ServerRevision,
			Snapshot:       view.Snapshot,
			MyScore:        scoreView.MyScore,
			OpponentScore:  scoreView.OpponentScore,
		}, nil
	}

	// 验证用户权限
	_, authorityErr := validateMatchWriteAuthority(match, userId)
	if authorityErr != nil {
		if authorityErr == errMatchViewerNotParticipant {
			l.Logger.Errorf("用户不是对局参与者: matchUserId=%d, matchOpponentId=%v, refereeUserId=%v, currentUserId=%d",
				match.UserId, match.OpponentId, match.RefereeUserId, userId)
			return &types.FinishMatchResp{Success: false}, nil
		}
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载权限拒绝快照失败: matchId=%d, userId=%d, err=%v", match.Id, userId, stateErr)
			return &types.FinishMatchResp{Success: false, Accepted: false}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		result := 3
		if match.Result != nil {
			result = *match.Result
		}
		return &types.FinishMatchResp{
			Success:        false,
			Accepted:       false,
			Result:         result,
			ServerRevision: view.Snapshot.ServerRevision,
			Snapshot:       view.Snapshot,
			MyScore:        scoreView.MyScore,
			OpponentScore:  scoreView.OpponentScore,
		}, nil
	}
	if req.ClientActionId == "" {
		return &types.FinishMatchResp{Success: false, Accepted: false}, nil
	}
	existingAction, err := l.svcCtx.MatchModel.FindActionByClientActionID(match.Id, req.ClientActionId)
	if err != nil {
		l.Logger.Errorf("查询幂等操作失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, err)
		return &types.FinishMatchResp{Success: false, Accepted: false}, nil
	}
	if existingAction != nil {
		if existingAction.ActionType != "match_end" || existingAction.Actor != resolveFinishActionActor(match, userId) || existingAction.BaseRevision != req.BaseRevision {
			return &types.FinishMatchResp{Success: false, Accepted: false, ClientActionId: req.ClientActionId}, nil
		}
		if model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked {
			l.awardMemberGrowthForMatch(match.Id, match.UserId, match.OpponentId)
		}
		l.syncAchievementProgressForCompletedMatch(match)
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载重放快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
			return &types.FinishMatchResp{Success: false, Accepted: false}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		result := 3
		if match.Result != nil {
			result = *match.Result
		}
		return &types.FinishMatchResp{
			Accepted:       true,
			Success:        true,
			Result:         result,
			ClientActionId: req.ClientActionId,
			ServerRevision: view.Snapshot.ServerRevision,
			Snapshot:       view.Snapshot,
			MyScore:        scoreView.MyScore,
			OpponentScore:  scoreView.OpponentScore,
		}, nil
	}
	if err := validateMatchActionMeta(matchActionMeta{
		ClientActionID: req.ClientActionId,
		BaseRevision:   req.BaseRevision,
	}, match.SyncRevision); err != nil {
		view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
		if stateErr != nil {
			l.Logger.Errorf("加载冲突快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
			return &types.FinishMatchResp{Success: false, Accepted: false}, nil
		}
		scoreView := buildMatchWriteScoreView(userId, match)
		result := 3
		if match.Result != nil {
			result = *match.Result
		}
		return &types.FinishMatchResp{
			Accepted:       false,
			Success:        false,
			Result:         result,
			ClientActionId: req.ClientActionId,
			ServerRevision: view.Snapshot.ServerRevision,
			Snapshot:       view.Snapshot,
			MyScore:        scoreView.MyScore,
			OpponentScore:  scoreView.OpponentScore,
		}, nil
	}

	var settlement finishMatchSettlement
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		settlement, err = l.settleMatchWithTx(tx, match, userId, req)
		return err
	})
	if err != nil {
		if isRetryableMatchWriteError(err) {
			replayState, replayErr := reloadMatchWriteReplayState(l.svcCtx, userId, req.MatchId, req.ClientActionId)
			if replayErr != nil {
				l.Logger.Errorf("重载写入快照失败: matchId=%d, clientActionId=%s, err=%v", req.MatchId, req.ClientActionId, replayErr)
				return &types.FinishMatchResp{Success: false, Accepted: false}, nil
			}
			scoreView := buildMatchWriteScoreView(userId, replayState.Match)
			result := 3
			if replayState.Match != nil && replayState.Match.Result != nil {
				result = *replayState.Match.Result
			}
			if replayState.ExistingAction != nil {
				if replayState.Match != nil {
					if model.NormalizeMatchMode(replayState.Match.MatchMode) == model.MatchModeRanked {
						l.awardMemberGrowthForMatch(replayState.Match.Id, replayState.Match.UserId, replayState.Match.OpponentId)
					}
					l.syncAchievementProgressForCompletedMatch(replayState.Match)
				}
				return &types.FinishMatchResp{
					Accepted:       true,
					Success:        true,
					Result:         result,
					ClientActionId: req.ClientActionId,
					ServerRevision: replayState.View.Snapshot.ServerRevision,
					Snapshot:       replayState.View.Snapshot,
					MyScore:        scoreView.MyScore,
					OpponentScore:  scoreView.OpponentScore,
				}, nil
			}
			return &types.FinishMatchResp{
				Accepted:       false,
				Success:        false,
				Result:         result,
				ClientActionId: req.ClientActionId,
				ServerRevision: replayState.View.Snapshot.ServerRevision,
				Snapshot:       replayState.View.Snapshot,
				MyScore:        scoreView.MyScore,
				OpponentScore:  scoreView.OpponentScore,
			}, nil
		}
		l.Logger.Errorf("结束对局事务失败: %v", err)
		return &types.FinishMatchResp{Success: false}, nil
	}
	result := settlement.Result
	return l.finishMatchPostCommit(req, userId, match, result, settlement.CompetitiveRevisions, settlement.SeasonID)
}

func (l *FinishMatchLogic) finishMatchPostCommit(req *types.FinishMatchReq, userId int64, match *model.Match, result int, competitiveRevisions map[int64]int64, seasonID int64) (*types.FinishMatchResp, error) {
	if model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked {
		l.applyMatchReputation(match)
	}
	view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
	if stateErr != nil {
		l.Logger.Errorf("加载写入快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
		return &types.FinishMatchResp{Success: false, Accepted: false}, nil
	}
	if model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked {
		l.awardMemberGrowthForMatch(match.Id, match.UserId, match.OpponentId)
	}
	l.syncAchievementProgressForCompletedMatch(match)

	l.Logger.Infof("用户 %d 结束对局 %d，比分: %d:%d，结果: %d",
		userId, match.Id, match.MyScore, match.OpponentScore, result)
	broadcastRankInfoUpdated(match, result)
	broadcastUserDataUpdated(match, result, competitiveRevisions)
	l.invalidateCompetitiveSharedReads(match, result, seasonID)
	if result != 3 {
		resultText := "胜利"
		if result == 2 {
			resultText = "失败"
		}
		content := fmt.Sprintf("你的对局已结束，结果：%s（%d:%d）", resultText, match.MyScore, match.OpponentScore)
		if notifyErr := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
			UserId:      match.UserId,
			Type:        "match_result",
			DedupeKey:   fmt.Sprintf("match:%d", match.Id),
			Title:       "对局已结束",
			Content:     content,
			Data:        buildNotificationPayload("/subPages/match/matchResult", match.Id, 0),
			PushTitle:   "对局已结束",
			PushContent: fmt.Sprintf("结果：%s（%d:%d）", resultText, match.MyScore, match.OpponentScore),
			PushData: map[string]interface{}{
				"url": fmt.Sprintf("/subPages/match/matchResult?match_id=%d", match.Id),
			},
			WSCategory: "match_result",
		}); notifyErr != nil {
			l.Logger.Errorf("分发己方对局结束通知失败: matchId=%d userId=%d err=%v", match.Id, match.UserId, notifyErr)
		}
		if match.OpponentId != nil && *match.OpponentId > 0 {
			opResultText := "胜利"
			if result == 1 {
				opResultText = "失败"
			}
			opContent := fmt.Sprintf("你的对局已结束，结果：%s（%d:%d）", opResultText, match.OpponentScore, match.MyScore)
			if notifyErr := logicx.DispatchNotification(l.svcCtx, logicx.NotificationDispatchInput{
				UserId:      *match.OpponentId,
				Type:        "match_result",
				DedupeKey:   fmt.Sprintf("match:%d", match.Id),
				Title:       "对局已结束",
				Content:     opContent,
				Data:        buildNotificationPayload("/subPages/match/matchResult", match.Id, 0),
				PushTitle:   "对局已结束",
				PushContent: fmt.Sprintf("结果：%s（%d:%d）", opResultText, match.OpponentScore, match.MyScore),
				PushData: map[string]interface{}{
					"url": fmt.Sprintf("/subPages/match/matchResult?match_id=%d", match.Id),
				},
				WSCategory: "match_result",
			}); notifyErr != nil {
				l.Logger.Errorf("分发对手对局结束通知失败: matchId=%d userId=%d err=%v", match.Id, *match.OpponentId, notifyErr)
			}
		}
	}

	if ws.GlobalHub != nil {
		l.Logger.Infof("广播对局结束: matchId=%d, result=%d, player1=%d, player2=%d",
			match.Id, result, match.MyScore, match.OpponentScore)
		ws.GlobalHub.BroadcastToMatch(match.Id, &ws.Message{
			Type: "match_end",
			Data: map[string]interface{}{
				"match_id":        match.Id,
				"server_revision": view.Snapshot.ServerRevision,
				"my_score":        match.MyScore,
				"opponent_score":  match.OpponentScore,
				"player1_score":   match.MyScore,
				"player2_score":   match.OpponentScore,
				"status":          match.Status,
				"result":          result,
				"match_mode":      model.NormalizeMatchMode(match.MatchMode),
			},
		})
	}

	scoreView := buildMatchWriteScoreView(userId, match)
	return &types.FinishMatchResp{
		Accepted:       true,
		Success:        true,
		Result:         result,
		ClientActionId: req.ClientActionId,
		ServerRevision: view.Snapshot.ServerRevision,
		Snapshot:       view.Snapshot,
		MyScore:        scoreView.MyScore,
		OpponentScore:  scoreView.OpponentScore,
	}, nil
}

func broadcastUserDataUpdated(match *model.Match, result int, competitiveRevisions map[int64]int64) {
	if match == nil {
		return
	}
	scopes := []string{"history", "h2h", "opponents"}
	if result != 3 && model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked {
		scopes = append(scopes, "rank", "stats", "honor", "season", "leaderboard")
	}
	userIDs := []int64{match.UserId}
	if match.OpponentId != nil && *match.OpponentId > 0 && *match.OpponentId != match.UserId {
		userIDs = append(userIDs, *match.OpponentId)
	}
	for _, targetUserID := range userIDs {
		logicx.SendUserDataUpdated(targetUserID, logicx.UserDataUpdatedEvent{
			Scopes:              scopes,
			CompetitiveRevision: competitiveRevisions[targetUserID],
			MatchID:             match.Id,
			GameType:            match.GameType,
		})
	}
}

func (l *FinishMatchLogic) invalidateCompetitiveSharedReads(match *model.Match, result int, seasonID int64) {
	if l == nil || l.svcCtx == nil || l.svcCtx.Redis == nil || match == nil || result == 3 || model.NormalizeMatchMode(match.MatchMode) != model.MatchModeRanked {
		return
	}
	if err := publiclogic.BumpLeaderboardCacheVersion(l.ctx, l.svcCtx, match.GameType); err != nil {
		l.Logger.Errorf("失效公共排行榜缓存失败: matchId=%d gameType=%d err=%v", match.Id, match.GameType, err)
	}
	if seasonID <= 0 && l.svcCtx.SeasonModel != nil {
		completedAt := match.MatchTime
		if match.CompletedAt != nil {
			completedAt = *match.CompletedAt
		} else if match.EndTime != nil {
			completedAt = *match.EndTime
		}
		season, err := l.svcCtx.SeasonModel.FindByEffectiveTime(completedAt)
		if err != nil {
			l.Logger.Errorf("解析赛季排行榜缓存版本失败: matchId=%d err=%v", match.Id, err)
		} else if season != nil {
			seasonID = season.Id
		}
	}
	if seasonID > 0 {
		if err := seasonlogic.BumpSeasonLeaderboardCacheVersion(l.ctx, l.svcCtx, seasonID, match.GameType); err != nil {
			l.Logger.Errorf("失效赛季排行榜缓存失败: matchId=%d seasonId=%d gameType=%d err=%v", match.Id, seasonID, match.GameType, err)
		}
	}
}

func broadcastRankInfoUpdated(match *model.Match, result int) {
	if ws.GlobalHub == nil || match == nil || result == 3 || model.NormalizeMatchMode(match.MatchMode) != model.MatchModeRanked {
		return
	}

	userIds := map[int64]struct{}{match.UserId: struct{}{}}
	if match.OpponentId != nil && *match.OpponentId > 0 {
		userIds[*match.OpponentId] = struct{}{}
	}
	for userId := range userIds {
		if userId <= 0 {
			continue
		}
		ws.GlobalHub.SendToUser(userId, &ws.Message{
			Type: "rank_info_updated",
			Data: map[string]interface{}{
				"match_id":  match.Id,
				"game_type": match.GameType,
			},
		})
	}
}

func resolveCompletedByUserId(match *model.Match) int64 {
	if match == nil || match.Status != 2 || match.CompletedByUserId == nil {
		return 0
	}
	return *match.CompletedByUserId
}

func resolveCompletionSource(match *model.Match) string {
	if match == nil || match.Status != 2 || match.CompletionSource == "" {
		return model.CompletionSourceUnknown
	}
	return match.CompletionSource
}

type finishMatchSettlement struct {
	Result               int
	ServerRevision       int64
	CompetitiveRevisions map[int64]int64
	SeasonID             int64
}

func (l *FinishMatchLogic) settleMatchWithTx(tx *gorm.DB, match *model.Match, userId int64, req *types.FinishMatchReq) (finishMatchSettlement, error) {
	return l.settleMatchWithCompletionSourceTx(tx, match, userId, req, "")
}

func (l *FinishMatchLogic) settleMatchWithCompletionSourceTx(tx *gorm.DB, match *model.Match, userId int64, req *types.FinishMatchReq, completionSource string) (finishMatchSettlement, error) {
	if match == nil || match.Status != 1 || req == nil {
		return finishMatchSettlement{}, errFinishActionInvalid
	}

	result := 3
	if match.MyScore > match.OpponentScore {
		result = 1
	} else if match.MyScore < match.OpponentScore {
		result = 2
	}
	now := time.Now()
	match.Status = 2
	match.Result = &result
	match.EndTime = &now
	match.CompletedAt = &now
	clearFinishRequest(match)

	// 写入完成归因
	if match.CompletedByUserId == nil {
		completedBy := &userId
		source := completionSource
		if match.RefereeUserId != nil && *match.RefereeUserId > 0 {
			completedBy = match.RefereeUserId
			source = model.CompletionSourceReferee
		}
		if source == "" {
			source = model.CompletionSourcePlayerDirect
		}
		match.CompletedByUserId = completedBy
		match.CompletionSource = source
	}
	if match.GameType == 1 {
		match.CurrentFrameStarted = false
		match.CurrentFrameMyScore = 0
		match.CurrentFrameOpponentScore = 0
	}
	if req.Remark != "" {
		match.Remark = req.Remark
	}

	if isSnookerV2Match(match) {
		actions, err := l.svcCtx.MatchModel.ListActiveActionsWithTx(tx, match.Id)
		if err != nil {
			return finishMatchSettlement{}, err
		}
		countsByActor, err := calculateSnookerBreakAchievementCounts(actions)
		if err != nil {
			return finishMatchSettlement{}, err
		}
		for actor, counts := range countsByActor {
			for _, item := range []struct {
				achievementType string
				count           int
			}{
				{achievementType: "break_50", count: counts.FiftyPlus},
				{achievementType: "break_100", count: counts.Centuries},
				{achievementType: "break_147", count: counts.Break147},
			} {
				if item.count > 0 {
					if err := l.svcCtx.MatchModel.SaveAchievementWithTx(tx, match.Id, item.achievementType, item.count, actor); err != nil {
						return finishMatchSettlement{}, err
					}
				}
			}
		}
	}

	player1Win := result == 1
	player1RawAchievementScore := 0
	player2RawAchievementScore := 0
	completedRounds := 0
	if result != 3 && model.NormalizeMatchMode(match.MatchMode) == model.MatchModeRanked {
		rewardMap, err := l.getAchievementRewardMap(match.GameType)
		if err != nil {
			return finishMatchSettlement{}, err
		}
		rounds, err := l.svcCtx.MatchModel.ListCompletedRoundsWithTx(tx, match.Id)
		if err != nil {
			return finishMatchSettlement{}, err
		}
		completedRounds = len(rounds)
		actions, err := l.svcCtx.MatchModel.ListActiveActionsWithTx(tx, match.Id)
		if err != nil {
			return finishMatchSettlement{}, err
		}
		player1RawAchievementScore, player2RawAchievementScore, err = resolveReplayAchievementScoresStrict(
			match.GameType,
			rounds,
			actions,
			nil,
			rewardMap,
		)
		if err != nil {
			return finishMatchSettlement{}, err
		}
	}

	serverRevision, err := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, match)
	if err != nil {
		return finishMatchSettlement{}, err
	}
	if err := l.svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, &model.MatchAction{
		MatchId:        match.Id,
		RoundNo:        0,
		ActionType:     "match_end",
		Actor:          resolveFinishActionActor(match, userId),
		ScoreChange:    0,
		ClientActionId: stringPointer(req.ClientActionId),
		BaseRevision:   req.BaseRevision,
	}, serverRevision); err != nil {
		return finishMatchSettlement{}, err
	}

	if result == 3 || model.NormalizeMatchMode(match.MatchMode) == model.MatchModePractice {
		projection, err := projectCompetitiveReadModelWithTx(tx, l.svcCtx, logicx.CompetitiveProjectionInput{Match: match})
		if err != nil {
			return finishMatchSettlement{}, err
		}
		return finishMatchSettlement{
			Result:               result,
			ServerRevision:       serverRevision,
			CompetitiveRevisions: projection.CompetitiveRevisions,
			SeasonID:             projection.SeasonID,
		}, nil
	}

	settlementService := NewRankSettlementService(l.svcCtx.RankingModel)
	effectiveAt := resolveRankChangeEffectiveAt(match)
	player1Ranking, err := l.svcCtx.RankingModel.FindOrCreateWithTx(tx, match.UserId, match.GameType)
	if err != nil {
		return finishMatchSettlement{}, err
	}
	player1OpponentScore := -1
	var player2Ranking *model.UserRanking
	player2OpponentScore := -1
	if match.OpponentId != nil && *match.OpponentId > 0 {
		player2Ranking, err = l.svcCtx.RankingModel.FindOrCreateWithTx(tx, *match.OpponentId, match.GameType)
		if err != nil {
			return finishMatchSettlement{}, err
		}
		player1OpponentScore = player2Ranking.RankScore
		player2OpponentScore = player1Ranking.RankScore
	}

	player1Policy, err := buildRankSettlementPolicy(
		tx,
		l.svcCtx,
		match.UserId,
		match.OpponentId,
		match.GameType,
		effectiveAt,
		match.Id,
		completedRounds,
		player1OpponentScore,
	)
	if err != nil {
		return finishMatchSettlement{}, err
	}
	player1AchievementScore := 0
	if player1Policy.MemberActive {
		player1AchievementScore = calculateMemberAchievementRankingScoreWithPercent(player1RawAchievementScore, player1Win, true, player1Policy.MemberMultiplierPercent)
	} else if player1Policy.OrdinaryUserAchievementEnabled && player1Win {
		player1AchievementScore = player1RawAchievementScore
	}
	player1Settlement := settlementService.SettleWithPolicy(player1Ranking, player1Win, player1AchievementScore, player1Policy)
	applySettlementToRanking(player1Ranking, player1Settlement)
	if err := l.svcCtx.RankingModel.UpdateRankingSnapshot(tx, player1Ranking); err != nil {
		return finishMatchSettlement{}, err
	}

	player1BeforeProjection := &model.UserRanking{
		UserId: match.UserId, GameType: match.GameType,
		RankScore: player1Settlement.BeforeScore, RankLevel: player1Settlement.BeforeLevel,
	}
	player1AfterProjection := &model.UserRanking{
		UserId: match.UserId, GameType: match.GameType,
		RankScore: player1Settlement.AfterScore, RankLevel: player1Settlement.AfterLevel,
	}
	var player2BeforeProjection, player2AfterProjection *model.UserRanking
	changeLogs := []model.RankChangeLog{
		buildRankChangeLog(match.Id, match.UserId, match.GameType, resultLabel(player1Win, false), effectiveAt, player1Settlement),
	}
	if match.OpponentId != nil && *match.OpponentId > 0 {
		player2Policy, err := buildRankSettlementPolicy(
			tx,
			l.svcCtx,
			*match.OpponentId,
			&match.UserId,
			match.GameType,
			effectiveAt,
			match.Id,
			completedRounds,
			player2OpponentScore,
		)
		if err != nil {
			return finishMatchSettlement{}, err
		}
		player2AchievementScore := 0
		if player2Policy.MemberActive {
			player2AchievementScore = calculateMemberAchievementRankingScoreWithPercent(player2RawAchievementScore, !player1Win, true, player2Policy.MemberMultiplierPercent)
		} else if player2Policy.OrdinaryUserAchievementEnabled && !player1Win {
			player2AchievementScore = player2RawAchievementScore
		}
		player2Settlement := settlementService.SettleWithPolicy(player2Ranking, !player1Win, player2AchievementScore, player2Policy)
		player2BeforeProjection = &model.UserRanking{
			UserId: *match.OpponentId, GameType: match.GameType,
			RankScore: player2Settlement.BeforeScore, RankLevel: player2Settlement.BeforeLevel,
		}
		player2AfterProjection = &model.UserRanking{
			UserId: *match.OpponentId, GameType: match.GameType,
			RankScore: player2Settlement.AfterScore, RankLevel: player2Settlement.AfterLevel,
		}
		applySettlementToRanking(player2Ranking, player2Settlement)
		if err := l.svcCtx.RankingModel.UpdateRankingSnapshot(tx, player2Ranking); err != nil {
			return finishMatchSettlement{}, err
		}
		changeLogs = append(changeLogs, buildRankChangeLog(match.Id, *match.OpponentId, match.GameType, resultLabel(!player1Win, false), effectiveAt, player2Settlement))
	}
	if err := l.svcCtx.RankingModel.CreateRankChangeLogs(tx, changeLogs); err != nil {
		return finishMatchSettlement{}, err
	}
	projection, err := projectCompetitiveReadModelWithTx(tx, l.svcCtx, logicx.CompetitiveProjectionInput{
		Match:         match,
		Player1Before: player1BeforeProjection,
		Player1After:  player1AfterProjection,
		Player2Before: player2BeforeProjection,
		Player2After:  player2AfterProjection,
	})
	if err != nil {
		return finishMatchSettlement{}, err
	}
	return finishMatchSettlement{
		Result:               result,
		ServerRevision:       serverRevision,
		CompetitiveRevisions: projection.CompetitiveRevisions,
		SeasonID:             projection.SeasonID,
	}, nil
}

func projectCompetitiveReadModelWithTx(tx *gorm.DB, svcCtx *svc.ServiceContext, input logicx.CompetitiveProjectionInput) (logicx.CompetitiveProjectionResult, error) {
	if svcCtx == nil || svcCtx.CompetitiveReadModel == nil {
		return logicx.CompetitiveProjectionResult{
			AppliedUsers:         map[int64]bool{},
			CompetitiveRevisions: map[int64]int64{},
		}, nil
	}
	return logicx.NewCompetitiveProjector(svcCtx).ProjectWithTx(tx, input)
}

func (l *FinishMatchLogic) getAchievementRewardMap(gameType int) (map[string]int, error) {
	config, err := logicx.NewMemberRightsConfigService(l.svcCtx).GetConfig()
	if err == nil {
		return buildAchievementRewardMapFromRightsRules(config.RankingRights), nil
	}

	configs, err := l.svcCtx.RankingModel.GetAchievementRewardConfigs(gameType)
	if err != nil {
		return nil, err
	}

	rewardMap := make(map[string]int, len(configs))
	for _, item := range configs {
		rewardMap[item.AchievementType] = item.RewardScore
	}
	return rewardMap, nil
}

func (l *FinishMatchLogic) awardMemberGrowthForMatch(matchId, userId int64, opponentId *int64) {
	memberGrowthService := NewMemberGrowthService(l.svcCtx, nil)
	if growthResult, growthErr := memberGrowthService.AwardCompletedMatch(userId, matchId); growthErr != nil {
		l.Logger.Errorf("发放创建者会员成长失败: matchId=%d, userId=%d, err=%v", matchId, userId, growthErr)
	} else {
		l.Logger.Infof("创建者会员成长结算完成: matchId=%d, userId=%d, granted=%v, reason=%s, growthPoints=%d",
			matchId, userId, growthResult.Granted, growthResult.Reason, growthResult.GrowthPoints)
	}

	if opponentId != nil && *opponentId > 0 {
		if growthResult, growthErr := memberGrowthService.AwardCompletedMatch(*opponentId, matchId); growthErr != nil {
			l.Logger.Errorf("发放对手会员成长失败: matchId=%d, userId=%d, err=%v", matchId, *opponentId, growthErr)
		} else {
			l.Logger.Infof("对手会员成长结算完成: matchId=%d, userId=%d, granted=%v, reason=%s, growthPoints=%d",
				matchId, *opponentId, growthResult.Granted, growthResult.Reason, growthResult.GrowthPoints)
		}
	}
}

func (l *FinishMatchLogic) syncAchievementProgressForCompletedMatch(match *model.Match) {
	if err := l.syncAchievementProgressForCompletedMatchWithError(match); err != nil {
		l.Logger.Errorf("同步对局成就进度失败: matchId=%d, err=%v", match.Id, err)
	}
}

func (l *FinishMatchLogic) syncAchievementProgressForCompletedMatchWithError(match *model.Match) error {
	if match == nil || match.Status != 2 {
		return nil
	}
	if match.AchievementSyncedAt != nil {
		return nil
	}
	if l.svcCtx == nil || l.svcCtx.DB == nil || l.svcCtx.MatchModel == nil {
		return fmt.Errorf("achievement sync infrastructure is unavailable")
	}
	if match.Result == nil {
		return fmt.Errorf("completed match %d has no result", match.Id)
	}
	if model.NormalizeMatchMode(match.MatchMode) == model.MatchModePractice || *match.Result == 3 ||
		match.OpponentId == nil || *match.OpponentId <= 0 || *match.OpponentId == match.UserId {
		return l.markAchievementSyncComplete(match)
	}
	if l.svcCtx.RankingModel == nil || l.svcCtx.AchievementProgressEventModel == nil {
		return fmt.Errorf("achievement sync infrastructure is unavailable")
	}

	roundCount, err := l.svcCtx.MatchModel.GetRoundCount(match.Id)
	if err != nil {
		return err
	}
	if roundCount == 0 {
		return l.markAchievementSyncComplete(match)
	}

	player1ID := match.UserId
	player2ID := *match.OpponentId
	playerIDs := []int64{player1ID, player2ID}
	userByActor := map[int]int64{1: player1ID, 2: player2ID}
	progressService := achievementx.NewAchievementProgressService(l.svcCtx)

	for _, playerID := range playerIDs {
		if err := appendMatchAchievementProgressEvent(progressService, playerID, match, achievementx.MetricMatchesTotal, 1); err != nil {
			return err
		}
	}

	winnerID := player1ID
	if *match.Result == 2 {
		winnerID = player2ID
	}
	if err := appendMatchAchievementProgressEvent(progressService, winnerID, match, achievementx.MetricWinsTotal, 1); err != nil {
		return err
	}

	for _, playerID := range playerIDs {
		ranking, err := l.svcCtx.RankingModel.FindOrCreateByGameType(playerID, match.GameType)
		if err != nil {
			return err
		}
		if err := appendMatchAchievementProgressEvent(progressService, playerID, match, achievementx.MetricMaxWinStreak, ranking.MaxStreak); err != nil {
			return err
		}
	}

	records, err := l.svcCtx.MatchModel.GetAchievements(match.Id)
	if err != nil {
		return err
	}
	for _, record := range records {
		metricKey := metricKeyFromMatchAchievement(record.AchievementType)
		playerID, ok := userByActor[record.Actor]
		if !ok || metricKey == "" || record.Count <= 0 {
			continue
		}
		if err := appendMatchAchievementProgressEvent(progressService, playerID, match, metricKey, record.Count); err != nil {
			return err
		}
	}

	for _, playerID := range playerIDs {
		if _, err := progressService.RefreshUserAchievementsWithSource(playerID, achievementx.SourceTypeMatch, match.Id); err != nil {
			return err
		}
	}
	return l.markAchievementSyncComplete(match)
}

func (l *FinishMatchLogic) markAchievementSyncComplete(match *model.Match) error {
	now := time.Now()
	if err := l.svcCtx.MatchModel.MarkAchievementSynced(match.Id, now); err != nil {
		return err
	}
	match.AchievementSyncedAt = &now
	return nil
}

func appendMatchAchievementProgressEvent(service *achievementx.AchievementProgressService, userId int64, match *model.Match, metricKey string, metricValue int) error {
	_, err := service.AppendEvent(achievementx.AchievementProgressEventInput{
		UserId:      userId,
		SourceType:  achievementx.SourceTypeMatch,
		SourceId:    match.Id,
		GameType:    match.GameType,
		MetricKey:   metricKey,
		MetricValue: metricValue,
		OccurredAt:  resolveRankChangeEffectiveAt(match),
	})
	return err
}

func metricKeyFromMatchAchievement(achievementType string) string {
	switch normalizeAchievementType(achievementType) {
	case "break_and_run":
		return achievementx.MetricBreakClearTotal
	case "run_out":
		return achievementx.MetricContinueClearTotal
	case "golden_break":
		return achievementx.MetricGoldenBreakTotal
	case "nine_on_break":
		return achievementx.MetricNineOnBreakTotal
	case "break_50":
		return achievementx.MetricBreak50Total
	case "break_100":
		return achievementx.MetricBreak100Total
	case "break_147":
		return achievementx.MetricBreak147Total
	default:
		return ""
	}
}

func normalizeAchievementType(achievementType string) string {
	switch achievementType {
	case "break_clear":
		return "break_and_run"
	case "continue_clear":
		return "run_out"
	case "small_gold":
		return "golden_break"
	case "big_gold":
		return "nine_on_break"
	default:
		return achievementType
	}
}

func calculateAchievementScoresByActor(rounds []model.MatchRound, rewardMap map[string]int) (int, int) {
	actor1 := 0
	actor2 := 0

	for _, round := range rounds {
		if round.Winner == nil {
			continue
		}
		normalized := normalizeAchievementType(round.WinType)
		if normalized == "" || normalized == "normal" || normalized == "start" {
			continue
		}
		reward := rewardMap[normalized]
		if reward <= 0 {
			continue
		}

		if *round.Winner == 1 {
			actor1 += reward
		} else if *round.Winner == 2 {
			actor2 += reward
		}
	}

	return actor1, actor2
}

func applySettlementToRanking(ranking *model.UserRanking, settlement RankSettlementResult) {
	ranking.RankScore = settlement.AfterScore
	ranking.RankLevel = settlement.AfterLevel
	ranking.TotalWins = settlement.TotalWins
	ranking.TotalLosses = settlement.TotalLosses
	ranking.CurrentStreak = settlement.CurrentStreak
	ranking.MaxStreak = settlement.MaxStreak
}

func resolveRankChangeEffectiveAt(match *model.Match) time.Time {
	if match == nil {
		return time.Now()
	}
	if match.EndTime != nil && !match.EndTime.IsZero() {
		return *match.EndTime
	}
	if !match.MatchTime.IsZero() {
		return match.MatchTime
	}
	return time.Now()
}

func buildRankChangeLog(matchId, userId int64, gameType int, result string, effectiveAt time.Time, settlement RankSettlementResult) model.RankChangeLog {
	return model.RankChangeLog{
		UserId:           userId,
		MatchId:          matchId,
		ChangeType:       "match_result",
		GameType:         gameType,
		Result:           result,
		BaseScore:        settlement.BaseScore,
		AchievementScore: settlement.AchievementScore,
		FinalChange:      settlement.FinalChange,
		BeforeScore:      settlement.BeforeScore,
		AfterScore:       settlement.AfterScore,
		BeforeLevel:      settlement.BeforeLevel,
		AfterLevel:       settlement.AfterLevel,
		Remark:           buildRankSettlementRemark(settlement),
		EffectiveAt:      effectiveAt,
	}
}

func resultLabel(isWin bool, isDraw bool) string {
	if isDraw {
		return "draw"
	}
	if isWin {
		return "win"
	}
	return "lose"
}
