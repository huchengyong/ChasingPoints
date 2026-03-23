package logic

import (
	"testing"

	"billiard_master/internal/types"
)

func TestBuildBindPhoneSuccessRespForMergedAccount(t *testing.T) {
	resp := buildBindPhoneSuccessResp(true)

	if !resp.Success {
		t.Fatalf("expected success response")
	}
	if !resp.MergedAccount {
		t.Fatalf("expected merged account flag to be true")
	}
	if resp.Message != "账号已合并，请使用手机号登录" {
		t.Fatalf("unexpected message: %s", resp.Message)
	}
}

func TestBuildBindPhoneSuccessRespForNormalBinding(t *testing.T) {
	resp := buildBindPhoneSuccessResp(false)

	if !resp.Success {
		t.Fatalf("expected success response")
	}
	if resp.MergedAccount {
		t.Fatalf("expected merged account flag to be false")
	}
	if resp.Message != "绑定成功" {
		t.Fatalf("unexpected message: %s", resp.Message)
	}
}

func TestBuildBindPhoneSuccessRespUsesBindPhoneRespType(t *testing.T) {
	var _ *types.BindPhoneResp = buildBindPhoneSuccessResp(false)
}
