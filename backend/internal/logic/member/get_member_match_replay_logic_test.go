package member

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type replayFixture struct {
	t      *testing.T
	db     *gorm.DB
	svcCtx *svc.ServiceContext
	now    time.Time
}

// newReplayFixture 准备一场已完成的中式八球抢 5 对局：1 号为创建者，2 号为对手，3 号是裁判，4 号与对局无关。
// 本人（创建者）以 5:3 获胜，连赢第 2–5 局。
func newReplayFixture(t *testing.T) *replayFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare replay schema: %v", err)
	}
	f := &replayFixture{t: t, db: db, svcCtx: &svc.ServiceContext{DB: db}, now: time.Date(2026, 10, 22, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))}
	member := f.now.Add(30 * 24 * time.Hour)
	for _, user := range []model.User{
		{Id: 1, Nickname: "甲", MemberExpiresAt: &member},
		{Id: 2, Nickname: "乙"},
		{Id: 3, Nickname: "裁判", MemberExpiresAt: &member},
		{Id: 4, Nickname: "路人", MemberExpiresAt: &member},
	} {
		u := user
		if err := db.Create(&u).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	opponentID, refereeID := int64(2), int64(3)
	result := 1
	completedAt := f.now.Add(-time.Hour)
	if err := db.Create(&model.Match{
		Id: 10, UserId: 1, OpponentId: &opponentID, RefereeUserId: &refereeID, OpponentName: "乙", GameType: 3, MatchFormat: model.MatchFormatRaceTo, TargetWins: 5,
		Status: 2, Result: &result, MyScore: 5, OpponentScore: 3, MatchTime: completedAt, CompletedAt: &completedAt,
	}).Error; err != nil {
		t.Fatalf("seed match: %v", err)
	}
	winners := []int{2, 1, 1, 1, 1, 2, 2, 1}
	p1, p2 := 0, 0
	for i, winner := range winners {
		if winner == 1 {
			p1++
		} else {
			p2++
		}
		w := winner
		if err := db.Create(&model.MatchRound{MatchId: 10, RoundNo: i + 1, MyScore: p1, OpponentScore: p2, Winner: &w, WinType: "normal"}).Error; err != nil {
			t.Fatalf("seed round: %v", err)
		}
		if err := db.Create(&model.MatchAction{MatchId: 10, RoundNo: i + 1, ActionType: "win", Actor: winner, ScoreChange: 1}).Error; err != nil {
			t.Fatalf("seed action: %v", err)
		}
	}
	return f
}

func (f *replayFixture) get(userID, matchID int64) (*types.GetMemberMatchReplayResp, int64) {
	f.t.Helper()
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", userID), metrics)
	l := NewGetMemberMatchReplayLogic(ctx, f.svcCtx)
	l.now = func() time.Time { return f.now }
	resp, err := l.GetMemberMatchReplay(&types.GetMemberMatchReplayReq{MatchId: matchID})
	if err != nil {
		f.t.Fatalf("get replay: %v", err)
	}
	return resp, metrics.Snapshot().SQLCount
}

func TestMemberMatchReplayReturnsFullReplayToAMemberParticipant(t *testing.T) {
	f := newReplayFixture(t)

	resp, queries := f.get(1, 10)
	if !resp.Success || resp.Locked || resp.MatchId != 10 || resp.GameType != 3 || resp.FrameCount != 8 || len(resp.Frames) != 8 || resp.FrameOnly || resp.DetailTruncated {
		t.Fatalf("full replay: %+v", resp)
	}
	if s := resp.Summary; s == nil || s.MyFrameWins != 5 || s.OpponentFrameWins != 3 || s.MyLongestStreak != 4 || s.OpponentLongestStreak != 2 {
		t.Fatalf("a 5:3 win with four straight racks: %+v", s)
	}
	if resp.Frames[0].Winner != 2 || resp.Frames[1].Winner != 1 {
		t.Fatalf("frame winners from the creator's view: %+v", resp.Frames[:2])
	}
	// 对局、用户、局记录、动作数、动作各 1 次。
	if queries != 5 {
		t.Fatalf("expected 5 bounded queries, got %d", queries)
	}
}

func TestMemberMatchReplayUsesTheParticipantsOwnPerspective(t *testing.T) {
	f := newReplayFixture(t)
	member := f.now.Add(30 * 24 * time.Hour)
	if err := f.db.Model(&model.User{}).Where("id = ?", 2).Update("member_expires_at", member).Error; err != nil {
		t.Fatalf("make opponent a member: %v", err)
	}

	resp, _ := f.get(2, 10)
	if !resp.Success || resp.Locked || resp.Summary.MyFrameWins != 3 || resp.Summary.OpponentFrameWins != 5 || resp.Summary.MyLongestStreak != 2 || resp.Summary.OpponentLongestStreak != 4 {
		t.Fatalf("the opponent sees the same match from their side: %+v", resp.Summary)
	}
	if resp.Frames[0].Winner != 1 || resp.Frames[1].Winner != 2 {
		t.Fatalf("frame winners must flip: %+v", resp.Frames[:2])
	}
}

func TestMemberMatchReplayLocksFreeAndExpiredParticipantsWithoutReplayData(t *testing.T) {
	for name, userID := range map[string]int64{"free opponent": 2} {
		t.Run(name, func(t *testing.T) {
			f := newReplayFixture(t)
			resp, queries := f.get(userID, 10)
			if !resp.Success || !resp.Locked || resp.MatchId != 10 || resp.GameType != 3 || resp.FrameCount != 8 {
				t.Fatalf("locked state with the frame count: %+v", resp)
			}
			if resp.Summary != nil || len(resp.Frames) != 0 {
				t.Fatalf("locked response must not carry replay data: %+v", resp)
			}
			raw, _ := json.Marshal(resp)
			var decoded map[string]any
			_ = json.Unmarshal(raw, &decoded)
			if decoded["summary"] != nil {
				t.Fatalf("locked JSON must have no summary: %v", decoded["summary"])
			}
			// 对局、用户、局数各 1 次，不读取动作。
			if queries != 3 {
				t.Fatalf("locked requests must not load actions, got %d queries", queries)
			}
		})
	}

	f := newReplayFixture(t)
	expired := f.now.Add(-time.Minute)
	if err := f.db.Model(&model.User{}).Where("id = ?", 1).Update("member_expires_at", expired).Error; err != nil {
		t.Fatalf("expire member: %v", err)
	}
	if resp, _ := f.get(1, 10); !resp.Locked || resp.Summary != nil {
		t.Fatalf("an expired member is locked: %+v", resp)
	}
	renewed := f.now.Add(24 * time.Hour)
	if err := f.db.Model(&model.User{}).Where("id = ?", 1).Update("member_expires_at", renewed).Error; err != nil {
		t.Fatalf("renew: %v", err)
	}
	if resp, _ := f.get(1, 10); resp.Locked || resp.Summary == nil {
		t.Fatalf("renewal restores the replay of the same match: %+v", resp)
	}
}

func TestMemberMatchReplayRejectsEveryNonParticipantTheSameWay(t *testing.T) {
	f := newReplayFixture(t)
	if err := f.db.Model(&model.Match{}).Where("id = ?", 10).Update("visibility", "private").Error; err != nil {
		t.Fatalf("make match private: %v", err)
	}

	missing, _ := f.get(1, 99999) // 对局不存在
	bystander, _ := f.get(4, 10)  // 与对局无关的会员
	referee, _ := f.get(3, 10)    // 裁判（会员）也不是参与者
	for name, resp := range map[string]*types.GetMemberMatchReplayResp{"bystander": bystander, "referee": referee} {
		if resp.Success || resp.Locked || resp.MatchId != 0 || resp.FrameCount != 0 || resp.Summary != nil || len(resp.Frames) != 0 {
			t.Fatalf("%s must be rejected without any match information: %+v", name, resp)
		}
		if !reflect.DeepEqual(resp, missing) {
			t.Fatalf("%s rejection must be indistinguishable from a missing match: got %+v want %+v", name, resp, missing)
		}
	}
}

func TestMemberMatchReplayIsNotAvailableForUnfinishedMatches(t *testing.T) {
	for name, status := range map[string]int{"in progress": 1, "cancelled": 3} {
		t.Run(name, func(t *testing.T) {
			f := newReplayFixture(t)
			if err := f.db.Model(&model.Match{}).Where("id = ?", 10).Update("status", status).Error; err != nil {
				t.Fatalf("set status: %v", err)
			}
			missing, _ := f.get(1, 99999)
			resp, _ := f.get(1, 10)
			if resp.Success || !reflect.DeepEqual(resp, missing) {
				t.Fatalf("an unfinished match must be rejected like a missing one: %+v", resp)
			}
		})
	}
}

func TestMemberMatchReplayOnlyReturnsFrameLevelDataForOversizedMatches(t *testing.T) {
	f := newReplayFixture(t)
	actions := make([]model.MatchAction, 0, 5000)
	for i := 0; i < 5000; i++ { // 已有 8 条胜局动作，再加 5000 条超过 5000 的上限
		actions = append(actions, model.MatchAction{MatchId: 10, RoundNo: 1, ActionType: "score", Actor: 1, ScoreChange: 1})
	}
	if err := f.db.CreateInBatches(&actions, 500).Error; err != nil {
		t.Fatalf("seed actions: %v", err)
	}

	resp, queries := f.get(1, 10)
	if !resp.Success || !resp.DetailTruncated || len(resp.Frames) != 8 || resp.Summary == nil || resp.Summary.MyFrameWins != 5 {
		t.Fatalf("oversized matches keep frame-level data: %+v", resp)
	}
	// 对局、用户、局记录、动作数各 1 次，超出上限后不再读取动作。
	if queries != 4 {
		t.Fatalf("oversized matches must not load their actions, got %d queries", queries)
	}
}

func TestMemberMatchReplayRejectsMissingRequestData(t *testing.T) {
	f := newReplayFixture(t)
	if resp, _ := f.get(1, 0); resp.Success {
		t.Fatalf("a request without a match id must fail: %+v", resp)
	}
	l := NewGetMemberMatchReplayLogic(context.Background(), f.svcCtx)
	if resp, _ := l.GetMemberMatchReplay(&types.GetMemberMatchReplayReq{MatchId: 10}); resp.Success {
		t.Fatalf("a request without a user must fail: %+v", resp)
	}
	if resp, _ := f.get(404, 10); resp.Success {
		t.Fatalf("an unknown user is not a participant: %+v", resp)
	}
}
