package member

import (
	"context"

	logicx "chasing_points/internal/logic"
	paymentlogic "chasing_points/internal/logic/payment"
	"chasing_points/internal/requestctx"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMemberStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取会员状态
func NewGetMemberStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMemberStatusLogic {
	return &GetMemberStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetMemberStatusLogic) GetMemberStatus() (resp *types.GetMemberStatusResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMemberStatusResp{Success: false}, nil
	}

	user := requestctx.ActiveUser(l.ctx)
	if user == nil || user.Id != userID {
		user, err = l.svcCtx.UserModel.FindById(userID)
		if err != nil {
			l.Logger.Errorf("查询用户失败: userId=%d err=%v", userID, err)
			return &types.GetMemberStatusResp{Success: false}, nil
		}
	}
	if user == nil {
		return &types.GetMemberStatusResp{Success: false}, nil
	}

	resp = paymentlogic.BuildMemberStatusResp(user)
	if resp == nil {
		return &types.GetMemberStatusResp{Success: false}, nil
	}

	growthService := logicx.NewMemberGrowthService(l.svcCtx, logicx.NowUTC8)
	snapshot, growthErr := growthService.GetSnapshot(user)
	if growthErr != nil {
		l.Logger.Errorf("读取会员成长快照失败: userId=%d err=%v", userID, growthErr)
		return resp, nil
	}

	resp.GrowthLevel = snapshot.GrowthLevel
	resp.GrowthPoints = snapshot.GrowthPoints
	resp.TodayGrowthCount = snapshot.TodayGrowthCount
	resp.GrowthDailyCap = snapshot.DailyCap
	resp.GrowthFrozen = snapshot.Frozen
	resp.NextGrowthLevel = snapshot.NextLevel
	resp.NextGrowthLevelPoints = snapshot.NextLevelPoints
	resp.RemainingGrowthPoints = snapshot.RemainingPoints
	resp.CurrentTime = logicx.FormatUTC8Time(logicx.NowUTC8())

	return resp, nil
}
