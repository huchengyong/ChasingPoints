package match

import (
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

func normalizedPoolMatchFormat(match *model.Match) (string, int) {
	if match == nil || !model.IsPoolMatchFormatGameType(match.GameType) {
		return "", 0
	}
	format, targetWins, ok := model.NormalizePoolMatchFormat(match.GameType, match.MatchFormat, match.TargetWins)
	if !ok {
		return model.MatchFormatLegacy, 0
	}
	return format, targetWins
}

func canChangeMatchFormat(svcCtx *svc.ServiceContext, userID int64, match *model.Match) bool {
	return canChangeMatchFormatWithTx(svcCtx, nil, userID, match)
}

func canChangeMatchFormatWithTx(svcCtx *svc.ServiceContext, tx *gorm.DB, userID int64, match *model.Match) bool {
	if svcCtx == nil || match == nil || !model.IsFlexiblePoolMatch(match) {
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

func poolNormalFinishEligible(match *model.Match) bool {
	if !model.IsFlexiblePoolMatch(match) || match.Status != 1 {
		return false
	}
	format, targetWins := normalizedPoolMatchFormat(match)
	switch format {
	case model.MatchFormatFree:
		return match.MyScore+match.OpponentScore >= 1
	case model.MatchFormatRaceTo:
		return targetWins > 0 && (match.MyScore >= targetWins || match.OpponentScore >= targetWins)
	default:
		return true
	}
}

func poolMatchFormatLimitReached(match *model.Match) bool {
	if !model.IsFlexiblePoolMatch(match) {
		return false
	}
	format, _ := normalizedPoolMatchFormat(match)
	return format == model.MatchFormatFree && match.MyScore+match.OpponentScore >= model.PoolMatchMaxCompletedRounds ||
		format == model.MatchFormatRaceTo && model.PoolMatchTargetReached(match)
}
