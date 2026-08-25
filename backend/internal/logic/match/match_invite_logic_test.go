package match

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg/matchinvite"
	"chasing_points/internal/types"
)

func TestMatchInviteQRCodeAndPreviewUseServerSignedIdentity(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	svcCtx.Config.Security.MatchInvite.SigningSecret = "invite-test-secret"
	svcCtx.Config.Security.MatchInvite.TTLSeconds = 300
	seedMatchLogicUser(t, svcCtx, 1001, "发起人")

	qrResp, err := NewGetMatchQRCodeLogic(matchLogicCtx(1001), svcCtx).GetMatchQRCode()
	if err != nil || !qrResp.Success || qrResp.QrcodeData == "" || qrResp.InviteToken == "" {
		t.Fatalf("issue QR invite: resp=%+v err=%v", qrResp, err)
	}
	if qrResp.QrcodeData != qrResp.InviteToken || qrResp.ExpiresInSeconds <= 0 {
		t.Fatalf("unexpected QR response: %+v", qrResp)
	}
	if qrResp.QrcodeData[0] == '{' {
		t.Fatalf("qrcode must not expose mutable user JSON")
	}

	preview, err := NewPreviewMatchInviteLogic(matchLogicCtx(2002), svcCtx).PreviewMatchInvite(&types.MatchInvitePreviewReq{
		InviteToken: qrResp.InviteToken,
	})
	if err != nil || !preview.Success || preview.Preview == nil {
		t.Fatalf("preview invite: resp=%+v err=%v", preview, err)
	}
	if preview.Preview.OpponentId != 1001 || preview.Preview.OpponentName != "发起人" {
		t.Fatalf("preview must use persisted inviter profile: %+v", preview.Preview)
	}
}

func TestMatchInvitePreviewRejectsExpiredAndTamperedTokens(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	svcCtx.Config.Security.MatchInvite.SigningSecret = "invite-test-secret"
	svcCtx.Config.Security.MatchInvite.TTLSeconds = 60
	seedMatchLogicUser(t, svcCtx, 1001, "发起人")

	signer, err := matchinvite.NewSigner("invite-test-secret", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	expired, _, err := signer.IssueAt(1001, time.Now().Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{expired, "x." + expired} {
		resp, err := NewPreviewMatchInviteLogic(matchLogicCtx(2002), svcCtx).PreviewMatchInvite(&types.MatchInvitePreviewReq{InviteToken: token})
		if err != nil || resp.Success {
			t.Fatalf("invalid invite must be rejected: resp=%+v err=%v", resp, err)
		}
	}
}

func TestStartMatchRequiresAndCanonicalizesInviteInProduction(t *testing.T) {
	svcCtx := newStartMatchChallengeTestSvc(t)
	svcCtx.Config.AppEnv = "production"
	svcCtx.Config.Security.MatchInvite.SigningSecret = "invite-test-secret"
	svcCtx.Config.Security.MatchInvite.TTLSeconds = 300
	for _, user := range []model.User{
		{Id: 1001, Nickname: "扫码者", Status: 1},
		{Id: 2002, Nickname: "可信对手", Avatar: "trusted.png", Status: 1},
	} {
		if err := svcCtx.UserModel.Create(&user); err != nil {
			t.Fatal(err)
		}
	}
	signer, err := matchinvite.NewSigner("invite-test-secret", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := signer.Issue(2002)
	if err != nil {
		t.Fatal(err)
	}

	missing, err := NewStartMatchLogic(startReputationCtx(1001), svcCtx).StartMatch(&types.StartMatchReq{GameType: 3})
	if err != nil || missing.Success || missing.Message != "请扫描有效的匹配二维码" {
		t.Fatalf("production must require invite: resp=%+v err=%v", missing, err)
	}
	tampered, err := NewStartMatchLogic(startReputationCtx(1001), svcCtx).StartMatch(&types.StartMatchReq{
		GameType: 3, InviteToken: token, OpponentId: 9999,
	})
	if err != nil || tampered.Success || tampered.Message != "匹配二维码对手不匹配" {
		t.Fatalf("client opponent substitution must fail: resp=%+v err=%v", tampered, err)
	}

	resp, err := NewStartMatchLogic(startReputationCtx(1001), svcCtx).StartMatch(&types.StartMatchReq{
		GameType:    3,
		InviteToken: token,
		MatchMode:   model.MatchModePractice,
		Visibility:  model.MatchVisibilityPrivate,
	})
	if err != nil || !resp.Success || resp.Action != startMatchActionCreated {
		t.Fatalf("start with invite: resp=%+v err=%v", resp, err)
	}
	stored, err := svcCtx.MatchModel.FindById(resp.MatchId)
	if err != nil || stored == nil || stored.OpponentId == nil || *stored.OpponentId != 2002 ||
		stored.OpponentName != "可信对手" {
		t.Fatalf("stored match must use invite identity: match=%+v err=%v", stored, err)
	}

	replayed, err := NewStartMatchLogic(startReputationCtx(1001), svcCtx).StartMatch(&types.StartMatchReq{
		GameType: 3, InviteToken: token, MatchMode: model.MatchModePractice, Visibility: model.MatchVisibilityPrivate,
	})
	if err != nil || !replayed.Success || replayed.Action != startMatchActionResumeExisting || replayed.MatchId != resp.MatchId {
		t.Fatalf("repeat invite must converge to existing match: resp=%+v err=%v", replayed, err)
	}
}
