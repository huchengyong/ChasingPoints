package auth

import (
	"context"
	"fmt"

	"chasing_points/internal/model"
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
	if req == nil || req.Provider == wechatMiniProvider {
		return &types.LoginByOauthResp{Success: false}, nil
	}

	// 查找OAuth关联
	oauth, err := l.svcCtx.OauthModel.FindByProviderAndOpenId(req.Provider, req.OpenId)
	if err != nil {
		l.Logger.Errorf("查找OAuth关联失败: %v", err)
		return &types.LoginByOauthResp{Success: false}, err
	}

	var user *model.User
	needBindPhone := false

	if oauth != nil {
		// 已有OAuth关联，查找用户
		user, err = l.svcCtx.UserModel.FindById(oauth.UserId)
		if err != nil {
			l.Logger.Errorf("查找用户失败: %v", err)
			return &types.LoginByOauthResp{Success: false}, err
		}
		if user == nil {
			l.Logger.Errorf("OAuth关联的用户不存在: userId=%d", oauth.UserId)
			return &types.LoginByOauthResp{Success: false}, fmt.Errorf("用户数据异常")
		}

		// 检查是否绑定手机
		if user.Phone == nil || *user.Phone == "" {
			needBindPhone = true
		}
	} else {
		// 没有OAuth关联，创建新用户
		nickname := req.NickName
		if nickname == "" {
			nickname = "华为用户"
		}

		user = &model.User{
			Nickname:        nickname,
			Avatar:          req.AvatarUrl,
			Status:          1,
			MemberExpiresAt: resolveWelcomeMemberExpiresAt(l.svcCtx),
		}
		if err := l.svcCtx.UserModel.Create(user); err != nil {
			l.Logger.Errorf("创建用户失败: %v", err)
			return &types.LoginByOauthResp{Success: false}, err
		}

		// 创建OAuth关联
		var unionId *string
		if req.UnionId != "" {
			unionId = &req.UnionId
		}
		oauthRecord := &model.UserOauth{
			UserId:   user.Id,
			Provider: req.Provider,
			OpenId:   req.OpenId,
			UnionId:  unionId,
		}
		if err := l.svcCtx.OauthModel.Create(oauthRecord); err != nil {
			l.Logger.Errorf("创建OAuth关联失败: %v", err)
			return &types.LoginByOauthResp{Success: false}, err
		}

		needBindPhone = true
	}
	if user.Status != 1 {
		return &types.LoginByOauthResp{Success: false}, fmt.Errorf("%s", loginUnavailableMessage)
	}

	tokenPair, err := issueAuthTokenPair(user.Id, l.svcCtx)
	if err != nil {
		l.Logger.Errorf("生成登录令牌失败: %v", err)
		return &types.LoginByOauthResp{Success: false}, err
	}

	// 脱敏手机号
	maskedPhone := ""
	if user.Phone != nil && *user.Phone != "" {
		phone := *user.Phone
		if len(phone) == 11 {
			maskedPhone = phone[:3] + "****" + phone[7:]
		}
	}

	return &types.LoginByOauthResp{
		Success:       true,
		AccessToken:   tokenPair.AccessToken,
		RefreshToken:  tokenPair.RefreshToken,
		ExpiresIn:     tokenPair.ExpiresIn,
		NeedBindPhone: needBindPhone,
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
