package challenge

import (
	"context"

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

// 获取待处理挑战
func NewGetPendingChallengesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPendingChallengesLogic {
	return &GetPendingChallengesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetPendingChallengesLogic) GetPendingChallenges() (resp *types.GetPendingChallengesResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetPendingChallengesResp{Success: false, List: []types.ChallengeInfo{}}, nil
	}

	challenges, err := l.svcCtx.ChallengeModel.GetPendingByUserIdWithProfiles(userIdInt)
	if err != nil {
		l.Logger.Errorf("查询待处理挑战失败: userId=%d err=%v", userIdInt, err)
		return &types.GetPendingChallengesResp{Success: false, List: []types.ChallengeInfo{}}, nil
	}

	list := make([]types.ChallengeInfo, 0, len(challenges))
	for _, challenge := range challenges {
		info := types.ChallengeInfo{
			Id:           challenge.Id,
			FromUserId:   challenge.FromUserId,
			ToUserId:     challenge.ToUserId,
			FromNickname: challenge.FromNickname,
			FromAvatar:   challenge.FromAvatar,
			ToNickname:   challenge.ToNickname,
			ToAvatar:     challenge.ToAvatar,
			GameType:     challenge.GameType,
			Message:      challenge.Message,
			Status:       challenge.Status,
			CreatedAt:    challenge.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if challenge.MatchId != nil {
			info.MatchId = *challenge.MatchId
		}
		list = append(list, info)
	}

	return &types.GetPendingChallengesResp{Success: true, List: list}, nil
}
