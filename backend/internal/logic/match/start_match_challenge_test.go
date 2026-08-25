package match

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/matchinvite"
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

func TestStartMatchLinksAcceptedChallengeOnlyForMatchingParticipantsAndGameType(t *testing.T) {
	svcCtx := newStartMatchChallengeTestSvc(t)
	for _, user := range []model.User{
		{Id: 1001, Nickname: "发起人"},
		{Id: 2002, Nickname: "对手"},
	} {
		if err := svcCtx.UserModel.Create(&user); err != nil {
			t.Fatalf("create user %d: %v", user.Id, err)
		}
	}
	challenge := &model.Challenge{
		Id:         9101,
		FromUserId: 1001,
		ToUserId:   2002,
		GameType:   3,
		Status:     1,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	if err := svcCtx.ChallengeModel.Create(challenge); err != nil {
		t.Fatalf("create accepted challenge: %v", err)
	}

	resp, err := NewStartMatchLogic(startReputationCtx(1001), svcCtx).StartMatch(&types.StartMatchReq{
		GameType:     3,
		OpponentId:   2002,
		OpponentName: "对手",
		MatchMode:    model.MatchModePractice,
		Visibility:   model.MatchVisibilityPrivate,
		ChallengeId:  9101,
	})
	if err != nil || !resp.Success || resp.Action != startMatchActionCreated || resp.MatchId <= 0 {
		t.Fatalf("expected linked challenge match, resp=%#v err=%v", resp, err)
	}
	match, err := svcCtx.MatchModel.FindById(resp.MatchId)
	if err != nil || match == nil || match.MatchMode != model.MatchModePractice || match.Visibility != model.MatchVisibilityPrivate || match.FinishConfirmationRequired {
		t.Fatalf("unexpected linked match: match=%+v err=%v", match, err)
	}
	linked, err := svcCtx.ChallengeModel.FindById(9101)
	if err != nil || linked == nil || linked.MatchId == nil || *linked.MatchId != resp.MatchId {
		t.Fatalf("expected challenge to link match %d, challenge=%+v err=%v", resp.MatchId, linked, err)
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

func TestStartMatchRejectsChallengeWithWrongIdentityOrGameTypeWithoutMutation(t *testing.T) {
	tests := []struct {
		name       string
		fromUserID int64
		toUserID   int64
		gameType   int
		reqType    int
		message    string
	}{
		{
			name:       "wrong identity",
			fromUserID: 3003,
			toUserID:   4004,
			gameType:   3,
			reqType:    3,
			message:    "邀约对手或球种不匹配",
		},
		{
			name:       "wrong game type",
			fromUserID: 1001,
			toUserID:   2002,
			gameType:   2,
			reqType:    3,
			message:    "邀约对手或球种不匹配",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcCtx := newStartMatchChallengeTestSvc(t)
			for _, user := range []model.User{
				{Id: 1001, Nickname: "发起人"},
				{Id: 2002, Nickname: "对手"},
			} {
				if err := svcCtx.UserModel.Create(&user); err != nil {
					t.Fatalf("create user %d: %v", user.Id, err)
				}
			}
			if err := svcCtx.ChallengeModel.Create(&model.Challenge{
				Id:         9102,
				FromUserId: tt.fromUserID,
				ToUserId:   tt.toUserID,
				GameType:   tt.gameType,
				Status:     1,
				ExpiresAt:  time.Now().Add(24 * time.Hour),
			}); err != nil {
				t.Fatalf("create challenge: %v", err)
			}

			resp, err := NewStartMatchLogic(startReputationCtx(1001), svcCtx).StartMatch(&types.StartMatchReq{
				GameType:     tt.reqType,
				OpponentId:   2002,
				OpponentName: "对手",
				MatchMode:    model.MatchModeRanked,
				Visibility:   model.MatchVisibilityPublic,
				ChallengeId:  9102,
			})
			if err != nil || resp.Success || resp.Message != tt.message {
				t.Fatalf("expected challenge rejection, resp=%#v err=%v", resp, err)
			}
			var matchCount int64
			if err := svcCtx.DB.Model(&model.Match{}).Count(&matchCount).Error; err != nil {
				t.Fatalf("count matches: %v", err)
			}
			if matchCount != 0 {
				t.Fatalf("rejected challenge must not create match, got %d", matchCount)
			}
			stored, err := svcCtx.ChallengeModel.FindById(9102)
			if err != nil || stored == nil || stored.MatchId != nil {
				t.Fatalf("rejected challenge must remain unlinked, stored=%+v err=%v", stored, err)
			}
		})
	}
}

func TestStartMatchConvergesConcurrentAcceptedChallengeStarts(t *testing.T) {
	svcCtx := newStartMatchChallengeTestSvc(t)
	svcCtx.Config.AppEnv = "production"
	svcCtx.Config.Security.MatchInvite.SigningSecret = "invite-test-secret"
	for _, user := range []model.User{
		{Id: 1001, Nickname: "发起人", Status: 1},
		{Id: 2002, Nickname: "对手", Status: 1},
	} {
		if err := svcCtx.UserModel.Create(&user); err != nil {
			t.Fatalf("create user %d: %v", user.Id, err)
		}
	}
	if err := svcCtx.ChallengeModel.Create(&model.Challenge{
		Id:         9103,
		FromUserId: 1001,
		ToUserId:   2002,
		GameType:   3,
		Status:     1,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("create accepted challenge: %v", err)
	}

	signer, err := matchinvite.NewSigner("invite-test-secret", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	tokenFrom1001, _, err := signer.Issue(1001)
	if err != nil {
		t.Fatal(err)
	}
	tokenFrom2002, _, err := signer.Issue(2002)
	if err != nil {
		t.Fatal(err)
	}

	type startResult struct {
		resp *types.StartMatchResp
		err  error
	}
	start := make(chan struct{})
	results := make(chan startResult, 2)
	for _, input := range []struct {
		userID int64
		token  string
	}{
		{userID: 1001, token: tokenFrom2002},
		{userID: 2002, token: tokenFrom1001},
	} {
		input := input
		go func() {
			<-start
			resp, startErr := NewStartMatchLogic(startReputationCtx(input.userID), svcCtx).StartMatch(&types.StartMatchReq{
				GameType:    3,
				InviteToken: input.token,
				MatchMode:   model.MatchModePractice,
				Visibility:  model.MatchVisibilityPrivate,
				ChallengeId: 9103,
			})
			results <- startResult{resp: resp, err: startErr}
		}()
	}
	close(start)

	first, second := <-results, <-results
	for _, result := range []startResult{first, second} {
		if result.err != nil || result.resp == nil || !result.resp.Success || result.resp.MatchId <= 0 {
			t.Fatalf("concurrent challenge start must converge: resp=%+v err=%v", result.resp, result.err)
		}
	}
	if first.resp.MatchId != second.resp.MatchId {
		t.Fatalf("concurrent challenge starts must return one match: first=%d second=%d", first.resp.MatchId, second.resp.MatchId)
	}

	var matchCount int64
	if err := svcCtx.DB.Model(&model.Match{}).Count(&matchCount).Error; err != nil {
		t.Fatalf("count matches: %v", err)
	}
	if matchCount != 1 {
		t.Fatalf("accepted challenge must create only one match, got %d", matchCount)
	}
}
