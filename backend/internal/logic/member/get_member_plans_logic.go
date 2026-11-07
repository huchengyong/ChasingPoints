package member

import (
	"context"

	logicx "chasing_points/internal/logic"
	paymentlogic "chasing_points/internal/logic/payment"
	"chasing_points/internal/logic/staticread"
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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetMemberPlansLogic) GetMemberPlans() (resp *types.GetMemberPlansResp, err error) {
	if !l.svcCtx.Config.MemberPaymentEnabled() {
		return &types.GetMemberPlansResp{
			Success: false,
		}, nil
	}

	items, err := staticread.Load(l.ctx, l.svcCtx, "member-plans", func() ([]types.MemberPlanInfo, error) {
		plans := paymentlogic.MemberPlans()
		result := make([]types.MemberPlanInfo, 0, len(plans))
		for _, item := range plans {
			result = append(result, paymentlogic.BuildMemberPlanInfo(item))
		}
		return result, nil
	})
	if err != nil {
		l.Logger.Errorf("获取会员套餐缓存失败: %v", err)
		return &types.GetMemberPlansResp{Success: false}, nil
	}
	return &types.GetMemberPlansResp{Success: true, CurrentTime: logicx.FormatUTC8Time(logicx.NowUTC8()), Plans: items}, nil
}
