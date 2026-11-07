package logic

import "chasing_points/internal/pkg/ws"

// UserDataUpdatedEvent 是认证用户资源失效的提交后通知。
type UserDataUpdatedEvent struct {
	Scopes                    []string `json:"scopes"`
	CompetitiveRevision       int64    `json:"competitive_revision"`
	MatchID                   int64    `json:"match_id"`
	GameType                  int      `json:"game_type"`
	PendingFriendRequestCount *int64   `json:"pending_friend_request_count,omitempty"`
}

// SendUserDataUpdated 在数据库提交后尽力发送失效事件。发送失败不能影响已提交业务数据。
func SendUserDataUpdated(userID int64, event UserDataUpdatedEvent) {
	if userID <= 0 || ws.GlobalHub == nil {
		return
	}
	ws.GlobalHub.SendToUser(userID, &ws.Message{
		Type: "user_data_updated",
		Data: event,
	})
}
