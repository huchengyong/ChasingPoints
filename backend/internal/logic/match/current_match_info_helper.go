package match

import (
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func buildCurrentMatchInfo(svcCtx *svc.ServiceContext, userId int64, match *model.Match) *types.CurrentMatchInfo {
	if match == nil {
		return nil
	}

	roundCount := int64(0)
	if svcCtx != nil && svcCtx.MatchModel != nil {
		roundCount, _ = svcCtx.MatchModel.GetRoundCount(match.Id)
	}

	capabilities := resolveMatchViewerCapabilities(match, userId)
	isPlayer1 := !shouldUsePlayer2Perspective(capabilities.ViewerRole)
	myScore := match.MyScore
	opponentScore := match.OpponentScore
	currentFrameMyScore := match.CurrentFrameMyScore
	currentFrameOpponentScore := match.CurrentFrameOpponentScore
	opponentName := match.OpponentName
	opponentAvatar := ""
	opponentId := int64(0)
	refereeName := ""

	if capabilities.RefereeBound && svcCtx != nil && svcCtx.UserModel != nil {
		if referee, err := svcCtx.UserModel.FindById(capabilities.RefereeUserId); err == nil && referee != nil {
			refereeName = referee.Nickname
		}
	}

	if capabilities.ViewerRole == matchViewerRoleReferee {
		if match.OpponentId != nil {
			opponentId = *match.OpponentId
			if svcCtx != nil && svcCtx.UserModel != nil {
				if user, err := svcCtx.UserModel.FindById(*match.OpponentId); err == nil && user != nil {
					if user.Nickname != "" {
						opponentName = user.Nickname
					}
					opponentAvatar = user.Avatar
				}
			}
		}
	} else if isPlayer1 {
		if match.OpponentId != nil {
			opponentId = *match.OpponentId
			if svcCtx != nil && svcCtx.UserModel != nil {
				if user, err := svcCtx.UserModel.FindById(*match.OpponentId); err == nil && user != nil {
					if user.Nickname != "" {
						opponentName = user.Nickname
					}
					opponentAvatar = user.Avatar
				}
			}
		}
	} else {
		myScore = match.OpponentScore
		opponentScore = match.MyScore
		currentFrameMyScore = match.CurrentFrameOpponentScore
		currentFrameOpponentScore = match.CurrentFrameMyScore
		opponentId = match.UserId
		if svcCtx != nil && svcCtx.UserModel != nil {
			if user, err := svcCtx.UserModel.FindById(match.UserId); err == nil && user != nil {
				if user.Nickname != "" {
					opponentName = user.Nickname
				}
				opponentAvatar = user.Avatar
			}
		}
	}

	durationSeconds := int64(time.Since(match.MatchTime).Seconds())
	if durationSeconds < 0 {
		durationSeconds = 0
	}

	return &types.CurrentMatchInfo{
		Id:                        match.Id,
		GameType:                  match.GameType,
		GameTypeName:              GetGameTypeName(match.GameType),
		GameMode:                  match.GameMode,
		MatchMode:                 model.NormalizeMatchMode(match.MatchMode),
		Visibility:                model.NormalizeMatchVisibility(match.Visibility, match.MatchMode),
		FinishState:               model.NormalizeFinishState(match.FinishState),
		FinishRequestedBy:         resolveFinishRequestedBy(match),
		ViewerRole:                capabilities.ViewerRole,
		RefereeBound:              capabilities.RefereeBound,
		RefereeUserId:             capabilities.RefereeUserId,
		RefereeName:               refereeName,
		CanScore:                  capabilities.CanScore,
		CanUndo:                   capabilities.CanUndo,
		CanFinish:                 capabilities.CanFinish,
		CanRequestFinish:          capabilities.CanRequestFinish,
		CanConfirmFinish:          capabilities.CanConfirmFinish,
		CanDisputeFinish:          capabilities.CanDisputeFinish,
		CanWithdrawFinish:         capabilities.CanWithdrawFinish,
		LastAction:                buildMatchLastAction(svcCtx, userId, match),
		OpponentId:                opponentId,
		OpponentName:              opponentName,
		OpponentAvatar:            opponentAvatar,
		MyScore:                   myScore,
		OpponentScore:             opponentScore,
		CurrentFrameStarted:       match.CurrentFrameStarted,
		CurrentFrameMyScore:       currentFrameMyScore,
		CurrentFrameOpponentScore: currentFrameOpponentScore,
		CurrentRound:              int(roundCount) + 1,
		ServerRevision:            match.SyncRevision,
		MatchTime:                 match.MatchTime.Format("2006-01-02T15:04:05+08:00"),
		DurationSeconds:           durationSeconds,
	}
}
