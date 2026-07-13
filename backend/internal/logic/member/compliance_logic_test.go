package member

import (
	"context"
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func TestCreateMemberSubscriptionOrderBlockedInComplianceMode(t *testing.T) {
	svcCtx := &svc.ServiceContext{
		Config: config.Config{
			Compliance: config.ComplianceConfig{
				RestrictedMode: true,
			},
		},
	}

	ctx := context.WithValue(context.Background(), "user_id", int64(1001))
	logic := NewCreateMemberSubscriptionOrderLogic(ctx, svcCtx)
	resp, err := logic.CreateMemberSubscriptionOrder(&types.CreateMemberSubscriptionOrderReq{
		PlanCode:   "member_monthly",
		PayChannel: "alipay",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected disabled response, got %#v", resp)
	}
	if resp.Message == "" {
		t.Fatalf("expected disable message, got %#v", resp)
	}
}
