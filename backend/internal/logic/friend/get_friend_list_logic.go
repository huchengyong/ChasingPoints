package friend

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFriendListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取好友列表
func NewGetFriendListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFriendListLogic {
	return &GetFriendListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetFriendListLogic) GetFriendList(req *types.GetFriendListReq) (resp *types.GetFriendListResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetFriendListResp{Success: false}, nil
	}

	friends, total, err := l.svcCtx.FriendModel.GetFriendListWithProfiles(userIdInt, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询好友列表失败: %v", err)
		return &types.GetFriendListResp{Success: false, List: []types.FriendInfo{}}, nil
	}

	list := make([]types.FriendInfo, 0, len(friends))
	for _, item := range friends {
		list = append(list, types.FriendInfo{
			Id:        item.Id,
			UserId:    item.UserId,
			Nickname:  item.Nickname,
			Avatar:    item.Avatar,
			RankLevel: item.RankLevel,
			RankName:  item.RankName,
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.GetFriendListResp{Success: true, Total: total, List: list}, nil
}
