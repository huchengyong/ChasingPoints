package match

import (
	"context"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMatchDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取对局详情
func NewGetMatchDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchDetailLogic {
	return &GetMatchDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetMatchDetailLogic) GetMatchDetail(req *types.GetMatchDetailReq) (resp *types.GetMatchDetailResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMatchDetailResp{Success: false}, nil
	}

	// 查询对局
	match, err := l.svcCtx.MatchModel.FindById(req.MatchId)
	if err != nil || match == nil {
		l.Logger.Errorf("对局不存在: %v", err)
		return &types.GetMatchDetailResp{Success: false}, nil
	}
	match = effectiveMatchForRead(match, time.Now())

	// 验证用户权限（对局双方和裁判都可以查看）
	capabilities := resolveMatchViewerCapabilities(match, userId)
	if capabilities.ViewerRole == matchViewerRoleUnknown {
		l.Logger.Errorf("用户不是对局参与者: matchUserId=%d, matchOpponentId=%v, refereeUserId=%v, currentUserId=%d",
			match.UserId, match.OpponentId, match.RefereeUserId, userId)
		return &types.GetMatchDetailResp{Success: false}, nil
	}
	isPlayer1 := capabilities.ViewerRole != matchViewerRolePlayer2
	var completedCore *logicx.CompletedMatchCoreSummary
	if match.Status == 2 {
		completedCore, _ = logicx.BuildCompletedMatchCoreSummary(l.ctx, l.svcCtx, match)
	}

	// 根据视角获取分数和玩家信息
	var myScore, opponentScore int
	var currentFrameMyScore, currentFrameOpponentScore int
	var myName, opponentName, myAvatar, opponentAvatar string

	player1Name := "玩家1"
	player1Avatar := ""
	if completedCore != nil && completedCore.Player1 != nil {
		player1Name = completedCore.Player1.Nickname
		player1Avatar = completedCore.Player1.Avatar
	} else if player1, _ := l.svcCtx.UserModel.FindById(match.UserId); player1 != nil {
		player1Name = player1.Nickname
		player1Avatar = player1.Avatar
	}

	player2Name := match.OpponentName
	player2Avatar := ""
	if completedCore != nil && completedCore.Player2 != nil {
		player2Name = completedCore.Player2.Nickname
		player2Avatar = completedCore.Player2.Avatar
	} else if match.OpponentId != nil {
		if player2, _ := l.svcCtx.UserModel.FindById(*match.OpponentId); player2 != nil {
			player2Name = player2.Nickname
			player2Avatar = player2.Avatar
		}
	}
	refereeName := ""
	refereeAvatar := ""
	refereeJoinedAt := ""
	if completedCore != nil && completedCore.Referee != nil {
		refereeName = completedCore.Referee.Nickname
		refereeAvatar = completedCore.Referee.Avatar
	} else if capabilities.RefereeBound && capabilities.RefereeUserId > 0 {
		if referee, err := l.svcCtx.UserModel.FindById(capabilities.RefereeUserId); err == nil && referee != nil {
			refereeName = referee.Nickname
			refereeAvatar = referee.Avatar
		}
	}

	if match.RefereeJoinedAt != nil {
		refereeJoinedAt = match.RefereeJoinedAt.Format("2006-01-02T15:04:05+08:00")
	}

	refereeDurationSeconds := int64(0)
	if match.RefereeJoinedAt != nil {
		if match.EndTime != nil {
			refereeDurationSeconds = int64(match.EndTime.Sub(*match.RefereeJoinedAt).Seconds())
		} else {
			refereeDurationSeconds = int64(time.Since(*match.RefereeJoinedAt).Seconds())
		}
		if refereeDurationSeconds < 0 {
			refereeDurationSeconds = 0
		}
	}

	completedByUserId := resolveCompletedByUserId(match)

	if capabilities.ViewerRole == matchViewerRoleReferee || isPlayer1 {
		// 当前用户是创建者，使用原始视角
		myScore = match.MyScore
		opponentScore = match.OpponentScore
		currentFrameMyScore = match.CurrentFrameMyScore
		currentFrameOpponentScore = match.CurrentFrameOpponentScore
		myName = player1Name
		myAvatar = player1Avatar
		opponentName = player2Name
		opponentAvatar = player2Avatar
	} else {
		// 当前用户是对手，交换视角
		myScore = match.OpponentScore
		opponentScore = match.MyScore
		currentFrameMyScore = match.CurrentFrameOpponentScore
		currentFrameOpponentScore = match.CurrentFrameMyScore
		myName = player2Name
		myAvatar = player2Avatar
		opponentName = player1Name
		opponentAvatar = player1Avatar
	}
	if completedCore != nil {
		viewer := completedCore.ViewerPerspective(userId)
		myScore, opponentScore = viewer.MyScore, viewer.OpponentScore
		myName, myAvatar = viewer.MyName, viewer.MyAvatar
		opponentName, opponentAvatar = viewer.OpponentName, viewer.OpponentAvatar
	}
	player2Id := int64(0)
	if match.OpponentId != nil {
		player2Id = *match.OpponentId
	}
	opponentId := int64(0)
	if capabilities.ViewerRole == matchViewerRolePlayer1 {
		opponentId = player2Id
	} else if capabilities.ViewerRole == matchViewerRolePlayer2 {
		opponentId = match.UserId
	}

	achievements := types.MatchAchievement{}
	if completedCore != nil {
		achievements = completedCore.Achievements
	} else if achList, err := l.svcCtx.MatchModel.GetAchievements(match.Id); err == nil {
		achievements = buildMatchAchievementPayload(achList)
	}

	player1Profile, _ := logicx.LoadCurrentCompetitiveProfile(l.svcCtx, match.UserId, match.GameType)
	player2Profile := logicx.CurrentCompetitiveProfile{}
	if match.OpponentId != nil {
		player2Profile, _ = logicx.LoadCurrentCompetitiveProfile(l.svcCtx, *match.OpponentId, match.GameType)
	}
	player1WinRate, player1MaxScore := player1Profile.WinRate, player1Profile.MaxScore
	player2WinRate, player2MaxScore := player2Profile.WinRate, player2Profile.MaxScore

	// 根据视角设置我方和对手的统计数据
	var myWinRate, opponentWinRate float64
	var myMaxScore, opponentMaxScore int
	if isPlayer1 {
		myWinRate = player1WinRate
		myMaxScore = player1MaxScore
		opponentWinRate = player2WinRate
		opponentMaxScore = player2MaxScore
	} else {
		myWinRate = player2WinRate
		myMaxScore = player2MaxScore
		opponentWinRate = player1WinRate
		opponentMaxScore = player1MaxScore
	}

	myRankChange := 0
	opponentRankChange := 0
	myRankDetails := []types.RankDetail{}

	if match.Status == 2 {
		if completedCore != nil {
			for index := range completedCore.RankChanges {
				rankLog := &completedCore.RankChanges[index]
				if rankLog.UserId == userId {
					myRankChange = rankLog.FinalChange
					myRankDetails = rankLogToDetails(rankLog)
				} else {
					opponentRankChange = rankLog.FinalChange
				}
			}
		} else {
			myRankLog, logErr := l.svcCtx.RankingModel.FindMatchRankChangeByUserAndGameType(match.Id, userId, match.GameType)
			if logErr != nil {
				l.Logger.Errorf("获取我的段位明细失败: matchId=%d userId=%d err=%v", match.Id, userId, logErr)
			} else if myRankLog != nil {
				myRankChange = myRankLog.FinalChange
				myRankDetails = rankLogToDetails(myRankLog)
			}

			allLogs, logsErr := l.svcCtx.RankingModel.ListRankChangesByMatchAndGameType(match.Id, match.GameType)
			if logsErr != nil {
				l.Logger.Errorf("获取对局段位明细失败: matchId=%d err=%v", match.Id, logsErr)
			} else if opponentLog := findOpponentRankLog(allLogs, userId); opponentLog != nil {
				opponentRankChange = opponentLog.FinalChange
			}
		}
	}

	var roundCount int64
	if completedCore != nil {
		roundCount = int64(len(completedCore.Rounds))
	} else {
		roundCount, _ = l.svcCtx.MatchModel.GetRoundCount(match.Id)
	}
	currentRound := int(roundCount) + 1
	redBallCount := 0
	snookerClearanceStarted := false
	snookerClearedColors := make([]int, 0, 6)
	snookerExpectedClearanceScore := 0
	snookerClearanceCompleted := false
	var completedRounds []model.MatchRound
	var actions []model.MatchAction
	if completedCore != nil {
		completedRounds = completedCore.Rounds
		actions = completedCore.Actions
	} else {
		completedRounds, _ = l.svcCtx.MatchModel.ListCompletedRounds(match.Id)
		actions, _ = l.svcCtx.MatchModel.ListActiveActions(match.Id)
	}
	actionsForSummary := actions
	var lastAction *types.MatchLastAction
	if completedCore != nil && len(actions) > 0 {
		lastAction = buildMatchLastActionFromAction(userId, match, &actions[len(actions)-1])
	} else {
		lastAction = buildMatchLastAction(l.svcCtx, userId, match)
	}
	snookerState := model.SnookerRoundState{ClearedColors: make([]int, 0, 6)}
	if match.GameType == 1 {
		stateRound := currentRound
		if !match.CurrentFrameStarted && roundCount > 0 {
			stateRound = int(roundCount)
		}
		var state model.SnookerRoundState
		var stateErr error
		if completedCore != nil {
			state, stateErr = buildSnookerStateForMatchFromActions(match, actions, stateRound)
		} else {
			state, stateErr = loadSnookerStateForMatch(l.svcCtx, match, stateRound)
		}
		if stateErr == nil {
			snookerState = state
		}
		redBallCount = snookerState.RedBallCount
		snookerClearanceStarted = snookerState.ClearanceStarted
		snookerClearedColors = snookerState.ClearedColors
		snookerExpectedClearanceScore = snookerState.ExpectedClearanceScore
		snookerClearanceCompleted = snookerState.ClearanceCompleted
		if !match.CurrentFrameStarted {
			actionsForSummary = filterSnookerActionsToCompletedRounds(actions, completedRounds)
			if match.SnookerRulesVersion != model.SnookerRulesVersionWPBSA {
				redBallCount = 0
				snookerClearanceStarted = false
				snookerClearedColors = make([]int, 0, 6)
				snookerExpectedClearanceScore = 0
				snookerClearanceCompleted = false
			}
		}
	}

	myActor := 1
	if !isPlayer1 {
		myActor = 2
	}
	summaryHighlights, summaryStats := buildMatchSummary(
		match.GameType,
		myActor,
		myScore,
		opponentScore,
		myWinRate,
		opponentWinRate,
		myMaxScore,
		opponentMaxScore,
		redBallCount,
		match.CreatedAt.Format("2006-01-02 15:04:05"),
		completedRounds,
		actionsForSummary,
		achievements,
	)

	l.Logger.Infof("用户 %d 获取对局 %d 详情, isPlayer1=%v", userId, match.Id, isPlayer1)
	snookerFormat, snookerTargetWins := normalizedSnookerFormat(match)
	matchFormat, targetWins := normalizedPoolMatchFormat(match)

	return &types.GetMatchDetailResp{
		Success: true,
		Match: types.MatchDetailData{
			Id:                            match.Id,
			Player1Id:                     match.UserId,
			Player2Id:                     player2Id,
			OpponentId:                    opponentId,
			GameType:                      match.GameType,
			MatchMode:                     model.NormalizeMatchMode(match.MatchMode),
			Visibility:                    model.NormalizeMatchVisibility(match.Visibility, match.MatchMode),
			FinishState:                   model.NormalizeFinishState(match.FinishState),
			FinishRequestedBy:             resolveFinishRequestedBy(match),
			Status:                        match.Status,
			ServerRevision:                match.SyncRevision,
			IsPlayer1:                     isPlayer1,
			ViewerRole:                    capabilities.ViewerRole,
			RefereeBound:                  capabilities.RefereeBound,
			RefereeUserId:                 capabilities.RefereeUserId,
			RefereeName:                   refereeName,
			RefereeAvatar:                 refereeAvatar,
			RefereeJoinedAt:               refereeJoinedAt,
			RefereeDurationSeconds:        refereeDurationSeconds,
			CompletedByUserId:             completedByUserId,
			CompletionSource:              resolveCompletionSource(match),
			CanScore:                      capabilities.CanScore,
			CanUndo:                       capabilities.CanUndo,
			CanFinish:                     capabilities.CanFinish,
			CanRequestFinish:              capabilities.CanRequestFinish,
			CanConfirmFinish:              capabilities.CanConfirmFinish,
			CanDisputeFinish:              capabilities.CanDisputeFinish,
			CanWithdrawFinish:             capabilities.CanWithdrawFinish,
			LastAction:                    lastAction,
			MyScore:                       myScore,
			OpponentScore:                 opponentScore,
			MyName:                        myName,
			OpponentName:                  opponentName,
			MyAvatar:                      myAvatar,
			OpponentAvatar:                opponentAvatar,
			MyWinRate:                     myWinRate,
			OpponentWinRate:               opponentWinRate,
			MyMaxScore:                    myMaxScore,
			OpponentMaxScore:              opponentMaxScore,
			MyRankChange:                  myRankChange,
			OpponentRankChange:            opponentRankChange,
			MyRankDetails:                 myRankDetails,
			CurrentFrameStarted:           match.CurrentFrameStarted,
			CurrentFrameMyScore:           currentFrameMyScore,
			CurrentFrameOpponentScore:     currentFrameOpponentScore,
			RedBallCount:                  redBallCount,
			SnookerClearanceStarted:       snookerClearanceStarted,
			SnookerClearedColors:          snookerClearedColors,
			SnookerExpectedClearanceScore: snookerExpectedClearanceScore,
			SnookerClearanceCompleted:     snookerClearanceCompleted,
			SnookerRulesVersion:           match.SnookerRulesVersion,
			BestOfFrames:                  match.BestOfFrames,
			SnookerFormat:                 snookerFormat,
			SnookerTargetWins:             snookerTargetWins,
			CanChangeSnookerFormat:        canChangeSnookerFormat(l.svcCtx, userId, match),
			MatchFormat:                   matchFormat,
			TargetWins:                    targetWins,
			CanChangeMatchFormat:          canChangeMatchFormat(l.svcCtx, userId, match),
			StartingActor:                 match.StartingActor,
			SnookerPhase:                  snookerState.Phase,
			SnookerBallOn:                 snookerState.BallOn,
			SnookerStriker:                snookerState.Striker,
			SnookerVisitNo:                snookerState.VisitNo,
			SnookerCurrentBreak:           snookerState.CurrentBreak,
			SnookerRedsRemaining:          snookerState.RedsRemaining,
			SnookerFreeBallAvailable:      snookerState.FreeBallAvailable,
			SnookerCueBallInHand:          snookerState.CueBallInHand,
			SnookerMissWarningActive:      snookerState.MissWarningActive,
			SnookerRespottedBlackPending:  snookerState.Phase == model.SnookerPhaseRespottedBlackPending,
			SnookerPendingConcessionActor: snookerState.PendingConcessionActor,
			SnookerPendingConcessionScope: snookerState.PendingConcessionScope,
			SnookerFrameEndReason:         snookerState.FrameEndReason,
			SummaryHighlights:             summaryHighlights,
			SummaryStats:                  summaryStats,
			Achievements:                  achievements,
			CreatedAt:                     match.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}
