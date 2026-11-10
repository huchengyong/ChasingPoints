package logic

import (
	"time"

	"chasing_points/internal/model"
)

// MatchDurationSeconds 计算对局已持续秒数：进行中按当前时间，已结束按结束时间；负数归 0。
// 参与者详情与公开详情共用此口径。
func MatchDurationSeconds(match *model.Match, now time.Time) int64 {
	if match == nil {
		return 0
	}
	var durationSeconds int64
	if match.Status == 1 { // 进行中
		durationSeconds = int64(now.Sub(match.CreatedAt).Seconds())
	} else if match.EndTime != nil {
		durationSeconds = int64(match.EndTime.Sub(match.CreatedAt).Seconds())
	}
	if durationSeconds < 0 {
		durationSeconds = 0
	}
	return durationSeconds
}
