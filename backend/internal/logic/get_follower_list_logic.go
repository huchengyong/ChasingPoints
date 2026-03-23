package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFollowerListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取粉丝列表
func NewGetFollowerListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFollowerListLogic {
	return &GetFollowerListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFollowerListLogic) GetFollowerList(req *types.GetFollowListReq) (resp *types.GetFollowListResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetFollowListResp{Success: false}, nil
	}

	list, total, err := l.svcCtx.FollowModel.GetFollowerList(userIdInt, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询粉丝列表失败: user=%d err=%v", userIdInt, err)
		return &types.GetFollowListResp{Success: false}, nil
	}

	items := make([]types.FollowUserInfo, 0, len(list))
	for _, item := range list {
		user, findErr := l.svcCtx.UserModel.FindById(item.FollowerId)
		if findErr != nil {
			l.Logger.Errorf("查询粉丝用户信息失败: follower=%d err=%v", item.FollowerId, findErr)
			continue
		}
		if user == nil {
			continue
		}

		isFollowing, followErr := l.svcCtx.FollowModel.IsFollowing(userIdInt, item.FollowerId)
		if followErr != nil {
			l.Logger.Errorf("查询互相关注状态失败: user=%d follower=%d err=%v", userIdInt, item.FollowerId, followErr)
			isFollowing = false
		}

		items = append(items, types.FollowUserInfo{
			UserId:      user.Id,
			Nickname:    user.Nickname,
			Avatar:      user.Avatar,
			RankName:    "",
			IsFollowing: isFollowing,
		})
	}

	return &types.GetFollowListResp{Success: true, Total: total, List: items}, nil
}
