package logic

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
		svcCtx: svcCtx,
	}
}

func (l *GetFriendListLogic) GetFriendList(req *types.GetFriendListReq) (resp *types.GetFriendListResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetFriendListResp{Success: false}, nil
	}

	friends, total, err := l.svcCtx.FriendModel.GetFriendList(userIdInt, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("查询好友列表失败: %v", err)
		return &types.GetFriendListResp{Success: false, List: []types.FriendInfo{}}, nil
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

	list := make([]types.FriendInfo, 0, len(friends))
	for _, item := range friends {
		friendUser, userErr := l.svcCtx.UserModel.FindById(item.FriendId)
		if userErr != nil {
			l.Logger.Errorf("查询好友用户信息失败: friendId=%d err=%v", item.FriendId, userErr)
			continue
		}
		if friendUser == nil {
			continue
		}

		rankLevel := 1
		rankName := rankNameMap[1]
		ranking, rankErr := l.svcCtx.RankingModel.FindByUserId(item.FriendId)
		if rankErr != nil {
			l.Logger.Errorf("查询好友段位失败: friendId=%d err=%v", item.FriendId, rankErr)
		} else if ranking != nil {
			rankLevel = ranking.RankLevel
			if rankNameMap[rankLevel] != "" {
				rankName = rankNameMap[rankLevel]
			}
		}

		list = append(list, types.FriendInfo{
			Id:        item.Id,
			UserId:    friendUser.Id,
			Nickname:  friendUser.Nickname,
			Avatar:    friendUser.Avatar,
			RankLevel: rankLevel,
			RankName:  rankName,
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.GetFriendListResp{Success: true, Total: total, List: list}, nil
}
