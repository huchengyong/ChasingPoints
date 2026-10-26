package gear

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const reasonNotFriend = "not_friend"

type GetGearUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取好友的装备
func NewGetGearUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetGearUserLogic {
	return &GetGearUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetGearUserLogic) GetGearUser(req *types.GetGearUserReq) (resp *types.GetGearUserResp, err error) {
	failed := func(reason, message string) *types.GetGearUserResp {
		return &types.GetGearUserResp{
			Success: false, Message: message, Reason: reason,
			Items: []types.GearItemInfo{}, Defaults: []types.GearDefaultInfo{},
		}
	}
	viewerId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return failed("", "用户未登录"), nil
	}
	if req == nil || req.UserId <= 0 {
		return failed("", "用户不存在"), nil
	}

	viewerIsOwner := req.UserId == viewerId
	if !viewerIsOwner {
		allowed, err := l.canViewFriendGear(viewerId, req.UserId)
		if err != nil {
			l.Logger.Errorf("校验好友关系失败: viewerId=%d targetId=%d err=%v", viewerId, req.UserId, err)
			return failed("", "读取装备失败"), nil
		}
		if !allowed {
			return failed(reasonNotFriend, "仅好友可以查看装备"), nil
		}
	}

	// 好友只能看到在用装备，退役装备只对本人可见。
	items, err := l.svcCtx.GearModel.ListUserItems(req.UserId, []int{model.GearItemStatusActive})
	if err != nil {
		l.Logger.Errorf("读取好友装备失败: targetId=%d err=%v", req.UserId, err)
		return failed("", "读取装备失败"), nil
	}
	catalog, err := loadGearCatalog(l.svcCtx.GearModel, items)
	if err != nil {
		l.Logger.Errorf("读取装备目录信息失败: targetId=%d err=%v", req.UserId, err)
		return failed("", "读取装备失败"), nil
	}
	defaults, err := l.svcCtx.GearModel.ListDefaults(req.UserId)
	if err != nil {
		l.Logger.Errorf("读取好友主力设置失败: targetId=%d err=%v", req.UserId, err)
		return failed("", "读取装备失败"), nil
	}

	return &types.GetGearUserResp{
		Success:  true,
		Message:  "ok",
		Items:    catalog.itemInfos(items, viewerIsOwner),
		Defaults: defaultInfos(defaults),
	}, nil
}

// canViewFriendGear 与荣誉墙一致：必须是好友，且双方没有拉黑关系。
func (l *GetGearUserLogic) canViewFriendGear(viewerId, targetId int64) (bool, error) {
	if l.svcCtx.FriendModel == nil {
		return false, nil
	}
	blocked, err := l.svcCtx.FriendModel.HasBlacklistRelation(viewerId, targetId)
	if err != nil || blocked {
		return false, err
	}
	return l.svcCtx.FriendModel.AreFriends(viewerId, targetId)
}
