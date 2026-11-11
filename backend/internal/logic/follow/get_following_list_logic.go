package follow

import (
	"context"

	"chasing_points/internal/config"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFollowingListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取关注列表
func NewGetFollowingListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFollowingListLogic {
	return &GetFollowingListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetFollowingListLogic) GetFollowingList(req *types.GetFollowListReq) (resp *types.GetFollowListResp, err error) {
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

	list, total, err := l.svcCtx.FollowModel.GetFollowingList(userIdInt, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询关注列表失败: user=%d err=%v", userIdInt, err)
		return &types.GetFollowListResp{Success: false, List: []types.FollowUserInfo{}}, nil
	}

	followingIDs := make([]int64, 0, len(list))
	for _, item := range list {
		followingIDs = append(followingIDs, item.FollowingId)
	}
	users, err := l.svcCtx.UserModel.FindByIds(followingIDs)
	if err != nil {
		l.Logger.Errorf("批量查询关注用户信息失败: user=%d err=%v", userIdInt, err)
		return &types.GetFollowListResp{Success: false, List: []types.FollowUserInfo{}}, nil
	}

	items := make([]types.FollowUserInfo, 0, len(list))
	for _, item := range list {
		user, ok := users[item.FollowingId]
		if !ok {
			continue
		}
		items = append(items, types.FollowUserInfo{
			UserId:      user.Id,
			Nickname:    user.Nickname,
			Avatar:      user.Avatar,
			RankName:    "",
			IsFollowing: true,
		})
	}

	return &types.GetFollowListResp{Success: true, Total: total, List: items}, nil
}
