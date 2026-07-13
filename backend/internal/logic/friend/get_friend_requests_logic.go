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
		svcCtx: svcCtx,
	}
}

func (l *GetFriendRequestsLogic) GetFriendRequests(req *types.GetFriendRequestsReq) (resp *types.GetFriendRequestsResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetFriendRequestsResp{Success: false}, nil
	}

	requests, err := l.svcCtx.FriendModel.GetPendingRequests(userIdInt)
	if err != nil {
		l.Logger.Errorf("查询好友请求失败: %v", err)
		return &types.GetFriendRequestsResp{Success: false, List: []types.FriendRequestInfo{}}, nil
	}

	total := int64(len(requests))
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start >= len(requests) {
		return &types.GetFriendRequestsResp{Success: true, Total: total, List: []types.FriendRequestInfo{}}, nil
	}
	end := start + pageSize
	if end > len(requests) {
		end = len(requests)
	}

	list := make([]types.FriendRequestInfo, 0, end-start)
	for _, item := range requests[start:end] {
		fromUser, userErr := l.svcCtx.UserModel.FindById(item.FromUserId)
		if userErr != nil {
			l.Logger.Errorf("查询请求发起方用户信息失败: fromUserId=%d err=%v", item.FromUserId, userErr)
			continue
		}
		if fromUser == nil {
			continue
		}

		list = append(list, types.FriendRequestInfo{
			Id:         item.Id,
			FromUserId: item.FromUserId,
			Nickname:   fromUser.Nickname,
			Avatar:     fromUser.Avatar,
			Message:    item.Message,
			Status:     item.Status,
			CreatedAt:  item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.GetFriendRequestsResp{Success: true, Total: total, List: list}, nil
}
