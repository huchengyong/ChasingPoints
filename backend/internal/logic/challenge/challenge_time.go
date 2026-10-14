package challenge

import (
	"fmt"
	"time"

	logicx "chasing_points/internal/logic"
)

// challengeMaxDaysAhead 只能约今天/明天/后天。
const challengeMaxDaysAhead = 2

// ChallengeScheduleInput 客户端提交的实际预约日期（北京时间）与整点区间。
type ChallengeScheduleInput struct {
	ScheduledDate string // YYYY-MM-DD，与 App 展示一致的实际预约日期
	StartHour     int
	EndHour       int
}

// ValidateChallengeSchedule 校验北京时间三天整点区间并计算次晨07:00失效时刻。
// 只按提交的实际日期校验，不做相对日换算，因此跨午夜也不会静默滚到另一天；
// 返回实际预约日期、失效时刻；拒绝超出今天到后天、无效或已完全过去的区间。
func ValidateChallengeSchedule(input ChallengeScheduleInput, now time.Time) (time.Time, time.Time, error) {
	if input.StartHour < 0 || input.StartHour > 23 || input.EndHour < 1 || input.EndHour > 24 || input.EndHour <= input.StartHour {
		return time.Time{}, time.Time{}, fmt.Errorf("请选择有效的整点区间")
	}
	if input.ScheduledDate == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("请选择预约日期")
	}
	scheduled, err := time.ParseInLocation(time.DateOnly, input.ScheduledDate, logicx.UTC8Location)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("预约日期无效")
	}
	beijingNow := logicx.InUTC8(now)
	today := time.Date(beijingNow.Year(), beijingNow.Month(), beijingNow.Day(), 0, 0, 0, 0, logicx.UTC8Location)
	if scheduled.Before(today) || scheduled.After(today.AddDate(0, 0, challengeMaxDaysAhead)) {
		return time.Time{}, time.Time{}, fmt.Errorf("仅支持今天/明天/后天，请重新选择")
	}
	// 区间不得完全过去：结束时刻（24点=次日0点）必须晚于当前北京时间。
	endTime := scheduled.AddDate(0, 0, 1)
	if input.EndHour < 24 {
		endTime = time.Date(scheduled.Year(), scheduled.Month(), scheduled.Day(), input.EndHour, 0, 0, 0, logicx.UTC8Location)
	}
	if !endTime.After(beijingNow) {
		return time.Time{}, time.Time{}, fmt.Errorf("所选时段已过去，请重新选择")
	}
	expiresAt := scheduled.AddDate(0, 0, 1)
	expiresAt = time.Date(expiresAt.Year(), expiresAt.Month(), expiresAt.Day(), 7, 0, 0, 0, logicx.UTC8Location)
	return scheduled, expiresAt, nil
}

// ChallengeScheduledStartTime 约定开始时刻（北京时间预约日 + 开始小时）。
func ChallengeScheduledStartTime(scheduledDate time.Time, startHour int) time.Time {
	date := logicx.InUTC8(scheduledDate)
	return time.Date(date.Year(), date.Month(), date.Day(), startHour, 0, 0, 0, logicx.UTC8Location)
}

// ChallengeScheduleExpired 未开局约球在失效时刻后不可再接受/进入；不依赖 worker。
func ChallengeScheduleExpired(expiresAt time.Time, now time.Time) bool {
	return !now.Before(expiresAt)
}
