package match

import (
	"time"

	"chasing_points/internal/model"
)

// isFinishRequestExpired 判断待确认的结束请求是否已到确认期截止时刻（按服务端时间）。
func isFinishRequestExpired(match *model.Match, now time.Time) bool {
	return match.FinishRequestExpired(now)
}
