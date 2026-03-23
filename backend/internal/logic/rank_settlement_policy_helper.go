package logic

import (
	"time"

	"billiard_master/internal/model"
	"billiard_master/internal/svc"

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
) (RankSettlementPolicy, error) {
	policy := RankSettlementPolicy{
		DailyPositiveCap: defaultDailyPositiveCap,
	}
	if svcCtx == nil || userId <= 0 {
		return policy, nil
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
