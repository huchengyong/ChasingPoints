package auth

import (
	"context"
	"fmt"
	"regexp"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 手机号登录/注册
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	// 验证手机号格式
	if !regexp.MustCompile(`^1[3-9]\d{9}$`).MatchString(req.Phone) {
		return &types.LoginResp{
			Success: false,
		}, fmt.Errorf("手机号格式不正确")
	}

	// 验证验证码
	valid, err := l.svcCtx.CodeManager.VerifyCode(l.ctx, req.Phone, req.SmsCode)
	if err != nil {
		return &types.LoginResp{
			Success: false,
		}, err
	}
	if !valid {
		return &types.LoginResp{
			Success: false,
		}, fmt.Errorf("验证码错误")
	}

	// 查找用户
	user, err := l.svcCtx.UserModel.FindByPhone(req.Phone)
	if err != nil {
		l.Logger.Errorf("查找用户失败: %v", err)
		return &types.LoginResp{
			Success: false,
		}, fmt.Errorf("系统错误")
	}

	// 如果用户不存在，自动注册
	if user == nil {
		user = &model.User{
			Phone:    &req.Phone,
			Nickname: fmt.Sprintf("用户%s", req.Phone[7:]),
			Avatar:   "https://cdn.dianzaozao.com/avatars/f512f44051984823941dd0d214ed84f6.jpg",
			Status:   1,
		}
		if err := l.svcCtx.UserModel.Create(user); err != nil {
			l.Logger.Errorf("创建用户失败: %v", err)
			return &types.LoginResp{
				Success: false,
			}, fmt.Errorf("注册失败")
		}
	}

	// 生成Token
	accessToken, err := pkg.GenerateToken(user.Id, l.svcCtx.Config.Auth.AccessSecret, l.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		l.Logger.Errorf("生成Token失败: %v", err)
		return &types.LoginResp{
			Success: false,
		}, fmt.Errorf("系统错误")
	}

	// 生成RefreshToken (7天)
	refreshToken, err := pkg.GenerateToken(user.Id, l.svcCtx.Config.Auth.AccessSecret, 7*24*3600)
	if err != nil {
		l.Logger.Errorf("生成RefreshToken失败: %v", err)
		return &types.LoginResp{
			Success: false,
		}, fmt.Errorf("系统错误")
	}

	// 脱敏手机号
	maskedPhone := ""
	if user.Phone != nil {
		phone := *user.Phone
		if len(phone) == 11 {
			maskedPhone = phone[:3] + "****" + phone[7:]
		}
	}

	return &types.LoginResp{
		Success:      true,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    l.svcCtx.Config.Auth.AccessExpire,
		UserInfo: &types.UserInfo{
			Id:        user.Id,
			Phone:     maskedPhone,
			Nickname:  user.Nickname,
			Avatar:    user.Avatar,
			Status:    user.Status,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	}, nil
}
