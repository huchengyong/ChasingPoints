package match

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRefereeHistoryTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}); err != nil {
		t.Fatalf("prepare referee history schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:         db,
		UserModel:  model.NewUserModel(db),
		MatchModel: model.NewMatchModel(db),
	}
}

func seedRefereeHistoryUser(t *testing.T, svcCtx *svc.ServiceContext, id int64, nickname, avatar string) {
	t.Helper()
	if err := svcCtx.UserModel.Create(&model.User{Id: id, Nickname: nickname, Avatar: avatar, Status: 1}); err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

func seedRefereeHistoryMatch(t *testing.T, svcCtx *svc.ServiceContext, match *model.Match) {
	t.Helper()
	if err := svcCtx.MatchModel.Create(match); err != nil {
		t.Fatalf("create match %d: %v", match.Id, err)
	}
}

func TestGetRefereeHistoryReturnsCompletedAndCancelledMatches(t *testing.T) {
	svcCtx := newRefereeHistoryTestSvc(t)
	refereeID := int64(100)
	now := time.Date(2026, 7, 15, 14, 0, 0, 0, time.UTC)
	opponentID := int64(200)

	seedRefereeHistoryUser(t, svcCtx, 10, "选手甲", "p1.png")
	seedRefereeHistoryUser(t, svcCtx, opponentID, "选手乙", "p2.png")
	seedRefereeHistoryUser(t, svcCtx, refereeID, "裁判长", "ref.png")

	joinedAt := now.Add(-30 * time.Minute)
	win := 1

	// Completed match
	endTime2 := now.Add(-5 * time.Minute)
	seedRefereeHistoryMatch(t, svcCtx, &model.Match{
		Id:               201,
		UserId:           10,
		OpponentId:       &opponentID,
		OpponentName:     "选手乙",
		GameType:         3,
		Visibility:       model.MatchVisibilityPrivate,
		Status:           2,
		Result:           &win,
		RefereeUserId:    &refereeID,
		RefereeJoinedAt:  &joinedAt,
		EndTime:          &endTime2,
		MyScore:          5,
		OpponentScore:    3,
		MatchTime:        now.Add(-1 * time.Hour),
		CompletionSource: model.CompletionSourceReferee,
	})

	// Cancelled match (more recent end_time → should come first)
	endTime3 := now
	legacyCompletedBy := refereeID
	seedRefereeHistoryMatch(t, svcCtx, &model.Match{
		Id:                202,
		UserId:            10,
		OpponentId:        &opponentID,
		OpponentName:      "选手乙",
		GameType:          1,
		Status:            3,
		RefereeUserId:     &refereeID,
		RefereeJoinedAt:   &joinedAt,
		EndTime:           &endTime3,
		MyScore:           2,
		OpponentScore:     1,
		MatchTime:         now.Add(-2 * time.Hour),
		CompletedByUserId: &legacyCompletedBy,
		CompletionSource:  "player_cancelled",
	})

	resp, err := NewGetRefereeHistoryLogic(refereeTestCtx(refereeID), svcCtx).GetRefereeHistory(&types.RefereeHistoryReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("GetRefereeHistory: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 2 {
		t.Fatalf("expected total=2, got %d", resp.Total)
	}
	if len(resp.List) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.List))
	}

	// Cancelled match (endTime = now) should come first
	if resp.List[0].Id != 202 {
		t.Fatalf("expected cancelled match (more recent end_time) first, got match %d", resp.List[0].Id)
	}
	if resp.List[0].Status != 3 || resp.List[0].StatusText != "已取消" {
		t.Fatalf("match 202 expected status 3/已取消, got %d/%s", resp.List[0].Status, resp.List[0].StatusText)
	}
	if resp.List[0].CompletionSource != model.CompletionSourceUnknown || resp.List[0].CompletedByUserId != 0 {
		t.Fatalf("cancelled match must hide legacy completion attribution, got %+v", resp.List[0])
	}
	if resp.List[1].Id != 201 {
		t.Fatalf("expected completed match second, got match %d", resp.List[1].Id)
	}
	if resp.List[1].Status != 2 || resp.List[1].StatusText != "已完成" {
		t.Fatalf("match 201 expected status 2/已完成, got %d/%s", resp.List[1].Status, resp.List[1].StatusText)
	}
	if resp.List[1].CompletionSource != "referee" {
		t.Fatalf("match 201 expected completion_source 'referee', got '%s'", resp.List[1].CompletionSource)
	}
}

func TestGetRefereeHistoryKeepsUnknownCompletionForCompletedLegacyMatch(t *testing.T) {
	svcCtx := newRefereeHistoryTestSvc(t)
	refereeID := int64(500)
	opponentID := int64(600)
	now := time.Now()
	endTime := now
	seedRefereeHistoryUser(t, svcCtx, 400, "选手甲", "")
	seedRefereeHistoryUser(t, svcCtx, opponentID, "选手乙", "")
	seedRefereeHistoryUser(t, svcCtx, refereeID, "裁判", "")
	seedRefereeHistoryMatch(t, svcCtx, &model.Match{
		Id: 401, UserId: 400, OpponentId: &opponentID, OpponentName: "选手乙", GameType: 3,
		Status: 2, RefereeUserId: &refereeID, RefereeJoinedAt: &now, EndTime: &endTime,
		MatchTime: now.Add(-time.Hour), CompletionSource: model.CompletionSourceUnknown,
	})

	resp, err := NewGetRefereeHistoryLogic(refereeTestCtx(refereeID), svcCtx).GetRefereeHistory(&types.RefereeHistoryReq{Page: 1, PageSize: 20})
	if err != nil || !resp.Success || len(resp.List) != 1 {
		t.Fatalf("load legacy history: resp=%#v err=%v", resp, err)
	}
	if resp.List[0].CompletedByUserId != 0 || resp.List[0].CompletionSource != model.CompletionSourceUnknown {
		t.Fatalf("legacy completion attribution must remain unknown: %+v", resp.List[0])
	}
}

func TestGetRefereeHistoryHandlesUnknownUserGracefully(t *testing.T) {
	svcCtx := newRefereeHistoryTestSvc(t)

	// User ID 404 exists in context but has no referee history in DB
	resp, err := NewGetRefereeHistoryLogic(refereeTestCtx(404), svcCtx).GetRefereeHistory(&types.RefereeHistoryReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("GetRefereeHistory: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success for unknown user with no history, got %#v", resp)
	}
	if resp.Total != 0 {
		t.Fatalf("expected total=0, got %d", resp.Total)
	}
	if len(resp.List) != 0 {
		t.Fatalf("expected empty list, got %d items", len(resp.List))
	}
}

func TestGetRefereeHistoryPaginationDefaults(t *testing.T) {
	svcCtx := newRefereeHistoryTestSvc(t)
	refereeID := int64(300)
	now := time.Now()
	opponentID := int64(400)

	seedRefereeHistoryUser(t, svcCtx, 20, "选手1", "")
	seedRefereeHistoryUser(t, svcCtx, opponentID, "选手2", "")
	seedRefereeHistoryUser(t, svcCtx, refereeID, "裁判", "")

	endTime := now
	win := 1
	for i := int64(1); i <= 3; i++ {
		seedRefereeHistoryMatch(t, svcCtx, &model.Match{
			Id:               300 + i,
			UserId:           20,
			OpponentId:       &opponentID,
			OpponentName:     "选手2",
			GameType:         3,
			Status:           2,
			Result:           &win,
			RefereeUserId:    &refereeID,
			RefereeJoinedAt:  &now,
			EndTime:          &endTime,
			MyScore:          3,
			OpponentScore:    1,
			MatchTime:        now.Add(-time.Duration(i) * time.Hour),
			CompletionSource: model.CompletionSourceReferee,
		})
	}

	// Test default page size (should work with page=1, pageSize=2)
	resp, err := NewGetRefereeHistoryLogic(refereeTestCtx(refereeID), svcCtx).GetRefereeHistory(&types.RefereeHistoryReq{
		Page:     1,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("GetRefereeHistory: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %#v", resp)
	}
	if resp.Total != 3 {
		t.Fatalf("expected total=3, got %d", resp.Total)
	}
	if len(resp.List) != 2 {
		t.Fatalf("expected 2 items in first page, got %d", len(resp.List))
	}

	// Test invalid page/limit defaults
	resp, err = NewGetRefereeHistoryLogic(refereeTestCtx(refereeID), svcCtx).GetRefereeHistory(&types.RefereeHistoryReq{
		Page:     -1,
		PageSize: 200,
	})
	if err != nil {
		t.Fatalf("GetRefereeHistory with bad params: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success with corrected params, got %#v", resp)
	}
	// page clamped to 1, pageSize clamped to 100
	if resp.Total != 3 {
		t.Fatalf("expected total=3, got %d", resp.Total)
	}
}

func TestGetRefereeHistoryCalculatesRefereeDuration(t *testing.T) {
	svcCtx := newRefereeHistoryTestSvc(t)
	refereeID := int64(500)
	now := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
	opponentID := int64(600)

	seedRefereeHistoryUser(t, svcCtx, 30, "选手A", "")
	seedRefereeHistoryUser(t, svcCtx, opponentID, "选手B", "")
	seedRefereeHistoryUser(t, svcCtx, refereeID, "裁判X", "")

	joinedAt := now.Add(-45 * time.Minute) // referee joined 45 min ago
	endTime := now                         // ended now
	win := 1

	seedRefereeHistoryMatch(t, svcCtx, &model.Match{
		Id:               501,
		UserId:           30,
		OpponentId:       &opponentID,
		OpponentName:     "选手B",
		GameType:         3,
		Status:           2,
		Result:           &win,
		RefereeUserId:    &refereeID,
		RefereeJoinedAt:  &joinedAt,
		EndTime:          &endTime,
		MyScore:          5,
		OpponentScore:    2,
		MatchTime:        now.Add(-1 * time.Hour),
		CompletionSource: model.CompletionSourceReferee,
	})

	resp, err := NewGetRefereeHistoryLogic(refereeTestCtx(refereeID), svcCtx).GetRefereeHistory(&types.RefereeHistoryReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("GetRefereeHistory: %v", err)
	}
	if !resp.Success || len(resp.List) != 1 {
		t.Fatalf("expected 1 item, got %#v", resp)
	}

	item := resp.List[0]
	expectedDuration := int64(45 * 60) // 2700 seconds
	if item.RefereeDurationSeconds != expectedDuration {
		t.Fatalf("expected referee duration %d seconds, got %d", expectedDuration, item.RefereeDurationSeconds)
	}
	if item.Player1Name != "选手A" {
		t.Fatalf("expected player1 name '选手A', got '%s'", item.Player1Name)
	}
	if item.Player2Name != "选手B" {
		t.Fatalf("expected player2 name '选手B', got '%s'", item.Player2Name)
	}
	if item.GameTypeName != "中式八球" {
		t.Fatalf("expected game type name '中式八球', got '%s'", item.GameTypeName)
	}
}

func TestGetRefereeHistoryCompletedByUserIdFromModel(t *testing.T) {
	svcCtx := newRefereeHistoryTestSvc(t)
	refereeID := int64(700)
	now := time.Now()
	opponentID := int64(800)

	seedRefereeHistoryUser(t, svcCtx, 40, "我方", "")
	seedRefereeHistoryUser(t, svcCtx, opponentID, "对手", "")
	seedRefereeHistoryUser(t, svcCtx, refereeID, "裁判", "")

	completedByUserID := int64(refereeID) // referee completed it
	win := 1
	endTime := now

	seedRefereeHistoryMatch(t, svcCtx, &model.Match{
		Id:                601,
		UserId:            40,
		OpponentId:        &opponentID,
		OpponentName:      "对手",
		GameType:          3,
		Status:            2,
		Result:            &win,
		RefereeUserId:     &refereeID,
		RefereeJoinedAt:   &now,
		EndTime:           &endTime,
		CompletedByUserId: &completedByUserID,
		CompletionSource:  model.CompletionSourceReferee,
		MyScore:           5,
		OpponentScore:     3,
		MatchTime:         now.Add(-30 * time.Minute),
	})

	resp, err := NewGetRefereeHistoryLogic(refereeTestCtx(refereeID), svcCtx).GetRefereeHistory(&types.RefereeHistoryReq{
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("GetRefereeHistory: %v", err)
	}
	if !resp.Success || len(resp.List) != 1 {
		t.Fatalf("expected 1 item, got %#v", resp)
	}

	item := resp.List[0]
	if item.CompletedByUserId != refereeID {
		t.Fatalf("expected completed_by_user_id=%d, got %d", refereeID, item.CompletedByUserId)
	}
	if item.CompletionSource != "referee" {
		t.Fatalf("expected completion_source='referee', got '%s'", item.CompletionSource)
	}
}
