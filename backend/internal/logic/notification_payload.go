package logic

import "encoding/json"

type notificationTargetPayload struct {
	Target    string `json:"target,omitempty"`
	MatchId   int64  `json:"match_id,omitempty"`
	RequestId int64  `json:"request_id,omitempty"`
}

func buildNotificationPayload(target string, matchId, requestId int64) *string {
	payload := notificationTargetPayload{
		Target:    target,
		MatchId:   matchId,
		RequestId: requestId,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}

	value := string(data)
	return &value
}

func BuildNotificationPayload(target string, matchId, requestId int64) *string {
	return buildNotificationPayload(target, matchId, requestId)
}
