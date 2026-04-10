package user

import (
	"context"
	"errors"
	"unicode/utf8"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateNicknameLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新昵称
func NewUpdateNicknameLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateNicknameLogic {
	return &UpdateNicknameLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateNicknameLogic) UpdateNickname(req *types.UpdateNicknameReq) (resp *types.UpdateNicknameResp, err error) {
	// 从上下文获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.UpdateNicknameResp{
			Success: false,
			Message: "用户未登录",
		}, nil
	}

	// 验证昵称长度 (2-12字符)，避免移动端个人中心头部被长昵称撑开。
	nicknameLen := utf8.RuneCountInString(req.Nickname)
	if nicknameLen < 2 || nicknameLen > 12 {
		return &types.UpdateNicknameResp{
			Success: false,
			Message: "昵称长度需要在2-12个字符之间",
		}, nil
	}

	// 查询用户是否存在
	user, err := l.svcCtx.UserModel.FindById(userId)
	if err != nil {
		l.Logger.Errorf("查询用户失败: %v", err)
		return &types.UpdateNicknameResp{
			Success: false,
			Message: "系统错误",
		}, nil
	}

	if user == nil {
		return &types.UpdateNicknameResp{
			Success: false,
			Message: "用户不存在",
		}, errors.New("用户不存在")
	}

	// 更新昵称
	user.Nickname = req.Nickname
	if err := l.svcCtx.UserModel.Update(user); err != nil {
		l.Logger.Errorf("更新昵称失败: %v", err)
		return &types.UpdateNicknameResp{
			Success: false,
			Message: "更新失败",
		}, nil
	}

	l.Logger.Infof("用户 %d 更新昵称为 %s", userId, req.Nickname)

	return &types.UpdateNicknameResp{
		Success: true,
		Message: "更新成功",
		UserInfo: buildUserInfoPayload(user),
	}, nil
}
