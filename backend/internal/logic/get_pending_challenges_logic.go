package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

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
		svcCtx: svcCtx,
	}
}

func (l *GetPendingChallengesLogic) GetPendingChallenges() (resp *types.GetPendingChallengesResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetPendingChallengesResp{Success: false, List: []types.ChallengeInfo{}}, nil
	}

	if err = l.svcCtx.ChallengeModel.ExpireOld(); err != nil {
		l.Logger.Errorf("过期挑战清理失败: err=%v", err)
	}

	challenges, err := l.svcCtx.ChallengeModel.GetPendingByUserId(userIdInt)
	if err != nil {
		l.Logger.Errorf("查询待处理挑战失败: userId=%d err=%v", userIdInt, err)
		return &types.GetPendingChallengesResp{Success: false, List: []types.ChallengeInfo{}}, nil
	}

	list := make([]types.ChallengeInfo, 0, len(challenges))
	for _, challenge := range challenges {
		fromUser, fromErr := l.svcCtx.UserModel.FindById(challenge.FromUserId)
		if fromErr != nil {
			l.Logger.Errorf("查询挑战发起人失败: fromUserId=%d err=%v", challenge.FromUserId, fromErr)
			continue
		}
		toUser, toErr := l.svcCtx.UserModel.FindById(challenge.ToUserId)
		if toErr != nil {
			l.Logger.Errorf("查询挑战目标用户失败: toUserId=%d err=%v", challenge.ToUserId, toErr)
			continue
		}

		info := types.ChallengeInfo{
			Id:         challenge.Id,
			FromUserId: challenge.FromUserId,
			ToUserId:   challenge.ToUserId,
			GameType:   challenge.GameType,
			Message:    challenge.Message,
			Status:     challenge.Status,
			CreatedAt:  challenge.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if fromUser != nil {
			info.FromNickname = fromUser.Nickname
			info.FromAvatar = fromUser.Avatar
		}
		if toUser != nil {
			info.ToNickname = toUser.Nickname
			info.ToAvatar = toUser.Avatar
		}

		list = append(list, info)
	}

	return &types.GetPendingChallengesResp{Success: true, List: list}, nil
}
