package member

import (
	"context"

	logicx "chasing_points/internal/logic"
	paymentlogic "chasing_points/internal/logic/payment"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMemberPlansLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取会员套餐列表
func NewGetMemberPlansLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMemberPlansLogic {
	return &GetMemberPlansLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMemberPlansLogic) GetMemberPlans() (resp *types.GetMemberPlansResp, err error) {
	if !l.svcCtx.Config.MemberPaymentEnabled() {
		return &types.GetMemberPlansResp{
			Success: false,
		}, nil
	}

	plans := paymentlogic.MemberPlans()
	items := make([]types.MemberPlanInfo, 0, len(plans))
	for _, item := range plans {
		items = append(items, paymentlogic.BuildMemberPlanInfo(item))
	}

	return &types.GetMemberPlansResp{
		Success:     true,
		CurrentTime: logicx.FormatUTC8Time(logicx.NowUTC8()),
		Plans:       items,
	}, nil
}
