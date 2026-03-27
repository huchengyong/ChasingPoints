package auth

import (
	"context"
	"regexp"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type BindPhoneLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 绑定手机号
func NewBindPhoneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindPhoneLogic {
	return &BindPhoneLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BindPhoneLogic) BindPhone(req *types.BindPhoneReq) (resp *types.BindPhoneResp, err error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.BindPhoneResp{
			Success: false,
			Message: "用户未登录",
		}, nil
	}

	// 验证手机号格式
	if !regexp.MustCompile(`^1[3-9]\d{9}$`).MatchString(req.Phone) {
		return &types.BindPhoneResp{
			Success: false,
			Message: "手机号格式不正确",
		}, nil
	}

	// 验证验证码
	valid, err := l.svcCtx.CodeManager.VerifyCode(l.ctx, req.Phone, req.SmsCode)
	if err != nil {
		return &types.BindPhoneResp{
			Success: false,
			Message: err.Error(),
		}, nil
	}
	if !valid {
		return &types.BindPhoneResp{
			Success: false,
			Message: "验证码错误",
		}, nil
	}

	// 检查手机号是否已被使用
	existingUser, err := l.svcCtx.UserModel.FindByPhone(req.Phone)
	if err != nil {
		l.Logger.Errorf("查询用户失败: %v", err)
		return &types.BindPhoneResp{
			Success: false,
			Message: "系统错误",
		}, nil
	}
	if existingUser != nil && existingUser.Id != userId {
		// 账号合并：将当前用户的 OAuth 记录迁移到已有手机号用户
		l.Logger.Infof("检测到账号合并场景: OAuth用户 %d -> 手机号用户 %d", userId, existingUser.Id)

		if err := l.svcCtx.OauthModel.UpdateUserId(userId, existingUser.Id); err != nil {
			l.Logger.Errorf("账号合并失败: %v", err)
			return &types.BindPhoneResp{
				Success: false,
				Message: "账号合并失败，请联系客服",
			}, nil
		}

		// 删除旧的 OAuth 用户记录
		if err := l.svcCtx.UserModel.DeleteById(userId); err != nil {
			l.Logger.Errorf("删除旧用户失败: %v", err)
			// 不影响主流程，只记录日志
		}

		l.Logger.Infof("账号合并成功: OAuth用户 %d 已迁移到手机号用户 %d，旧用户已删除", userId, existingUser.Id)

		return buildBindPhoneSuccessResp(true), nil
	}

	// 更新用户手机号
	if err := l.svcCtx.UserModel.UpdatePhone(userId, req.Phone); err != nil {
		l.Logger.Errorf("更新手机号失败: %v", err)
		return &types.BindPhoneResp{
			Success: false,
			Message: "绑定失败",
		}, nil
	}

	l.Logger.Infof("用户 %d 绑定手机号 %s 成功", userId, req.Phone)

	return buildBindPhoneSuccessResp(false), nil
}
