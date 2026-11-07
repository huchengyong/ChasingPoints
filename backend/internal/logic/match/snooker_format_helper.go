package match

import (
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

func canChangeSnookerFormat(svcCtx *svc.ServiceContext, userID int64, match *model.Match) bool {
	return canChangeSnookerFormatWithTx(svcCtx, nil, userID, match)
}

func canChangeSnookerFormatWithTx(svcCtx *svc.ServiceContext, tx *gorm.DB, userID int64, match *model.Match) bool {
	if svcCtx == nil || match == nil || match.GameType != 1 || match.SnookerRulesVersion != model.SnookerRulesVersionWPBSA {
		return false
	}
	if userID <= 0 || match.UserId != userID || match.Status != 1 || model.NormalizeFinishState(match.FinishState) != model.FinishStateNone || match.RefereeUserId != nil {
		return false
	}
	actionCount, err := svcCtx.MatchModel.CountActionsWithTx(tx, match.Id)
	if err != nil || actionCount > 0 {
		return false
	}
	roundCount, err := svcCtx.MatchModel.GetRoundCountWithTx(tx, match.Id)
	return err == nil && roundCount == 0
}

func normalizedSnookerFormat(match *model.Match) (string, int) {
	if match == nil || match.GameType != 1 {
		return "", 0
	}
	format, targetWins, ok := model.NormalizeSnookerFormat(match.SnookerFormat, match.SnookerTargetWins, match.BestOfFrames)
	if !ok {
		return model.SnookerFormatLegacy, 0
	}
	return format, targetWins
}

func snookerNormalFinishEligible(match *model.Match) bool {
	if !isSnookerV2Match(match) || match.Status != 1 || match.CurrentFrameStarted {
		return false
	}
	format, targetWins := normalizedSnookerFormat(match)
	switch format {
	case model.SnookerFormatFree:
		return match.MyScore+match.OpponentScore >= 1
	case model.SnookerFormatRaceTo:
		return targetWins > 0 && (match.MyScore >= targetWins || match.OpponentScore >= targetWins)
	default:
		return model.SnookerTargetReached(match)
	}
}

func snookerFormatLimitReached(match *model.Match) bool {
	if !isSnookerV2Match(match) {
		return false
	}
	format, _ := normalizedSnookerFormat(match)
	return format == model.SnookerFormatFree && match.MyScore+match.OpponentScore >= model.SnookerMaxCompletedFrames ||
		format == model.SnookerFormatRaceTo && model.SnookerTargetReached(match)
}
