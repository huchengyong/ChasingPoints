package logic

import (
	"time"

	"billiard_master/internal/model"
	"billiard_master/internal/svc"
	"billiard_master/internal/types"
)

func buildCurrentMatchInfo(svcCtx *svc.ServiceContext, userId int64, match *model.Match) *types.CurrentMatchInfo {
	if match == nil {
		return nil
	}

	roundCount := int64(0)
	if svcCtx != nil && svcCtx.MatchModel != nil {
		roundCount, _ = svcCtx.MatchModel.GetRoundCount(match.Id)
	}

	isPlayer1 := match.UserId == userId
	myScore := match.MyScore
	opponentScore := match.OpponentScore
	currentFrameMyScore := match.CurrentFrameMyScore
	currentFrameOpponentScore := match.CurrentFrameOpponentScore
	opponentName := match.OpponentName
	opponentAvatar := ""
	opponentId := int64(0)

	if isPlayer1 {
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
