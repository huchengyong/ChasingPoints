package logic

import (
	"encoding/json"
	"testing"
	"time"

	"chasing_points/internal/pkg/ws"
)

func TestSendUserDataUpdatedCarriesScopedRevisionAndExactFriendRequestCount(t *testing.T) {
	previousHub := ws.GlobalHub
	hub := ws.NewHub()
	ws.GlobalHub = hub
	t.Cleanup(func() { ws.GlobalHub = previousHub })

	pendingCount := int64(3)
	SendUserDataUpdated(42, UserDataUpdatedEvent{
		Scopes:                    []string{"rank", "stats", "friend_requests"},
		CompetitiveRevision:       9,
		MatchID:                   1001,
		GameType:                  3,
		PendingFriendRequestCount: &pendingCount,
	})

	select {
	case message := <-hub.SendUser:
		if message.UserId != 42 {
			t.Fatalf("unexpected user id: %d", message.UserId)
		}
		var payload struct {
			Type string               `json:"type"`
			Data UserDataUpdatedEvent `json:"data"`
		}
		if err := json.Unmarshal(message.Message, &payload); err != nil {
			t.Fatalf("decode message: %v", err)
		}
		if payload.Type != "user_data_updated" || payload.Data.CompetitiveRevision != 9 || payload.Data.MatchID != 1001 || payload.Data.GameType != 3 {
			t.Fatalf("unexpected event payload: %+v", payload)
		}
		if payload.Data.PendingFriendRequestCount == nil || *payload.Data.PendingFriendRequestCount != 3 {
			t.Fatalf("missing precise friend request count: %+v", payload.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("expected user data event")
	}
}
