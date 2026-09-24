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

func newCurrentMatchInfoTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("prepare user schema: %v", err)
	}

	return &svc.ServiceContext{UserModel: model.NewUserModel(db)}
}

func seedCurrentMatchInfoUser(t *testing.T, svcCtx *svc.ServiceContext, id int64, nickname, avatar string) {
	t.Helper()
	if err := svcCtx.UserModel.Create(&model.User{Id: id, Nickname: nickname, Avatar: avatar, Status: 1}); err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

func TestBuildCurrentMatchInfoIncludesSnookerCurrentFrameFromViewerPerspective(t *testing.T) {
	now := time.Date(2026, 3, 17, 20, 0, 0, 0, time.FixedZone("CST", 8*3600))
	opponentID := int64(2002)
	match := &model.Match{
		Id:                        18,
		UserId:                    1001,
		OpponentId:                &opponentID,
		OpponentName:              "对手甲",
		GameType:                  1,
		GameMode:                  "",
		MyScore:                   3,
		OpponentScore:             2,
		CurrentFrameMyScore:       46,
		CurrentFrameOpponentScore: 33,
		CurrentFrameStarted:       true,
		SyncRevision:              7,
		MatchTime:                 now.Add(-18 * time.Minute),
	}

	info := buildCurrentMatchInfo(nil, 1001, match)
	assertCurrentMatchInfo(t, info, 3, 2, 46, 33, true)

	info = buildCurrentMatchInfo(nil, opponentID, match)
	assertCurrentMatchInfo(t, info, 2, 3, 33, 46, true)
}

func TestBuildCurrentMatchInfoKeepsFixedParticipantsAcrossPlayerPerspectives(t *testing.T) {
	svcCtx := newCurrentMatchInfoTestSvc(t)
	opponentID := int64(2002)
	seedCurrentMatchInfoUser(t, svcCtx, 1001, "选手甲", "player1.png")
	seedCurrentMatchInfoUser(t, svcCtx, opponentID, "选手乙", "player2.png")
	match := &model.Match{
		Id:            19,
		UserId:        1001,
		OpponentId:    &opponentID,
		OpponentName:  "旧对手名",
		GameType:      3,
		MyScore:       5,
		OpponentScore: 4,
		MatchTime:     time.Now(),
	}

	player1Info := buildCurrentMatchInfo(svcCtx, 1001, match)
	assertFixedParticipants(t, player1Info, 1001, "选手甲", "player1.png", opponentID, "选手乙", "player2.png")
	if player1Info.OpponentId != opponentID || player1Info.OpponentName != "选手乙" || player1Info.OpponentAvatar != "player2.png" {
		t.Fatalf("unexpected player1 opponent view: %#v", player1Info)
	}

	player2Info := buildCurrentMatchInfo(svcCtx, opponentID, match)
	assertFixedParticipants(t, player2Info, 1001, "选手甲", "player1.png", opponentID, "选手乙", "player2.png")
	if player2Info.OpponentId != 1001 || player2Info.OpponentName != "选手甲" || player2Info.OpponentAvatar != "player1.png" {
		t.Fatalf("unexpected player2 opponent view: %#v", player2Info)
	}
	if player2Info.MyScore != 4 || player2Info.OpponentScore != 5 {
		t.Fatalf("expected player2 score view 4:5, got %d:%d", player2Info.MyScore, player2Info.OpponentScore)
	}
}

func TestBuildCurrentMatchInfoFallsBackForEmptyAndUnregisteredParticipants(t *testing.T) {
	svcCtx := newCurrentMatchInfoTestSvc(t)
	seedCurrentMatchInfoUser(t, svcCtx, 1001, "", "")
	match := &model.Match{
		Id:           20,
		UserId:       1001,
		OpponentName: "线下球友",
		GameType:     3,
		MatchTime:    time.Now(),
	}

	info := buildCurrentMatchInfo(svcCtx, 1001, match)
	assertFixedParticipants(t, info, 1001, "玩家1", "", 0, "线下球友", "")
	if info.OpponentId != 0 || info.OpponentName != "线下球友" || info.OpponentAvatar != "" {
		t.Fatalf("unexpected unregistered opponent view: %#v", info)
	}

	opponentID := int64(2002)
	seedCurrentMatchInfoUser(t, svcCtx, opponentID, "", "")
	match.OpponentId = &opponentID
	info = buildCurrentMatchInfo(svcCtx, 1001, match)
	assertFixedParticipants(t, info, 1001, "玩家1", "", opponentID, "线下球友", "")
}

func assertFixedParticipants(t *testing.T, info *types.CurrentMatchInfo, player1ID int64, player1Name, player1Avatar string, player2ID int64, player2Name, player2Avatar string) {
	t.Helper()
	if info == nil {
		t.Fatalf("expected current match info")
	}
	if info.Player1Id != player1ID || info.Player1Name != player1Name || info.Player1Avatar != player1Avatar {
		t.Fatalf("unexpected player1 fields: %#v", info)
	}
	if info.Player2Id != player2ID || info.Player2Name != player2Name || info.Player2Avatar != player2Avatar {
		t.Fatalf("unexpected player2 fields: %#v", info)
	}
}

func assertCurrentMatchInfo(t *testing.T, info *types.CurrentMatchInfo, myScore, opponentScore, frameMy, frameOpponent int, started bool) {
	t.Helper()
	if info == nil {
		t.Fatalf("expected current match info")
	}
	if info.MyScore != myScore || info.OpponentScore != opponentScore {
		t.Fatalf("expected match score %d:%d, got %d:%d", myScore, opponentScore, info.MyScore, info.OpponentScore)
	}
	if info.CurrentFrameMyScore != frameMy || info.CurrentFrameOpponentScore != frameOpponent {
		t.Fatalf("expected current frame score %d:%d, got %d:%d", frameMy, frameOpponent, info.CurrentFrameMyScore, info.CurrentFrameOpponentScore)
	}
	if info.CurrentFrameStarted != started {
		t.Fatalf("expected current_frame_started=%v, got %v", started, info.CurrentFrameStarted)
	}
	if info.ServerRevision != 7 {
		t.Fatalf("expected server_revision=7, got %d", info.ServerRevision)
	}
}
