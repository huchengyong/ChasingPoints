package friend

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFriendBlacklistLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取黑名单列表
func NewGetFriendBlacklistLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFriendBlacklistLogic {
	return &GetFriendBlacklistLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetFriendBlacklistLogic) GetFriendBlacklist(req *types.GetFriendBlacklistReq) (resp *types.GetFriendBlacklistResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetFriendBlacklistResp{Success: false, List: []types.FriendBlacklistItem{}}, nil
	}
	if req == nil {
		req = &types.GetFriendBlacklistReq{}
	}

	rows, total, err := l.svcCtx.FriendModel.GetBlacklistWithProfiles(userIdInt, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询黑名单列表失败: user=%d err=%v", userIdInt, err)
		return &types.GetFriendBlacklistResp{Success: false, List: []types.FriendBlacklistItem{}}, nil
	}

	list := make([]types.FriendBlacklistItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, types.FriendBlacklistItem{
			UserId:    row.UserId,
			Nickname:  row.Nickname,
			Avatar:    row.Avatar,
			CreatedAt: row.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.GetFriendBlacklistResp{Success: true, Total: total, List: list}, nil
}
