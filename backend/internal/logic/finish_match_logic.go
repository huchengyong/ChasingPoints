package logic

import (
	"context"
	"fmt"
	"time"

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
		svcCtx: svcCtx,
	}
}

func (l *FinishMatchLogic) FinishMatch(req *types.FinishMatchReq) (resp *types.FinishMatchResp, err error) {
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

	// 验证用户权限（双方都可以结束对局）
	isPlayer1 := match.UserId == userId
	isPlayer2 := match.OpponentId != nil && *match.OpponentId == userId
	if !isPlayer1 && !isPlayer2 {
		l.Logger.Errorf("用户不是对局参与者: matchUserId=%d, matchOpponentId=%v, currentUserId=%d",
			match.UserId, match.OpponentId, userId)
		return &types.FinishMatchResp{Success: false}, nil
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

	// 验证对局状态
	if match.Status != 1 {
		return &types.FinishMatchResp{Success: false}, nil
	}

	// 计算比赛结果（从创建者视角）
	var result int
	if match.MyScore > match.OpponentScore {
		result = 1 // 胜利
	} else if match.MyScore < match.OpponentScore {
		result = 2 // 失败
	} else {
		result = 3 // 平局
	}

	// 更新对局状态
	now := time.Now()
	match.Status = 2 // 已完成
	match.Result = &result
	match.EndTime = &now
	if match.GameType == 1 {
		match.CurrentFrameStarted = false
		match.CurrentFrameMyScore = 0
		match.CurrentFrameOpponentScore = 0
	}
	if req.Remark != "" {
		match.Remark = req.Remark
	}

	settlementService := NewRankSettlementService(l.svcCtx.RankingModel)
	player1Win := result == 1
	player1AchievementScore := 0
	player2AchievementScore := 0
	if result != 3 {
		rewardMap, rewardErr := l.getAchievementRewardMap(match.GameType)
		if rewardErr != nil {
			l.Logger.Errorf("获取成就奖励配置失败: %v", rewardErr)
			return &types.FinishMatchResp{Success: false}, nil
		}
		rounds, roundsErr := l.svcCtx.MatchModel.ListCompletedRounds(match.Id)
		if roundsErr != nil {
			l.Logger.Errorf("获取对局局记录失败: %v", roundsErr)
			return &types.FinishMatchResp{Success: false}, nil
		}
		actions, actionsErr := l.svcCtx.MatchModel.ListActiveActions(match.Id)
		if actionsErr != nil {
			l.Logger.Errorf("获取对局操作记录失败: %v", actionsErr)
			return &types.FinishMatchResp{Success: false}, nil
		}
		player1AchievementScore, player2AchievementScore = resolveReplayAchievementScores(
			match.GameType,
			rounds,
			actions,
			nil,
			rewardMap,
		)
	}

	actionActor := 1
	if !isPlayer1 {
		actionActor = 2
	}
	action := &model.MatchAction{
		MatchId:        match.Id,
		RoundNo:        0,
		ActionType:     "match_end",
		Actor:          actionActor,
		ScoreChange:    0,
		ClientActionId: stringPointer(req.ClientActionId),
		BaseRevision:   req.BaseRevision,
	}
	var serverRevision int64
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		revision, revisionErr := l.svcCtx.MatchModel.BumpMatchRevisionWithTx(tx, match)
		if revisionErr != nil {
			return revisionErr
		}
		serverRevision = revision
		if err := l.svcCtx.MatchModel.CreateActionWithRevisionWithTx(tx, action, serverRevision); err != nil {
			return err
		}

		if result == 3 {
			return nil
		}

		effectiveAt := resolveRankChangeEffectiveAt(match)
		player1Policy, err := buildRankSettlementPolicy(tx, l.svcCtx, match.UserId, match.OpponentId, match.GameType, effectiveAt, match.Id)
		if err != nil {
			return err
		}

		player1Ranking, err := l.svcCtx.RankingModel.FindOrCreateWithTx(tx, match.UserId, match.GameType)
		if err != nil {
			return err
		}
		player1Settlement := settlementService.SettleWithPolicy(player1Ranking, player1Win, player1AchievementScore, player1Policy)
		applySettlementToRanking(player1Ranking, player1Settlement)
		if err := l.svcCtx.RankingModel.UpdateRankingSnapshot(tx, player1Ranking); err != nil {
			return err
		}

		changeLogs := []model.RankChangeLog{
			buildRankChangeLog(match.Id, match.UserId, match.GameType, resultLabel(player1Win, false), effectiveAt, player1Settlement),
		}

		if match.OpponentId != nil && *match.OpponentId > 0 {
			player2Policy, err := buildRankSettlementPolicy(tx, l.svcCtx, *match.OpponentId, &match.UserId, match.GameType, effectiveAt, match.Id)
			if err != nil {
				return err
			}
			player2Ranking, err := l.svcCtx.RankingModel.FindOrCreateWithTx(tx, *match.OpponentId, match.GameType)
			if err != nil {
				return err
			}
			player2Settlement := settlementService.SettleWithPolicy(player2Ranking, !player1Win, player2AchievementScore, player2Policy)
			applySettlementToRanking(player2Ranking, player2Settlement)
			if err := l.svcCtx.RankingModel.UpdateRankingSnapshot(tx, player2Ranking); err != nil {
				return err
			}
			changeLogs = append(changeLogs, buildRankChangeLog(match.Id, *match.OpponentId, match.GameType, resultLabel(!player1Win, false), effectiveAt, player2Settlement))
		}

		return l.svcCtx.RankingModel.CreateRankChangeLogs(tx, changeLogs)
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
	view, stateErr := loadMatchWriteState(l.svcCtx, userId, match)
	if stateErr != nil {
		l.Logger.Errorf("加载写入快照失败: matchId=%d, clientActionId=%s, err=%v", match.Id, req.ClientActionId, stateErr)
		return &types.FinishMatchResp{Success: false, Accepted: false}, nil
	}

	l.Logger.Infof("用户 %d 结束对局 %d，比分: %d:%d，结果: %d",
		userId, match.Id, match.MyScore, match.OpponentScore, result)

	// ========== 通知双方对局结果 + 推送 ==========
	if result != 3 {
		resultText := "胜利"
		if result == 2 {
			resultText = "失败"
		}
		_ = l.svcCtx.NotificationModel.Create(&model.Notification{
			UserId:  match.UserId,
			Type:    "match_result",
			Title:   "对局已结束",
			Content: fmt.Sprintf("你的对局已结束，结果：%s（%d:%d）", resultText, match.MyScore, match.OpponentScore),
			Data:    buildNotificationPayload("/subPages/match/matchResult", match.Id, 0),
			IsRead:  0,
		})
		if p1, pushErr := l.svcCtx.UserModel.FindById(match.UserId); pushErr == nil && p1 != nil && p1.PushToken != "" {
			l.svcCtx.PushService.SendPush(p1.PushToken, "对局已结束", fmt.Sprintf("结果：%s（%d:%d）", resultText, match.MyScore, match.OpponentScore), map[string]interface{}{
				"url": fmt.Sprintf("/subPages/match/matchResult?match_id=%d", match.Id),
			})
		}
		if match.OpponentId != nil && *match.OpponentId > 0 {
			opResultText := "胜利"
			if result == 1 {
				opResultText = "失败"
			}
			_ = l.svcCtx.NotificationModel.Create(&model.Notification{
				UserId:  *match.OpponentId,
				Type:    "match_result",
				Title:   "对局已结束",
				Content: fmt.Sprintf("你的对局已结束，结果：%s（%d:%d）", opResultText, match.OpponentScore, match.MyScore),
				Data:    buildNotificationPayload("/subPages/match/matchResult", match.Id, 0),
				IsRead:  0,
			})
			if p2, pushErr := l.svcCtx.UserModel.FindById(*match.OpponentId); pushErr == nil && p2 != nil && p2.PushToken != "" {
				l.svcCtx.PushService.SendPush(p2.PushToken, "对局已结束", fmt.Sprintf("结果：%s（%d:%d）", opResultText, match.OpponentScore, match.MyScore), map[string]interface{}{
					"url": fmt.Sprintf("/subPages/match/matchResult?match_id=%d", match.Id),
				})
			}
		}
	}

	// 推送 WebSocket 消息通知双方对局已结束
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

func (l *FinishMatchLogic) getAchievementRewardMap(gameType int) (map[string]int, error) {
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
