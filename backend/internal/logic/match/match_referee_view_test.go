package match

import (
	"reflect"
	"testing"
	"time"

	"chasing_points/internal/model"
)

func TestBuildCurrentMatchInfoExposesRefereeRoleAndCapabilities(t *testing.T) {
	svcCtx := newCurrentMatchInfoTestSvc(t)
	now := time.Date(2026, 4, 8, 20, 0, 0, 0, time.FixedZone("CST", 8*3600))
	opponentID := int64(2002)
	refereeID := int64(3003)
	seedCurrentMatchInfoUser(t, svcCtx, 1001, "选手甲", "player1.png")
	seedCurrentMatchInfoUser(t, svcCtx, opponentID, "选手乙", "player2.png")
	seedCurrentMatchInfoUser(t, svcCtx, refereeID, "裁判丙", "referee.png")
	match := &model.Match{
		Id:           66,
		UserId:       1001,
		OpponentId:   &opponentID,
		OpponentName: "对手甲",
		GameType:     3,
		MyScore:      5,
		OpponentScore: 4,
		MatchTime:    now.Add(-12 * time.Minute),
		SyncRevision: 9,
	}
	setInt64Field(t, match, "RefereeUserId", refereeID)
	setTimeField(t, match, "RefereeJoinedAt", now.Add(-10*time.Minute))

	player1Info := buildCurrentMatchInfo(svcCtx, 1001, match)
	assertStructFieldEqual(t, player1Info, "ViewerRole", "player1")
	assertStructFieldEqual(t, player1Info, "RefereeBound", true)
	assertStructFieldEqual(t, player1Info, "RefereeUserId", refereeID)
	assertStructFieldEqual(t, player1Info, "CanScore", false)
	assertStructFieldEqual(t, player1Info, "CanUndo", false)
	assertStructFieldEqual(t, player1Info, "CanFinish", false)

	player2Info := buildCurrentMatchInfo(svcCtx, opponentID, match)
	assertStructFieldEqual(t, player2Info, "ViewerRole", "player2")
	assertStructFieldEqual(t, player2Info, "RefereeBound", true)
	assertStructFieldEqual(t, player2Info, "CanScore", false)

	refereeInfo := buildCurrentMatchInfo(svcCtx, refereeID, match)
	assertStructFieldEqual(t, refereeInfo, "ViewerRole", "referee")
	assertStructFieldEqual(t, refereeInfo, "RefereeBound", true)
	assertStructFieldEqual(t, refereeInfo, "RefereeUserId", refereeID)
	assertStructFieldEqual(t, refereeInfo, "CanScore", true)
	assertStructFieldEqual(t, refereeInfo, "CanUndo", true)
	assertStructFieldEqual(t, refereeInfo, "CanFinish", true)
	assertStructFieldEqual(t, refereeInfo, "RefereeName", "裁判丙")
	assertFixedParticipants(t, refereeInfo, 1001, "选手甲", "player1.png", opponentID, "选手乙", "player2.png")
	assertStructFieldEqual(t, refereeInfo, "OpponentId", opponentID)
	assertStructFieldEqual(t, refereeInfo, "OpponentName", "选手乙")
	assertStructFieldEqual(t, refereeInfo, "MyScore", 5)
	assertStructFieldEqual(t, refereeInfo, "OpponentScore", 4)
}

func TestBuildMatchSyncSnapshotForUserCarriesRoleCapabilities(t *testing.T) {
	opponentID := int64(2002)
	refereeID := int64(3003)
	match := &model.Match{
		Id:                        67,
		UserId:                    1001,
		OpponentId:                &opponentID,
		GameType:                  1,
		MyScore:                   6,
		OpponentScore:             3,
		CurrentFrameMyScore:       41,
		CurrentFrameOpponentScore: 18,
		CurrentFrameStarted:       true,
		Status:                    1,
		SyncRevision:              12,
	}
	setInt64Field(t, match, "RefereeUserId", refereeID)

	playerSnapshot := buildMatchSyncSnapshotForUser(1001, match, 2, model.SnookerRoundState{})
	assertStructFieldEqual(t, &playerSnapshot, "ViewerRole", "player1")
	assertStructFieldEqual(t, &playerSnapshot, "RefereeBound", true)
	assertStructFieldEqual(t, &playerSnapshot, "CanScore", false)
	assertStructFieldEqual(t, &playerSnapshot, "CanFinish", false)

	refereeSnapshot := buildMatchSyncSnapshotForUser(refereeID, match, 2, model.SnookerRoundState{})
	assertStructFieldEqual(t, &refereeSnapshot, "ViewerRole", "referee")
	assertStructFieldEqual(t, &refereeSnapshot, "RefereeBound", true)
	assertStructFieldEqual(t, &refereeSnapshot, "RefereeUserId", refereeID)
	assertStructFieldEqual(t, &refereeSnapshot, "CanScore", true)
	assertStructFieldEqual(t, &refereeSnapshot, "CanUndo", true)
	assertStructFieldEqual(t, &refereeSnapshot, "CanFinish", true)
}

func setInt64Field(t *testing.T, target any, fieldName string, value int64) {
	t.Helper()
	field := reflect.ValueOf(target).Elem().FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("expected field %s to exist on %T", fieldName, target)
	}
	switch field.Kind() {
	case reflect.Int64:
		field.SetInt(value)
	case reflect.Ptr:
		ptr := reflect.New(field.Type().Elem())
		ptr.Elem().SetInt(value)
		field.Set(ptr)
	default:
		t.Fatalf("field %s has unsupported kind %s", fieldName, field.Kind())
	}
}

func setTimeField(t *testing.T, target any, fieldName string, value time.Time) {
	t.Helper()
	field := reflect.ValueOf(target).Elem().FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("expected field %s to exist on %T", fieldName, target)
	}
	switch field.Kind() {
	case reflect.Struct:
		field.Set(reflect.ValueOf(value))
	case reflect.Ptr:
		ptr := reflect.New(field.Type().Elem())
		ptr.Elem().Set(reflect.ValueOf(value))
		field.Set(ptr)
	default:
		t.Fatalf("field %s has unsupported kind %s", fieldName, field.Kind())
	}
}

func assertStructFieldEqual(t *testing.T, target any, fieldName string, want any) {
	t.Helper()
	value := reflect.ValueOf(target)
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			t.Fatalf("expected %s on non-nil target", fieldName)
		}
		value = value.Elem()
	}
	field := value.FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("expected field %s on %T", fieldName, target)
	}
	got := field.Interface()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %s=%v, got %v", fieldName, want, got)
	}
}
