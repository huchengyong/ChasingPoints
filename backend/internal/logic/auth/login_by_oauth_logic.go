package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"chasing_points/internal/model"
	oauthverify "chasing_points/internal/pkg/oauth"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

var oauthIdentityCreateMu sync.Mutex

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
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *LoginByOauthLogic) LoginByOauth(req *types.LoginByOauthReq) (resp *types.LoginByOauthResp, err error) {
	if req == nil {
		return &types.LoginByOauthResp{Success: false}, nil
	}
	requestedProvider := strings.ToLower(strings.TrimSpace(req.Provider))
	if requestedProvider == "" || requestedProvider == wechatMiniProvider {
		return &types.LoginByOauthResp{Success: false}, nil
	}
	if l.svcCtx.OAuthVerifier == nil {
		return &types.LoginByOauthResp{Success: false}, fmt.Errorf("OAuth验证服务未配置")
	}

	identity, err := l.svcCtx.OAuthVerifier.Verify(l.ctx, oauthverify.VerifyRequest{
		Provider:       requestedProvider,
		Credential:     req.Credential,
		CredentialType: req.CredentialType,
		Platform:       req.Platform,
	})
	if err != nil {
		category := oauthverify.CategoryOf(err)
		l.Logger.Errorf("OAuth服务端验证失败: provider=%s category=%s err=%v", requestedProvider, category, err)
		if category == oauthverify.ErrorUnsupportedProvider {
			return &types.LoginByOauthResp{Success: false}, fmt.Errorf("不支持的登录方式")
		}
		return &types.LoginByOauthResp{Success: false}, err
	}
	if identity == nil || identity.Provider == "" || identity.Subject == "" || identity.Provider == wechatMiniProvider {
		return &types.LoginByOauthResp{Success: false}, fmt.Errorf("第三方身份验证响应无效")
	}

	// 查找OAuth关联
	oauth, err := l.svcCtx.OauthModel.FindByProviderAndOpenId(identity.Provider, identity.Subject)
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

	} else {
		nickname := strings.TrimSpace(req.NickName)
		if nickname == "" {
			nickname = "华为用户"
		}
		avatar := strings.TrimSpace(req.AvatarUrl)
		oauthIdentityCreateMu.Lock()
		err = func() error {
			defer oauthIdentityCreateMu.Unlock()
			return l.svcCtx.UserModel.Transaction(func(tx *gorm.DB) error {
				existingOauth, err := l.svcCtx.OauthModel.FindByProviderAndOpenIdWithTx(tx, identity.Provider, identity.Subject)
				if err != nil {
					return err
				}
				if existingOauth != nil {
					oauth = existingOauth
					return nil
				}

				user = &model.User{
					Nickname:        nickname,
					Avatar:          avatar,
					Status:          1,
					MemberExpiresAt: resolveWelcomeMemberExpiresAt(l.svcCtx),
				}
				if err := l.svcCtx.UserModel.CreateWithTx(tx, user); err != nil {
					return err
				}

				var unionId *string
				if identity.UnionID != "" {
					unionIDValue := identity.UnionID
					unionId = &unionIDValue
				}
				return l.svcCtx.OauthModel.CreateWithTx(tx, &model.UserOauth{
					UserId:   user.Id,
					Provider: identity.Provider,
					OpenId:   identity.Subject,
					UnionId:  unionId,
				})
			})
		}()
		if err != nil {
			oauth, findErr := l.svcCtx.OauthModel.FindByProviderAndOpenId(identity.Provider, identity.Subject)
			if findErr != nil || oauth == nil {
				l.Logger.Errorf("创建OAuth用户关联失败: provider=%s err=%v", identity.Provider, err)
				return &types.LoginByOauthResp{Success: false}, err
			}
		}
		if oauth != nil {
			user, err = l.svcCtx.UserModel.FindById(oauth.UserId)
			if err != nil {
				l.Logger.Errorf("查找并发创建OAuth用户失败: %v", err)
				return &types.LoginByOauthResp{Success: false}, err
			}
		}
	}
	if user == nil {
		return &types.LoginByOauthResp{Success: false}, fmt.Errorf("用户数据异常")
	}
	if user.Phone == nil || *user.Phone == "" {
		needBindPhone = true
	}
	if user.Status != 1 {
		return &types.LoginByOauthResp{Success: false}, fmt.Errorf("%s", loginUnavailableMessage)
	}
	if err := ensureUserRankingProfiles(l.svcCtx, user.Id); err != nil {
		l.Logger.Errorf("初始化用户段位失败: %v", err)
		return &types.LoginByOauthResp{Success: false}, err
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
