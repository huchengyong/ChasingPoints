package challenge

import (
	"context"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPendingChallengesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 活动约球列表：收到待回应 + 已接受/已开局 + 本人发出待回应
func NewGetPendingChallengesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPendingChallengesLogic {
	return &GetPendingChallengesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetPendingChallengesLogic) GetPendingChallenges() (resp *types.GetPendingChallengesResp, err error) {
	resp = &types.GetPendingChallengesResp{List: []types.ChallengeInfo{}}
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return resp, nil
	}
	now := time.Now()
	rows, err := l.svcCtx.ChallengeModel.ListActiveByUserWithRows(userId, now)
	if err != nil {
		l.Logger.Errorf("查询活动约球失败: userId=%d err=%v", userId, err)
		return resp, nil
	}
	resp.List = buildChallengeInfoList(rows, now)
	resp.Success = true
	return resp, nil
}
