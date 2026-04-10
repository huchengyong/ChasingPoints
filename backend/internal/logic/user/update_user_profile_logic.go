package user

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新用户资料
func NewUpdateUserProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserProfileLogic {
	return &UpdateUserProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserProfileLogic) UpdateUserProfile(req *types.UpdateUserProfileReq) (resp *types.UpdateUserProfileResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.UpdateUserProfileResp{
			Success: false,
			Message: "用户未登录",
		}, nil
	}

	user, err := l.svcCtx.UserModel.FindById(userID)
	if err != nil {
		l.Logger.Errorf("查询用户失败: %v", err)
		return &types.UpdateUserProfileResp{
			Success: false,
			Message: "系统错误",
		}, nil
	}

	if user == nil {
		return &types.UpdateUserProfileResp{
			Success: false,
			Message: "用户不存在",
		}, errors.New("用户不存在")
	}

	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		nickname = user.Nickname
	}

	nicknameLen := utf8.RuneCountInString(nickname)
	if nicknameLen < 2 || nicknameLen > 12 {
		return &types.UpdateUserProfileResp{
			Success: false,
			Message: "昵称长度需要在2-12个字符之间",
		}, nil
	}

	avatar := strings.TrimSpace(req.Avatar)
	if avatar == "" {
		avatar = user.Avatar
	}

	user.Nickname = nickname
	user.Avatar = avatar
	if err := l.svcCtx.UserModel.Update(user); err != nil {
		l.Logger.Errorf("更新用户资料失败: %v", err)
		return &types.UpdateUserProfileResp{
			Success: false,
			Message: "更新失败",
		}, nil
	}

	return &types.UpdateUserProfileResp{
		Success:  true,
		Message:  "更新成功",
		UserInfo: buildUserInfoPayload(user),
	}, nil
}
