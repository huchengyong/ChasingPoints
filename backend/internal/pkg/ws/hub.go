package ws

import (
	"encoding/json"
	"sync"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/core/logx"
)

// Client WebSocket客户端
type Client struct {
	Hub     *Hub
	Conn    *websocket.Conn
	SvcCtx  *svc.ServiceContext
	MatchId int64
	UserId  int64
	Send    chan []byte
}

// Message WebSocket消息结构
type Message struct {
	Type string      `json:"type"` // score_update, round_end, match_end, sync
	Data interface{} `json:"data"`
}

// ScoreUpdateData 分数更新数据
type ScoreUpdateData struct {
	MatchId                       int64  `json:"match_id"`
	ServerRevision                int64  `json:"server_revision"`
	MyScore                       int    `json:"my_score"`
	OpponentScore                 int    `json:"opponent_score"`
	Player1Score                  int    `json:"player1_score"`
	Player2Score                  int    `json:"player2_score"`
	CurrentFramePlayer1Score      int    `json:"current_frame_player1_score"`
	CurrentFramePlayer2Score      int    `json:"current_frame_player2_score"`
	CurrentFrameStarted           bool   `json:"current_frame_started"`
	CurrentRound                  int    `json:"current_round"`
	TotalRounds                   int    `json:"total_rounds"`
	RoundNumber                   int    `json:"round_number"`
	RoundPlayer1Score             int    `json:"round_player1_score"`
	RoundPlayer2Score             int    `json:"round_player2_score"`
	Winner                        int    `json:"winner"`
	ActionType                    string `json:"action_type"` // score, foul, win, undo
	Actor                         string `json:"actor"`       // me, opponent
	RedBallCount                  int    `json:"red_ball_count"`
	SnookerClearanceStarted       bool   `json:"snooker_clearance_started"`
	SnookerClearedColors          []int  `json:"snooker_cleared_colors,omitempty"`
	SnookerExpectedClearanceScore int    `json:"snooker_expected_clearance_score,omitempty"`
	SnookerClearanceCompleted     bool   `json:"snooker_clearance_completed,omitempty"`
	Status                        int    `json:"status"`
}

type MatchSyncRound struct {
	RoundNumber  int `json:"round_number"`
	Player1Score int `json:"player1_score"`
	Player2Score int `json:"player2_score"`
	Winner       int `json:"winner"`
}

type MatchSyncData struct {
	MatchId                       int64            `json:"match_id"`
	Status                        int              `json:"status"`
	ServerRevision                int64            `json:"server_revision"`
	Player1Score                  int              `json:"player1_score"`
	Player2Score                  int              `json:"player2_score"`
	CurrentFramePlayer1Score      int              `json:"current_frame_player1_score"`
	CurrentFramePlayer2Score      int              `json:"current_frame_player2_score"`
	CurrentFrameStarted           bool             `json:"current_frame_started"`
	RedBallCount                  int              `json:"red_ball_count"`
	SnookerClearanceStarted       bool             `json:"snooker_clearance_started"`
	SnookerClearedColors          []int            `json:"snooker_cleared_colors,omitempty"`
	SnookerExpectedClearanceScore int              `json:"snooker_expected_clearance_score,omitempty"`
	SnookerClearanceCompleted     bool             `json:"snooker_clearance_completed,omitempty"`
	CurrentRound                  int              `json:"current_round"`
	TotalRounds                   int              `json:"total_rounds"`
	Rounds                        []MatchSyncRound `json:"rounds"`
}

// Hub WebSocket连接管理器
type Hub struct {
	// 按MatchId分组的客户端连接
	rooms map[int64]map[*Client]bool

	// 按UserId分组的客户端连接（用于匹配通知）
	userRooms map[int64]map[*Client]bool

	// 注册新连接（按MatchId）
	Register chan *Client

	// 注销连接（按MatchId）
	Unregister chan *Client

	// 注册用户连接（按UserId）
	RegisterUser chan *Client

	// 注销用户连接（按UserId）
	UnregisterUser chan *Client

	// 广播消息到指定房间
	Broadcast chan *BroadcastMessage

	// 发送消息到指定用户
	SendUser chan *UserMessage

	// 互斥锁
	mu sync.RWMutex
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	MatchId int64
	Message []byte
}

// UserMessage 用户消息
type UserMessage struct {
	UserId  int64
	Message []byte
}

// 全局Hub实例
var GlobalHub *Hub

// NewHub 创建新的Hub
func NewHub() *Hub {
	return &Hub{
		rooms:          make(map[int64]map[*Client]bool),
		userRooms:      make(map[int64]map[*Client]bool),
		Register:       make(chan *Client, 64),
		Unregister:     make(chan *Client, 64),
		RegisterUser:   make(chan *Client, 64),
		UnregisterUser: make(chan *Client, 64),
		Broadcast:      make(chan *BroadcastMessage, 256), // 带缓冲，避免阻塞
		SendUser:       make(chan *UserMessage, 256),      // 带缓冲，避免阻塞
	}
}

// Run 启动Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if h.rooms[client.MatchId] == nil {
				h.rooms[client.MatchId] = make(map[*Client]bool)
			}
			h.rooms[client.MatchId][client] = true
			h.mu.Unlock()
			logx.Infof("客户端连接: MatchId=%d, UserId=%d", client.MatchId, client.UserId)

		case client := <-h.Unregister:
			h.mu.Lock()
			if room, ok := h.rooms[client.MatchId]; ok {
				if _, ok := room[client]; ok {
					delete(room, client)
					close(client.Send)
					// 如果房间为空，删除房间
					if len(room) == 0 {
						delete(h.rooms, client.MatchId)
					}
				}
			}
			h.mu.Unlock()
			logx.Infof("客户端断开: MatchId=%d, UserId=%d", client.MatchId, client.UserId)

		case message := <-h.Broadcast:
			h.mu.RLock()
			if room, ok := h.rooms[message.MatchId]; ok {
				for client := range room {
					select {
					case client.Send <- message.Message:
					default:
						close(client.Send)
						delete(room, client)
					}
				}
			}
			h.mu.RUnlock()

		case client := <-h.RegisterUser:
			h.mu.Lock()
			if h.userRooms[client.UserId] == nil {
				h.userRooms[client.UserId] = make(map[*Client]bool)
			}
			h.userRooms[client.UserId][client] = true
			h.mu.Unlock()
			logx.Infof("用户连接注册: UserId=%d", client.UserId)

		case client := <-h.UnregisterUser:
			h.mu.Lock()
			if room, ok := h.userRooms[client.UserId]; ok {
				if _, ok := room[client]; ok {
					delete(room, client)
					close(client.Send)
					if len(room) == 0 {
						delete(h.userRooms, client.UserId)
					}
				}
			}
			h.mu.Unlock()
			logx.Infof("用户连接断开: UserId=%d", client.UserId)

		case message := <-h.SendUser:
			h.mu.RLock()
			if room, ok := h.userRooms[message.UserId]; ok {
				for client := range room {
					select {
					case client.Send <- message.Message:
					default:
						close(client.Send)
						delete(room, client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToMatch 向指定对局广播消息
func (h *Hub) BroadcastToMatch(matchId int64, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logx.Errorf("序列化消息失败: %v", err)
		return
	}

	logx.Infof("广播对局消息: MatchId=%d, Type=%s, RoomSize=%d", matchId, msg.Type, h.GetRoomClientCount(matchId))

	h.Broadcast <- &BroadcastMessage{
		MatchId: matchId,
		Message: data,
	}
}

// GetRoomClientCount 获取房间客户端数量
func (h *Hub) GetRoomClientCount(matchId int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if room, ok := h.rooms[matchId]; ok {
		return len(room)
	}
	return 0
}

// SendToUser 向指定用户发送消息
func (h *Hub) SendToUser(userId int64, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logx.Errorf("序列化消息失败: %v", err)
		return
	}

	logx.Infof("向用户发送消息: UserId=%d, Type=%s", userId, msg.Type)

	h.SendUser <- &UserMessage{
		UserId:  userId,
		Message: data,
	}
}

const (
	// 写超时
	writeWait = 10 * time.Second

	// Pong超时
	pongWait = 60 * time.Second

	// Ping周期
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512
)

// ReadPump 读取客户端消息
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logx.Errorf("WebSocket读取错误: %v", err)
			}
			break
		}

		// 处理客户端消息（如同步请求）
		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			if msg.Type == "ping" {
				// 响应心跳
				pongMsg, _ := json.Marshal(Message{Type: "pong"})
				c.Send <- pongMsg
			}
			if msg.Type == "sync" {
				logx.Infof("收到对局同步请求: MatchId=%d, UserId=%d", c.MatchId, c.UserId)
				c.sendMatchSync()
			}
		}
	}
}

// WritePump 发送消息到客户端
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// InitGlobalHub 初始化全局Hub
func InitGlobalHub() {
	GlobalHub = NewHub()
	go GlobalHub.Run()
	logx.Info("WebSocket Hub 已启动")
}

func buildMatchSyncData(match *model.Match, completedRoundCount int64, snookerState model.SnookerRoundState, rounds []model.MatchRound) MatchSyncData {
	currentRound := int(completedRoundCount) + 1
	if match != nil && match.Status != 1 && !match.CurrentFrameStarted {
		currentRound = int(completedRoundCount)
	}
	return MatchSyncData{
		MatchId:                       match.Id,
		Status:                        match.Status,
		ServerRevision:                match.SyncRevision,
		Player1Score:                  match.MyScore,
		Player2Score:                  match.OpponentScore,
		CurrentFramePlayer1Score:      match.CurrentFrameMyScore,
		CurrentFramePlayer2Score:      match.CurrentFrameOpponentScore,
		CurrentFrameStarted:           match.CurrentFrameStarted,
		RedBallCount:                  snookerState.RedBallCount,
		SnookerClearanceStarted:       snookerState.ClearanceStarted,
		SnookerClearedColors:          snookerState.ClearedColors,
		SnookerExpectedClearanceScore: snookerState.ExpectedClearanceScore,
		SnookerClearanceCompleted:     snookerState.ClearanceCompleted,
		CurrentRound:                  currentRound,
		TotalRounds:                   int(completedRoundCount),
		Rounds:                        buildMatchSyncRounds(rounds),
	}
}

func buildMatchSyncRounds(rounds []model.MatchRound) []MatchSyncRound {
	list := make([]MatchSyncRound, 0, len(rounds))
	for _, round := range rounds {
		if round.Winner == nil || round.WinType == "start" {
			continue
		}
		list = append(list, MatchSyncRound{
			RoundNumber:  round.RoundNo,
			Player1Score: round.MyScore,
			Player2Score: round.OpponentScore,
			Winner:       *round.Winner,
		})
	}
	return list
}

func (c *Client) sendMatchSync() {
	if c.SvcCtx == nil || c.MatchId == 0 {
		return
	}

	match, err := c.SvcCtx.MatchModel.FindById(c.MatchId)
	if err != nil || match == nil {
		logx.Errorf("获取对局同步快照失败: matchId=%d, err=%v", c.MatchId, err)
		return
	}

	roundCount, err := c.SvcCtx.MatchModel.GetRoundCount(c.MatchId)
	if err != nil {
		logx.Errorf("获取对局局数失败: matchId=%d, err=%v", c.MatchId, err)
		return
	}

	rounds, err := c.SvcCtx.MatchModel.ListCompletedRounds(c.MatchId)
	if err != nil {
		logx.Errorf("获取对局局记录失败: matchId=%d, err=%v", c.MatchId, err)
		return
	}

	snookerState := model.SnookerRoundState{}
	if match.GameType == 1 {
		if actions, actionsErr := c.SvcCtx.MatchModel.ListActiveActions(c.MatchId); actionsErr == nil {
			snookerState = model.BuildSnookerRoundState(actions, int(roundCount)+1)
		} else {
			logx.Errorf("获取斯诺克当前局操作失败: matchId=%d, err=%v", c.MatchId, actionsErr)
		}
	}

	msg := &Message{
		Type: "sync",
		Data: buildMatchSyncData(match, roundCount, snookerState, rounds),
	}

	logx.Infof("发送对局同步快照: MatchId=%d, UserId=%d, CurrentRound=%d, TotalRounds=%d, Status=%d",
		match.Id, c.UserId, int(roundCount)+1, int(roundCount), match.Status)

	payload, err := json.Marshal(msg)
	if err != nil {
		logx.Errorf("序列化对局同步快照失败: matchId=%d, err=%v", c.MatchId, err)
		return
	}

	select {
	case c.Send <- payload:
	default:
		logx.Errorf("发送对局同步快照失败: matchId=%d, userId=%d", c.MatchId, c.UserId)
	}
}
