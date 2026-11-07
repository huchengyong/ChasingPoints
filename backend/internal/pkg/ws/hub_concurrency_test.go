package ws

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// 慢连接被踢出房间时，Hub 会修改房间 map；同时其他 goroutine 正通过
// GetRoomClientCount 读取同一个 map。用 -race 运行时可发现锁使用错误。
func TestEvictingSlowMatchClientDoesNotRaceRoomReaders(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	const matchID = int64(301)
	slow := &Client{Hub: hub, MatchId: matchID, UserId: 1, Send: make(chan []byte, 1)}
	slow.Send <- []byte("backlog")
	healthy := &Client{Hub: hub, MatchId: matchID, UserId: 2, Send: make(chan []byte, 8)}
	hub.Register <- slow
	hub.Register <- healthy
	waitForRoomSize(t, hub, matchID, 2)

	stop := make(chan struct{})
	var readers sync.WaitGroup
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
					hub.GetRoomClientCount(matchID)
				}
			}
		}()
	}

	hub.BroadcastToMatch(matchID, &Message{Type: "score_update"})
	waitForRoomSize(t, hub, matchID, 1)
	close(stop)
	readers.Wait()

	<-slow.Send
	if _, ok := <-slow.Send; ok {
		t.Fatal("evicted slow client's send channel must be closed")
	}
	select {
	case <-healthy.Send:
	case <-time.After(time.Second):
		t.Fatal("healthy client did not receive the broadcast")
	}
}

func TestEvictingSlowViewerClientRemovesOnlyThatClient(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	const matchID = int64(302)
	slow := &Client{Hub: hub, MatchId: matchID, UserId: 1, Send: make(chan []byte)}
	healthy := &Client{Hub: hub, MatchId: matchID, UserId: 2, Send: make(chan []byte, 8)}
	hub.Register <- slow
	hub.Register <- healthy
	waitForRoomSize(t, hub, matchID, 2)

	hub.BroadcastToMatchForUsers(matchID, map[int64]*Message{
		1: {Type: "sync"},
		2: {Type: "sync"},
	})
	waitForRoomSize(t, hub, matchID, 1)

	if _, ok := <-slow.Send; ok {
		t.Fatal("evicted viewer's send channel must be closed")
	}
}

func TestEvictingSlowUserClientClosesItsChannel(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	const userID = int64(303)
	slow := &Client{Hub: hub, UserId: userID, Send: make(chan []byte)}
	hub.RegisterUser <- slow
	waitForCondition(t, func() bool { return userRoomSize(hub, userID) == 1 })

	hub.SendToUser(userID, &Message{Type: "match_invite"})
	waitForCondition(t, func() bool { return userRoomSize(hub, userID) == 0 })

	if _, ok := <-slow.Send; ok {
		t.Fatal("evicted user client's send channel must be closed")
	}
}

// 连接被 Hub 踢出（Send 已关闭）后，读协程仍可能收到客户端的 ping；
// 回复 pong 不能向已关闭的 channel 发送而 panic。
func TestPingAfterMatchClientEvictionDoesNotPanic(t *testing.T) {
	client := &Client{Hub: NewHub(), MatchId: 304, UserId: 1, Send: make(chan []byte, 1)}
	client.closeSend()

	client.handleMatchMessage(mustMarshalMessage(t, Message{Type: "ping"}))
}

func TestPingAfterUserClientEvictionDoesNotPanic(t *testing.T) {
	client := &Client{Hub: NewHub(), UserId: 1, Send: make(chan []byte, 1)}
	client.closeSend()

	client.handleUserMessage(mustMarshalMessage(t, Message{Type: "ping"}))
}

// 发送缓冲已满时，回复 pong 不能阻塞读协程。
func TestPingWithFullSendBufferDoesNotBlock(t *testing.T) {
	client := &Client{Hub: NewHub(), MatchId: 305, UserId: 1, Send: make(chan []byte, 1)}
	client.Send <- []byte("backlog")

	done := make(chan struct{})
	go func() {
		client.handleMatchMessage(mustMarshalMessage(t, Message{Type: "ping"}))
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ping reply blocked on a full send buffer")
	}
}

func TestPingRepliesWithPong(t *testing.T) {
	client := &Client{Hub: NewHub(), MatchId: 306, UserId: 1, Send: make(chan []byte, 1)}

	client.handleMatchMessage(mustMarshalMessage(t, Message{Type: "ping"}))

	var reply Message
	if err := json.Unmarshal(<-client.Send, &reply); err != nil || reply.Type != "pong" {
		t.Fatalf("expected pong reply, got %#v err=%v", reply, err)
	}
}

// Hub 关闭 Send 与读协程、快照协程发送可能同时发生，用 -race 运行验证。
func TestConcurrentSendAndCloseIsSafe(t *testing.T) {
	for i := 0; i < 50; i++ {
		client := &Client{Send: make(chan []byte, 4)}
		var wg sync.WaitGroup
		for j := 0; j < 4; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for k := 0; k < 20; k++ {
					client.trySend([]byte("x"))
				}
			}()
		}
		wg.Add(2)
		go func() { defer wg.Done(); client.closeSend() }()
		go func() { defer wg.Done(); client.closeSend() }()
		wg.Wait()

		if client.trySend([]byte("late")) {
			t.Fatal("send after close must be rejected")
		}
	}
}

func TestDisconnectUserAfterEvictionDoesNotDoubleClose(t *testing.T) {
	hub := NewHub()
	client := &Client{Hub: hub, MatchId: 307, UserId: 9, Send: make(chan []byte, 1)}
	hub.rooms[307] = map[*Client]bool{client: true}
	client.closeSend()

	hub.DisconnectUser(9)

	if hub.GetRoomClientCount(307) != 0 {
		t.Fatal("disconnected user's match client was not removed")
	}
}

func mustMarshalMessage(t *testing.T, msg Message) []byte {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	return data
}

func userRoomSize(hub *Hub, userID int64) int {
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	return len(hub.userRooms[userID])
}

func waitForRoomSize(t *testing.T, hub *Hub, matchID int64, want int) {
	t.Helper()
	waitForCondition(t, func() bool { return hub.GetRoomClientCount(matchID) == want })
}

func waitForCondition(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not reached before deadline")
}
