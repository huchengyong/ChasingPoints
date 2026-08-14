package match

import (
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"gorm.io/gorm"
)

func buildRankSettlementPolicy(
	tx *gorm.DB,
	svcCtx *svc.ServiceContext,
	userId int64,
	opponentId *int64,
	gameType int,
	effectiveAt time.Time,
	excludeMatchId int64,
	completedRounds int,
	opponentCurrentRankScore int,
) (RankSettlementPolicy, error) {
	policy := RankSettlementPolicy{
		DailyPositiveCap:          defaultDailyPositiveCap,
		DailyMemberAchievementCap: logicx.DefaultMemberAchievementDailyCap,
		MemberMultiplierPercent:   100,
		CompletedRounds:           completedRounds,
		OpponentCurrentRankScore:  opponentCurrentRankScore,
	}
	if svcCtx == nil || userId <= 0 {
		return policy, nil
	}

	config, configErr := logicx.NewMemberRightsConfigService(svcCtx).GetConfig()
	if configErr == nil {
		policy.DailyMemberAchievementCap = config.RankingRights.DailyCap
		policy.DailyPositiveCap = config.RankingRights.DailyPositiveCap
		policy.OrdinaryUserAchievementEnabled = config.RankingRights.OrdinaryUserAchievementEnabled
	}

	dayStart, nextDayStart := rankSettlementDayRange(effectiveAt)
	todayPositiveGain, err := svcCtx.RankingModel.SumPositiveRankChangesByUserAndGameTypeBetween(
		tx,
		userId,
		gameType,
		dayStart,
		nextDayStart,
	)
	if err != nil {
		return policy, err
	}
	policy.TodayPositiveGain = todayPositiveGain

	todayMemberAchievementGain, err := svcCtx.RankingModel.SumPositiveAchievementRankChangesByUserAndGameTypeBetween(
		tx,
		userId,
		gameType,
		dayStart,
		nextDayStart,
	)
	if err != nil {
		return policy, err
	}
	policy.TodayMemberAchievementGain = todayMemberAchievementGain

	if svcCtx.UserModel != nil {
		user, err := svcCtx.UserModel.FindById(userId)
		if err != nil {
			return policy, err
		}
		if user != nil && user.MemberExpiresAt != nil && user.MemberExpiresAt.After(effectiveAt) {
			policy.MemberActive = true
		}
	}

	if svcCtx.MemberGrowthProfileModel != nil {
		profile, err := svcCtx.MemberGrowthProfileModel.FindByUserIdWithTx(tx, userId)
		if err != nil {
			return policy, err
		}
		if profile != nil {
			policy.MemberLevel = profile.GrowthLevel
		}
	}
	if policy.MemberLevel <= 0 {
		policy.MemberLevel = 1
	}
	if configErr == nil {
		switch policy.MemberLevel {
		case 5:
			policy.MemberMultiplierPercent = config.RankingRights.Level5Multiplier
		case 4:
			policy.MemberMultiplierPercent = config.RankingRights.Level4Multiplier
		case 3:
			policy.MemberMultiplierPercent = config.RankingRights.Level3Multiplier
		case 2:
			policy.MemberMultiplierPercent = config.RankingRights.Level2Multiplier
		default:
			policy.MemberMultiplierPercent = config.RankingRights.Level1Multiplier
		}
	}

	if opponentId != nil && *opponentId > 0 {
		sameOpponentMatchesToday, err := svcCtx.MatchModel.CountCompletedMatchesBetweenUsersByGameTypeBetween(
			tx,
			userId,
			*opponentId,
			gameType,
			dayStart,
			nextDayStart,
			excludeMatchId,
		)
		if err != nil {
			return policy, err
		}
		policy.SameOpponentMatchesToday = int(sameOpponentMatchesToday)
	}

	return policy, nil
}

func rankSettlementOpponentID(match *model.Match, userId int64) *int64 {
	if match == nil {
		return nil
	}
	if match.UserId == userId {
		return match.OpponentId
	}
	if match.OpponentId != nil && *match.OpponentId == userId {
		opponentId := match.UserId
		return &opponentId
	}
	return nil
}

func rankSettlementDayRange(at time.Time) (time.Time, time.Time) {
	if at.IsZero() {
		at = time.Now()
	}
	location := at.Location()
	dayStart := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, location)
	return dayStart, dayStart.Add(24 * time.Hour)
}
