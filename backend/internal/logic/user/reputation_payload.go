package user

import (
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func buildUserReputationResp(score int, banUntil *time.Time, now time.Time) *types.GetUserReputationResp {
	resp := &types.GetUserReputationResp{
		Success:    true,
		Score:      score,
		Status:     "good",
		StatusText: "良好",
	}

	if banUntil != nil && logicx.InUTC8(*banUntil).After(logicx.InUTC8(now)) {
		resp.Status = "restricted"
		resp.StatusText = "禁赛"
		resp.BanUntil = logicx.FormatUTC8TimePtr(banUntil)
	}

	return resp
}

func normalizedUserDisplayInitialScore(baseRules model.ReputationBaseRules) int {
	score := baseRules.InitialScore
	if score < baseRules.MinScore {
		score = baseRules.MinScore
	}
	if baseRules.MaxScore > 0 && score > baseRules.MaxScore {
		score = baseRules.MaxScore
	}
	return score
}

func applyUserDisplayReputationRecovery(profile *model.UserReputationProfile, baseRules model.ReputationBaseRules, rules model.ReputationRecoveryRules, now time.Time) {
	if profile == nil {
		return
	}

	if profile.ReputationScore < baseRules.MinScore {
		profile.ReputationScore = baseRules.MinScore
	}
	if baseRules.MaxScore > 0 && profile.ReputationScore > baseRules.MaxScore {
		profile.ReputationScore = baseRules.MaxScore
	}

	if profile.LastRecoveredAt == nil || profile.LastRecoveredAt.IsZero() {
		return
	}

	now = logicx.InUTC8(now)
	lastRecoveredAt := logicx.InUTC8(*profile.LastRecoveredAt)
	if !rules.Enabled || rules.RecoverPerHour <= 0 || !now.After(lastRecoveredAt) {
		return
	}

	elapsedHours := int(now.Sub(lastRecoveredAt) / time.Hour)
	if elapsedHours <= 0 {
		return
	}

	recoveryCap := rules.RecoverMaxScore
	if recoveryCap <= 0 || (baseRules.MaxScore > 0 && recoveryCap > baseRules.MaxScore) {
		recoveryCap = baseRules.MaxScore
	}
	if recoveryCap > 0 && profile.ReputationScore >= recoveryCap {
		if profile.ReputationScore > recoveryCap {
			profile.ReputationScore = recoveryCap
		}
		return
	}

	profile.ReputationScore += elapsedHours * rules.RecoverPerHour
	if profile.ReputationScore < baseRules.MinScore {
		profile.ReputationScore = baseRules.MinScore
	}
	if recoveryCap > 0 && profile.ReputationScore > recoveryCap {
		profile.ReputationScore = recoveryCap
	}
	if baseRules.MaxScore > 0 && profile.ReputationScore > baseRules.MaxScore {
		profile.ReputationScore = baseRules.MaxScore
	}
}

func buildUserReputationLogsResp(logs []model.UserReputationLog, total int64) *types.UserReputationLogsResp {
	items := make([]types.UserReputationLogItem, 0, len(logs))
	for i := range logs {
		items = append(items, buildUserReputationLogItem(&logs[i]))
	}

	return &types.UserReputationLogsResp{
		Success: true,
		Total:   total,
		List:    items,
	}
}

func buildUserReputationLogItem(log *model.UserReputationLog) types.UserReputationLogItem {
	if log == nil {
		return types.UserReputationLogItem{}
	}

	item := types.UserReputationLogItem{
		Id:             log.ID,
		ChangeType:     log.ChangeType,
		ChangeTypeText: model.ReputationChangeTypeText(log.ChangeType),
		ReasonCode:     SafeUserReputationReasonCode(log.ReasonCode),
		ReasonText:     userSafeReputationReasonText(log),
		ChangeScore:    log.ChangeScore,
		BeforeScore:    log.BeforeScore,
		AfterScore:     log.AfterScore,
		CreatedAt:      logicx.FormatUTC8Time(log.CreatedAt),
	}
	if log.MatchID != nil {
		item.MatchId = *log.MatchID
	}
	return item
}

func SafeUserReputationReasonCode(reasonCode string) string {
	return model.UserFacingReputationReasonCode(reasonCode)
}

func userSafeReputationReasonText(log *model.UserReputationLog) string {
	if log == nil {
		return "信誉变更"
	}

	switch SafeUserReputationReasonCode(log.ReasonCode) {
	case model.ReputationReasonDurationAbnormal:
		return model.ReputationReasonText(model.ReputationReasonDurationAbnormal, log.ChangeType)
	case model.ReputationReasonSameOpponentHighFrequency:
		return model.ReputationReasonText(model.ReputationReasonSameOpponentHighFrequency, log.ChangeType)
	case model.ReputationReasonSystemRecovery:
		return model.ReputationReasonText(model.ReputationReasonSystemRecovery, log.ChangeType)
	case "other":
		return "信誉变更"
	}
	return model.ReputationReasonText("", log.ChangeType)
}
