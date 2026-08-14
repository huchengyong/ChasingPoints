package friend

import (
	"context"
	"strings"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchFriendUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 搜索用户
func NewSearchFriendUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchFriendUserLogic {
	return &SearchFriendUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *SearchFriendUserLogic) SearchFriendUser(req *types.SearchUserReq) (resp *types.SearchUserResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.SearchUserResp{Success: false}, nil
	}

	keyword := strings.TrimSpace(req.Keyword)
	if keyword == "" {
		return &types.SearchUserResp{Success: true, List: []types.SearchUserItem{}}, nil
	}

	users, err := l.svcCtx.FriendModel.SearchUserProfiles(userIdInt, keyword, 20)
	if err != nil {
		l.Logger.Errorf("搜索用户失败: %v", err)
		return &types.SearchUserResp{Success: false, List: []types.SearchUserItem{}}, nil
	}

	list := make([]types.SearchUserItem, 0, len(users))
	for _, item := range users {
		if item.IsFriend {
			continue
		}
		list = append(list, types.SearchUserItem{
			UserId:   item.UserId,
			Nickname: item.Nickname,
			Avatar:   item.Avatar,
			RankName: item.RankName,
			IsFriend: false,
		})
	}

	return &types.SearchUserResp{Success: true, List: list}, nil
}
