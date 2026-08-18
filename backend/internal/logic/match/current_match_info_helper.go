package match

import (
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func buildCurrentMatchInfo(svcCtx *svc.ServiceContext, userId int64, match *model.Match) *types.CurrentMatchInfo {
	return buildCurrentMatchInfoWithKnownUser(svcCtx, userId, match, nil)
}

func BuildCurrentMatchInfoWithKnownUser(svcCtx *svc.ServiceContext, userId int64, match *model.Match, knownUser *model.User) *types.CurrentMatchInfo {
	return buildCurrentMatchInfoWithKnownUser(svcCtx, userId, match, knownUser)
}

func buildCurrentMatchInfoWithKnownUser(svcCtx *svc.ServiceContext, userId int64, match *model.Match, knownUser *model.User) *types.CurrentMatchInfo {
	if match == nil {
		return nil
	}

	roundCount := int64(0)
	if svcCtx != nil && svcCtx.MatchModel != nil {
		roundCount, _ = svcCtx.MatchModel.GetRoundCount(match.Id)
	}
	snookerState := model.SnookerRoundState{ClearedColors: make([]int, 0, 6)}
	if match.GameType == 1 {
		roundNo := int(roundCount) + 1
		if !match.CurrentFrameStarted && roundCount > 0 {
			roundNo = int(roundCount)
		}
		if state, err := loadSnookerStateForMatch(svcCtx, match, roundNo); err == nil {
			snookerState = state
		}
	}

	capabilities := resolveMatchViewerCapabilities(match, userId)
	isPlayer1 := !shouldUsePlayer2Perspective(capabilities.ViewerRole)
	myScore := match.MyScore
	opponentScore := match.OpponentScore
	currentFrameMyScore := match.CurrentFrameMyScore
	currentFrameOpponentScore := match.CurrentFrameOpponentScore
	player1Name := "玩家1"
	player1Avatar := ""
	player2Id := int64(0)
	player2Name := match.OpponentName
	player2Avatar := ""
	if player2Name == "" {
		player2Name = "玩家2"
	}
	if knownUser != nil && knownUser.Id == match.UserId {
		if knownUser.Nickname != "" {
			player1Name = knownUser.Nickname
		}
		player1Avatar = knownUser.Avatar
	} else if svcCtx != nil && svcCtx.UserModel != nil {
		if player1, err := svcCtx.UserModel.FindById(match.UserId); err == nil && player1 != nil {
			if player1.Nickname != "" {
				player1Name = player1.Nickname
			}
			player1Avatar = player1.Avatar
		}
	}
	if match.OpponentId != nil {
		player2Id = *match.OpponentId
		if knownUser != nil && knownUser.Id == *match.OpponentId {
			if knownUser.Nickname != "" {
				player2Name = knownUser.Nickname
			}
			player2Avatar = knownUser.Avatar
		} else if svcCtx != nil && svcCtx.UserModel != nil {
			if player2, err := svcCtx.UserModel.FindById(*match.OpponentId); err == nil && player2 != nil {
				if player2.Nickname != "" {
					player2Name = player2.Nickname
				}
				player2Avatar = player2.Avatar
			}
		}
	}
	opponentId := player2Id
	opponentName := player2Name
	opponentAvatar := player2Avatar
	refereeName := ""
	refereeAvatar := ""
	refereeJoinedAt := ""

	if capabilities.RefereeBound && knownUser != nil && knownUser.Id == capabilities.RefereeUserId {
		refereeName = knownUser.Nickname
		refereeAvatar = knownUser.Avatar
	} else if capabilities.RefereeBound && svcCtx != nil && svcCtx.UserModel != nil {
		if referee, err := svcCtx.UserModel.FindById(capabilities.RefereeUserId); err == nil && referee != nil {
			refereeName = referee.Nickname
			refereeAvatar = referee.Avatar
		}
	}

	if match.RefereeJoinedAt != nil {
		refereeJoinedAt = match.RefereeJoinedAt.Format("2006-01-02T15:04:05+08:00")
	}

	if capabilities.ViewerRole != matchViewerRoleReferee && !isPlayer1 {
		myScore = match.OpponentScore
		opponentScore = match.MyScore
		currentFrameMyScore = match.CurrentFrameOpponentScore
		currentFrameOpponentScore = match.CurrentFrameMyScore
		opponentId = match.UserId
		opponentName = player1Name
		opponentAvatar = player1Avatar
	}

	durationSeconds := int64(time.Since(match.MatchTime).Seconds())
	if durationSeconds < 0 {
		durationSeconds = 0
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
	snookerFormat, snookerTargetWins := normalizedSnookerFormat(match)

	return &types.CurrentMatchInfo{
		Id:                            match.Id,
		GameType:                      match.GameType,
		GameTypeName:                  GetGameTypeName(match.GameType),
		GameMode:                      match.GameMode,
		MatchMode:                     model.NormalizeMatchMode(match.MatchMode),
		Visibility:                    model.NormalizeMatchVisibility(match.Visibility, match.MatchMode),
		FinishState:                   model.NormalizeFinishState(match.FinishState),
		FinishRequestedBy:             resolveFinishRequestedBy(match),
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
		LastAction:                    buildMatchLastAction(svcCtx, userId, match),
		Player1Id:                     match.UserId,
		Player1Name:                   player1Name,
		Player1Avatar:                 player1Avatar,
		Player2Id:                     player2Id,
		Player2Name:                   player2Name,
		Player2Avatar:                 player2Avatar,
		OpponentId:                    opponentId,
		OpponentName:                  opponentName,
		OpponentAvatar:                opponentAvatar,
		MyScore:                       myScore,
		OpponentScore:                 opponentScore,
		CurrentFrameStarted:           match.CurrentFrameStarted,
		CurrentFrameMyScore:           currentFrameMyScore,
		CurrentFrameOpponentScore:     currentFrameOpponentScore,
		CurrentRound:                  int(roundCount) + 1,
		SnookerRulesVersion:           match.SnookerRulesVersion,
		BestOfFrames:                  match.BestOfFrames,
		SnookerFormat:                 snookerFormat,
		SnookerTargetWins:             snookerTargetWins,
		CanChangeSnookerFormat:        canChangeSnookerFormat(svcCtx, userId, match),
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
		ServerRevision:                match.SyncRevision,
		MatchTime:                     match.MatchTime.Format("2006-01-02T15:04:05+08:00"),
		DurationSeconds:               durationSeconds,
	}
}
