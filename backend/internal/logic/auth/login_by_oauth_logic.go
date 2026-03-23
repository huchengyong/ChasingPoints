package auth

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginByOauthLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// OAuth登录
func NewLoginByOauthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginByOauthLogic {
	return &LoginByOauthLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginByOauthLogic) LoginByOauth(req *types.LoginByOauthReq) (resp *types.LoginByOauthResp, err error) {
	// todo: add your logic here and delete this line

	return
}
