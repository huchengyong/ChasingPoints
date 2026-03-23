package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/pkg"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	loginFailPrefix     = "admin:login:fail:"
	loginFailMaxTimes   = 5
	loginFailLockExpire = 30 * time.Minute
	loginFailExpire     = 5 * time.Minute
)

type AdminLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 管理员登录
func NewAdminLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminLoginLogic {
	return &AdminLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminLoginLogic) AdminLogin(req *types.AdminLoginReq) (resp *types.AdminLoginResp, err error) {
	normalizedEmail := normalizeAdminEmail(req.Email)
	if !validateAdminEmail(normalizedEmail) || strings.TrimSpace(req.Password) == "" {
		return &types.AdminLoginResp{
			Code:    400,
			Success: false,
			Message: "请输入正确的邮箱和密码",
		}, nil
	}

	loginFailKey := loginFailPrefix + normalizedEmail

	// 检查是否被锁定
	lockKey := loginFailKey + ":lock"
	locked, _ := l.svcCtx.Redis.Exists(l.ctx, lockKey).Result()
	if locked > 0 {
		return &types.AdminLoginResp{
			Code:    429,
			Success: false,
			Message: "登录失败次数过多，请30分钟后再试",
		}, nil
	}

	// 查找管理员
	admin, err := l.svcCtx.AdminModel.FindByEmail(l.ctx, normalizedEmail)
	if err != nil {
		return &types.AdminLoginResp{
			Code:    401,
			Success: false,
			Message: "邮箱或密码错误",
		}, nil
	}

	// 检查账号状态
	if admin.Status != 1 {
		return &types.AdminLoginResp{
			Code:    403,
			Success: false,
			Message: "账号已被禁用",
		}, nil
	}

	// 验证密码
	if !utils.CheckPasswordHash(req.Password, admin.Password) {
		// 增加失败计数
		failCount, _ := l.svcCtx.Redis.Incr(l.ctx, loginFailKey).Result()
		l.svcCtx.Redis.Expire(l.ctx, loginFailKey, loginFailExpire)

		// 超过最大次数则锁定
		if failCount >= loginFailMaxTimes {
			l.svcCtx.Redis.Set(l.ctx, lockKey, "1", loginFailLockExpire)
			return &types.AdminLoginResp{
				Code:    429,
				Success: false,
				Message: "登录失败次数过多，请30分钟后再试",
			}, nil
		}

		return &types.AdminLoginResp{
			Code:    401,
			Success: false,
			Message: fmt.Sprintf("邮箱或密码错误，还剩%d次机会", loginFailMaxTimes-failCount),
		}, nil
	}

	// 生成Token (使用负数ID区分普通用户和admin)
	token, err := pkg.GenerateToken(-int64(admin.Id), l.svcCtx.Config.Auth.AccessSecret, l.svcCtx.Config.Auth.AccessExpire)
	if err != nil {
		l.Logger.Errorf("生成Token失败: %v", err)
		return &types.AdminLoginResp{
			Code:    500,
			Success: false,
			Message: "系统错误",
		}, nil
	}

	// 更新最后登录时间
	if err := l.svcCtx.AdminModel.UpdateLastLoginAt(l.ctx, admin.Id); err != nil {
		l.Logger.Errorf("更新登录时间失败: %v", err)
	}

	// 记录登录日志
	log := &model.AdminLoginLog{
		AdminId:   admin.Id,
		Ip:        utils.GetClientIP(l.ctx),
		UserAgent: utils.GetUserAgent(l.ctx),
	}
	if err := l.svcCtx.AdminLoginLogModel.Create(l.ctx, log); err != nil {
		l.Logger.Errorf("记录登录日志失败: %v", err)
	}

	// 清除登录失败计数
	l.svcCtx.Redis.Del(l.ctx, loginFailKey)

	return &types.AdminLoginResp{
		Code:    0,
		Success: true,
		Message: "登录成功",
		Token:   token,
		UserInfo: &types.AdminInfo{
			Id:       int64(admin.Id),
			Email:    admin.Email,
			Nickname: admin.Nickname,
			Avatar:   admin.Avatar,
			Role:     admin.Role,
		},
	}, nil
}
