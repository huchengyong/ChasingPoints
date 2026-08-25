package auth

import (
	"context"
	"fmt"
	"strings"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 刷新登录态
func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *RefreshTokenLogic) RefreshToken(req *types.RefreshTokenReq) (resp *types.RefreshTokenResp, err error) {
	if req == nil || strings.TrimSpace(req.RefreshToken) == "" {
		return &types.RefreshTokenResp{
			Success: false,
			Message: "refresh token 不能为空",
		}, nil
	}

	userId, err := parseRefreshTokenUserId(strings.TrimSpace(req.RefreshToken), l.svcCtx.Config.Auth.AccessSecret)
	if err != nil {
		return &types.RefreshTokenResp{
			Success: false,
			Message: "登录状态已失效",
			Reason:  "SESSION_INVALID",
		}, nil
	}

	user, err := l.svcCtx.UserModel.FindById(userId)
	if err != nil {
		l.Logger.Errorf("刷新登录态查找用户失败: %v", err)
		return &types.RefreshTokenResp{
			Success: false,
			Message: "系统错误",
		}, fmt.Errorf("系统错误")
	}
	if user == nil || user.Status != 1 {
		return &types.RefreshTokenResp{
			Success: false,
			Message: "登录状态已失效",
			Reason:  "SESSION_INVALID",
		}, nil
	}

	tokenPair, err := issueAuthTokenPair(userId, l.svcCtx)
	if err != nil {
		l.Logger.Errorf("刷新登录态生成令牌失败: %v", err)
		return &types.RefreshTokenResp{
			Success: false,
			Message: "系统错误",
		}, fmt.Errorf("系统错误")
	}

	return &types.RefreshTokenResp{
		Success:      true,
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}
