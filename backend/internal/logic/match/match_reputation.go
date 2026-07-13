package match

import (
	"fmt"
	"strings"
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
)

const (
	startMatchBlockReasonLowReputationBan = "low_reputation_ban"
	matchReputationReasonCodeCombined     = "abnormal_match"
)

type matchReputationTrigger struct {
	ReasonCode   string
	PenaltyScore int
	ReasonDetail string
}

func (l *StartMatchLogic) loadStartMatchReputationBlock(userID int64) *startMatchDecision {
	if l == nil || l.svcCtx == nil || l.svcCtx.UserReputationProfileModel == nil || l.svcCtx.ReputationConfigModel == nil {
		return nil
	}

	profile, err := logicx.NewReputationService(l.svcCtx, logicx.NowUTC8).GetOrCreateProfile(userID)
	if err != nil {
		l.Logger.Errorf("获取用户信誉状态失败，忽略禁赛拦截: userId=%d, err=%v", userID, err)
		return nil
	}
	if profile == nil || profile.BanUntil == nil {
		return nil
	}
	if logicx.InUTC8(*profile.BanUntil).After(logicx.InUTC8(logicx.NowUTC8())) {
		return &startMatchDecision{
			Action:      startMatchActionBlocked,
			BlockReason: startMatchBlockReasonLowReputationBan,
			Message:     buildLowReputationBanMessage(profile.BanUntil),
		}
	}
	return nil
}

func (l *FinishMatchLogic) applyMatchReputation(match *model.Match) {
	if l == nil || l.svcCtx == nil || match == nil || match.OpponentId == nil || *match.OpponentId <= 0 {
		return
	}

	reputationService := logicx.NewReputationService(l.svcCtx, logicx.NowUTC8)
	cfg, err := reputationService.GetConfig()
	if err != nil {
		l.Logger.Errorf("获取信誉配置失败，跳过信誉处罚: matchId=%d, err=%v", match.Id, err)
		return
	}

	endTime := resolveMatchReputationEndTime(match)
	if endTime.IsZero() || endTime.Before(match.MatchTime) {
		return
	}

	rounds, err := l.svcCtx.MatchModel.ListCompletedRounds(match.Id)
	if err != nil {
		l.Logger.Errorf("获取完赛局数失败，跳过信誉处罚: matchId=%d, err=%v", match.Id, err)
		return
	}

	priorMatchCount, err := l.countRecentCompletedMatchesForReputation(match, cfg.DetectionRules.SameOpponentRule, endTime)
	if err != nil {
		l.Logger.Errorf("统计同对手高频对局失败，跳过信誉处罚: matchId=%d, err=%v", match.Id, err)
		return
	}

	triggers := evaluateMatchReputationTriggers(
		match,
		cfg.DetectionRules,
		len(rounds),
		endTime.Sub(match.MatchTime),
		priorMatchCount,
	)
	if len(triggers) == 0 {
		return
	}

	for idx, penalty := range buildMatchReputationPenalties(match, triggers, cfg.DetectionRules.StackPenaltiesPerMatch) {
		countAsAbnormalMatch := idx == 0
		for _, userID := range []int64{match.UserId, *match.OpponentId} {
			if _, err := reputationService.ApplyPenalty(logicx.ReputationPenaltyInput{
				UserID:               userID,
				MatchID:              match.Id,
				PenaltyScore:         penalty.PenaltyScore,
				ReasonCode:           penalty.ReasonCode,
				ReasonDetail:         penalty.ReasonDetail,
				CountAsAbnormalMatch: countAsAbnormalMatch,
			}); err != nil {
				l.Logger.Errorf("写入信誉处罚失败: matchId=%d, userId=%d, reason=%s, err=%v", match.Id, userID, penalty.ReasonCode, err)
			}
		}
	}
}

func (l *FinishMatchLogic) countRecentCompletedMatchesForReputation(
	match *model.Match,
	rule model.ReputationSameOpponentRule,
	endTime time.Time,
) (int64, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.MatchModel == nil || match == nil || match.OpponentId == nil || *match.OpponentId <= 0 {
		return 0, nil
	}
	if !rule.Enabled || rule.WindowMinutes <= 0 || rule.MaxMatches <= 0 {
		return 0, nil
	}

	windowStart := endTime.Add(-time.Duration(rule.WindowMinutes) * time.Minute)
	windowEnd := endTime.Add(time.Second)
	if rule.RequireSameGameType {
		return l.svcCtx.MatchModel.CountCompletedMatchesBetweenUsersByGameTypeBetween(
			nil,
			match.UserId,
			*match.OpponentId,
			match.GameType,
			windowStart,
			windowEnd,
			match.Id,
		)
	}

	var total int64
	for _, gameType := range []int{1, 2, 3, 4} {
		count, err := l.svcCtx.MatchModel.CountCompletedMatchesBetweenUsersByGameTypeBetween(
			nil,
			match.UserId,
			*match.OpponentId,
			gameType,
			windowStart,
			windowEnd,
			match.Id,
		)
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

func resolveMatchReputationEndTime(match *model.Match) time.Time {
	if match == nil {
		return time.Time{}
	}
	if match.EndTime != nil && !match.EndTime.IsZero() {
		return *match.EndTime
	}
	return logicx.NowUTC8()
}

func evaluateMatchReputationTriggers(
	match *model.Match,
	rules model.ReputationDetectionRules,
	completedRounds int,
	totalDuration time.Duration,
	priorCompletedMatches int64,
) []matchReputationTrigger {
	if match == nil || completedRounds <= 0 || totalDuration <= 0 {
		return nil
	}

	triggers := make([]matchReputationTrigger, 0, 2)
	for _, rule := range rules.DurationRules {
		if rule.GameType != match.GameType || !rule.Enabled || rule.PenaltyScore <= 0 {
			continue
		}
		if completedRounds < rule.MinTotalRounds {
			break
		}
		if totalDuration < time.Duration(rule.MinTotalDurationMinutes)*time.Minute {
			triggers = append(triggers, matchReputationTrigger{
				ReasonCode:   model.ReputationReasonDurationAbnormal,
				PenaltyScore: rule.PenaltyScore,
				ReasonDetail: fmt.Sprintf("duration_abnormal game_type=%d rounds=%d duration_seconds=%d min_total_duration_minutes=%d", match.GameType, completedRounds, int(totalDuration.Seconds()), rule.MinTotalDurationMinutes),
			})
		}
		break
	}

	sameOpponent := rules.SameOpponentRule
	if sameOpponent.Enabled && sameOpponent.MaxMatches > 0 && sameOpponent.PenaltyScore > 0 {
		currentWindowMatches := int(priorCompletedMatches) + 1
		if currentWindowMatches > sameOpponent.MaxMatches {
			triggers = append(triggers, matchReputationTrigger{
				ReasonCode:   model.ReputationReasonSameOpponentHighFrequency,
				PenaltyScore: sameOpponent.PenaltyScore,
				ReasonDetail: fmt.Sprintf("same_opponent_high_frequency game_type=%d window_minutes=%d matches=%d max_matches=%d", match.GameType, sameOpponent.WindowMinutes, currentWindowMatches, sameOpponent.MaxMatches),
			})
		}
	}

	return triggers
}

func buildMatchReputationPenalties(
	match *model.Match,
	triggers []matchReputationTrigger,
	stackPenalties bool,
) []matchReputationTrigger {
	if len(triggers) == 0 {
		return nil
	}
	if stackPenalties || len(triggers) == 1 {
		return triggers
	}

	maxPenalty := triggers[0].PenaltyScore
	reasons := make([]string, 0, len(triggers))
	for _, trigger := range triggers {
		if trigger.PenaltyScore > maxPenalty {
			maxPenalty = trigger.PenaltyScore
		}
		reasons = append(reasons, trigger.ReasonCode)
	}

	return []matchReputationTrigger{
		{
			ReasonCode:   matchReputationReasonCodeCombined,
			PenaltyScore: maxPenalty,
			ReasonDetail: fmt.Sprintf("combined_abnormal_match match_id=%d reasons=%s", match.Id, strings.Join(reasons, ",")),
		},
	}
}

func buildLowReputationBanMessage(banUntil *time.Time) string {
	if banUntil == nil || banUntil.IsZero() {
		return "当前信誉状态限制发起比赛，请稍后再试"
	}
	return "当前信誉状态限制发起比赛，预计恢复时间：" + logicx.FormatUTC8Time(*banUntil)
}
