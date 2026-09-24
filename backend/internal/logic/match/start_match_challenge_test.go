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

func newStartMatchChallengeTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.Opponent{}, &model.Challenge{}, &model.MatchRound{}, &model.MatchAction{}); err != nil {
		t.Fatalf("prepare challenge start schema: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get challenge start sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return &svc.ServiceContext{
		DB:             db,
		UserModel:      model.NewUserModel(db),
		MatchModel:     model.NewMatchModel(db),
		ChallengeModel: model.NewChallengeModel(db),
	}
}

func TestStartMatchPersistsFinishConfirmationCompatibility(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		required bool
	}{
		{name: "legacy omitted mode", mode: "", required: false},
		{name: "explicit ranked", mode: model.MatchModeRanked, required: true},
	}

	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcCtx := newStartMatchChallengeTestSvc(t)
			userID := int64(1100 + index*10)
			opponentID := userID + 1
			for _, user := range []model.User{{Id: userID, Nickname: "发起人"}, {Id: opponentID, Nickname: "对手"}} {
				if err := svcCtx.UserModel.Create(&user); err != nil {
					t.Fatalf("create user %d: %v", user.Id, err)
				}
			}
			resp, err := NewStartMatchLogic(startReputationCtx(userID), svcCtx).StartMatch(&types.StartMatchReq{
				GameType:     3,
				OpponentId:   opponentID,
				OpponentName: "对手",
				MatchMode:    tt.mode,
			})
			if err != nil || !resp.Success || resp.MatchId <= 0 {
				t.Fatalf("start match: resp=%#v err=%v", resp, err)
			}
			stored, err := svcCtx.MatchModel.FindById(resp.MatchId)
			if err != nil || stored == nil || stored.FinishConfirmationRequired != tt.required {
				t.Fatalf("unexpected compatibility flag: stored=%+v err=%v", stored, err)
			}
		})
	}
}

func TestStartMatchIgnoresChallengeContextAndClearsOwnWaiting(t *testing.T) {
	svcCtx := newStartMatchChallengeTestSvc(t)
	for _, user := range []model.User{{Id: 1001, Nickname: "发起人"}, {Id: 2002, Nickname: "对手"}, {Id: 3003, Nickname: "第三人"}} {
		if err := svcCtx.UserModel.Create(&user); err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	scheduled := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	if err := svcCtx.ChallengeModel.Create(&model.Challenge{
		Id:            9101,
		FromUserId:    1001,
		ToUserId:      3003,
		GameType:      3,
		ScheduledDate: &scheduled,
		MatchMode:     model.MatchModePractice,
		Visibility:    model.MatchVisibilityPrivate,
		Status:        model.ChallengeStatusAccepted,
		WaitingUserId: challengeInt64Ptr(1001),
		ExpiresAt:     time.Now().Add(12 * time.Hour),
	}); err != nil {
		t.Fatalf("seed challenge: %v", err)
	}

	// 普通开局（无签名，测试环境允许 opponent_id），扫码开别场成功。
	resp, err := NewStartMatchLogic(startReputationCtx(1001), svcCtx).StartMatch(&types.StartMatchReq{
		GameType:     3,
		OpponentId:   2002,
		OpponentName: "第三人对手",
		MatchMode:    model.MatchModePractice,
		Visibility:   model.MatchVisibilityPrivate,
	})
	if err != nil || !resp.Success || resp.MatchId <= 0 {
		t.Fatalf("start scan match: resp=%#v err=%v", resp, err)
	}
	stored, err := svcCtx.ChallengeModel.FindById(9101)
	if err != nil || stored == nil {
		t.Fatalf("load challenge: %v", err)
	}
	if stored.Status != model.ChallengeStatusAccepted || stored.WaitingUserId != nil {
		t.Fatalf("scan start must clear own waiting but keep challenge: %+v", stored)
	}
	if stored.MatchId != nil {
		t.Fatalf("challenge must not link scan match: %+v", stored)
	}
}

func challengeInt64Ptr(value int64) *int64 {
	copy := value
	return &copy
}
