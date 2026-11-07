package auth

import (
	"testing"

	"chasing_points/internal/types"
)

func TestBuildBindPhoneSuccessRespForNormalBinding(t *testing.T) {
	resp := buildBindPhoneSuccessResp()

	if !resp.Success {
		t.Fatalf("expected success response")
	}
	if resp.MergedAccount {
		t.Fatalf("expected merged account flag to be false")
	}
	if resp.Message != "绑定成功" {
		t.Fatalf("unexpected message: %s", resp.Message)
	}
	if resp.NeedBindPhone {
		t.Fatalf("expected need_bind_phone to be false")
	}
	if resp.AccessToken != "" || resp.RefreshToken != "" || resp.UserInfo != nil {
		t.Fatalf("normal binding must not carry a replacement session: %+v", resp)
	}
}

func TestBuildBindPhoneSuccessRespUsesBindPhoneRespType(t *testing.T) {
	var _ *types.BindPhoneResp = buildBindPhoneSuccessResp()
}
