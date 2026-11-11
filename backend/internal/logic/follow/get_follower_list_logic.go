package follow

import (
	"context"

	"chasing_points/internal/config"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetFollowerListLogic) GetFollowerList(req *types.GetFollowListReq) (resp *types.GetFollowListResp, err error) {
	if !l.svcCtx.Config.SocialEnabled() {
		return &types.GetFollowListResp{Success: false, Message: config.DisabledFeatureMessage("social"), List: []types.FollowUserInfo{}}, nil
	}

	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetFollowListResp{Success: false, List: []types.FollowUserInfo{}}, nil
	}
	if req == nil {
		req = &types.GetFollowListReq{}
	}

	list, total, err := l.svcCtx.FollowModel.GetFollowerList(userIdInt, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询粉丝列表失败: user=%d err=%v", userIdInt, err)
		return &types.GetFollowListResp{Success: false, List: []types.FollowUserInfo{}}, nil
	}

	followerIDs := make([]int64, 0, len(list))
	for _, item := range list {
		followerIDs = append(followerIDs, item.FollowerId)
	}
	users, err := l.svcCtx.UserModel.FindByIds(followerIDs)
	if err != nil {
		l.Logger.Errorf("批量查询粉丝用户信息失败: user=%d err=%v", userIdInt, err)
		return &types.GetFollowListResp{Success: false, List: []types.FollowUserInfo{}}, nil
	}
	following, err := l.svcCtx.FollowModel.IsFollowingBatch(userIdInt, followerIDs)
	if err != nil {
		l.Logger.Errorf("批量查询互相关注状态失败: user=%d err=%v", userIdInt, err)
		following = map[int64]bool{}
	}

	items := make([]types.FollowUserInfo, 0, len(list))
	for _, item := range list {
		user, ok := users[item.FollowerId]
		if !ok {
			continue
		}
		items = append(items, types.FollowUserInfo{
			UserId:      user.Id,
			Nickname:    user.Nickname,
			Avatar:      user.Avatar,
			RankName:    "",
			IsFollowing: following[item.FollowerId],
		})
	}

	return &types.GetFollowListResp{Success: true, Total: total, List: items}, nil
}
