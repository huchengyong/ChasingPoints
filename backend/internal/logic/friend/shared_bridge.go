package friend

import logic "chasing_points/internal/logic"

func buildNotificationPayload(target string, matchId, requestId int64) *string {
	return logic.BuildNotificationPayload(target, matchId, requestId)
}
