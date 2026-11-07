package friend

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFriendRequestsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取好友请求列表
func NewGetFriendRequestsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFriendRequestsLogic {
	return &GetFriendRequestsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetFriendRequestsLogic) GetFriendRequests(req *types.GetFriendRequestsReq) (resp *types.GetFriendRequestsResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetFriendRequestsResp{Success: false}, nil
	}

	requests, total, err := l.svcCtx.FriendModel.GetPendingRequestPageWithProfiles(userIdInt, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询好友请求失败: %v", err)
		return &types.GetFriendRequestsResp{Success: false, List: []types.FriendRequestInfo{}}, nil
	}

	list := make([]types.FriendRequestInfo, 0, len(requests))
	for _, item := range requests {
		list = append(list, types.FriendRequestInfo{
			Id:         item.Id,
			FromUserId: item.FromUserId,
			Nickname:   item.Nickname,
			Avatar:     item.Avatar,
			Message:    item.Message,
			Status:     item.Status,
			CreatedAt:  item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.GetFriendRequestsResp{Success: true, Total: total, List: list}, nil
}
