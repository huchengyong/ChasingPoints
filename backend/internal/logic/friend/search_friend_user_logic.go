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
		svcCtx: svcCtx,
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

	users, err := l.svcCtx.FriendModel.SearchUsers(userIdInt, keyword, 20)
	if err != nil {
		l.Logger.Errorf("搜索用户失败: %v", err)
		return &types.SearchUserResp{Success: false, List: []types.SearchUserItem{}}, nil
	}

	rankNameMap := map[int]string{}
	configs, err := l.svcCtx.RankingModel.GetAllRankConfigs()
	if err == nil {
		for _, item := range configs {
			rankNameMap[item.Level] = item.Name
		}
	} else {
		l.Logger.Errorf("查询段位配置失败: %v", err)
	}

	list := make([]types.SearchUserItem, 0, len(users))
	for _, item := range users {
		isFriend, friendErr := l.svcCtx.FriendModel.AreFriends(userIdInt, item.Id)
		if friendErr != nil {
			l.Logger.Errorf("检查好友关系失败: userId=%d targetId=%d err=%v", userIdInt, item.Id, friendErr)
			continue
		}
		if isFriend {
			continue
		}

		rankName := rankNameMap[1]
		ranking, rankErr := l.svcCtx.RankingModel.FindByUserId(item.Id)
		if rankErr != nil {
			l.Logger.Errorf("查询用户段位失败: userId=%d err=%v", item.Id, rankErr)
		} else if ranking != nil {
			if rankNameMap[ranking.RankLevel] != "" {
				rankName = rankNameMap[ranking.RankLevel]
			}
		}

		list = append(list, types.SearchUserItem{
			UserId:   item.Id,
			Nickname: item.Nickname,
			Avatar:   item.Avatar,
			RankName: rankName,
			IsFriend: false,
		})
	}

	return &types.SearchUserResp{Success: true, List: list}, nil
}
