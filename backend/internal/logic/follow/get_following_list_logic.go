package follow

import (
	"context"

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
		svcCtx: svcCtx,
	}
}

func (l *GetFollowingListLogic) GetFollowingList(req *types.GetFollowListReq) (resp *types.GetFollowListResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetFollowListResp{Success: false}, nil
	}

	list, total, err := l.svcCtx.FollowModel.GetFollowingList(userIdInt, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询关注列表失败: user=%d err=%v", userIdInt, err)
		return &types.GetFollowListResp{Success: false}, nil
	}

	items := make([]types.FollowUserInfo, 0, len(list))
	for _, item := range list {
		user, findErr := l.svcCtx.UserModel.FindById(item.FollowingId)
		if findErr != nil {
			l.Logger.Errorf("查询关注用户信息失败: target=%d err=%v", item.FollowingId, findErr)
			continue
		}
		if user == nil {
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
