package auth

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type WechatMiniLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 微信小程序一键登录
func NewWechatMiniLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatMiniLoginLogic {
	return &WechatMiniLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *WechatMiniLoginLogic) WechatMiniLogin(req *types.WechatMiniLoginReq) (resp *types.WechatMiniLoginResp, err error) {
	if req == nil || strings.TrimSpace(req.Code) == "" {
		return &types.WechatMiniLoginResp{Success: false, Message: "微信登录凭证无效"}, nil
	}
	if l.svcCtx == nil || l.svcCtx.WechatMiniClient == nil {
		return &types.WechatMiniLoginResp{Success: false, Message: "微信小程序登录暂不可用"}, nil
	}

	identity, err := l.svcCtx.WechatMiniClient.ExchangeLoginCode(l.ctx, req.Code)
	if err != nil {
		return &types.WechatMiniLoginResp{Success: false, Message: err.Error()}, nil
	}
	if identity == nil || strings.TrimSpace(identity.OpenID) == "" {
		return &types.WechatMiniLoginResp{Success: false, Message: "微信登录凭证无效"}, nil
	}

	oauth, err := l.svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, identity.OpenID)
	if err != nil {
		l.Logger.Errorf("查询微信小程序关联失败: %v", err)
		return nil, err
	}

	var user *model.User
	if oauth == nil {
		user, err = l.createWechatMiniUser(identity.OpenID, identity.UnionID)
		if err != nil && isDuplicateOAuthError(err) {
			oauth, err = l.svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, identity.OpenID)
		}
		if err != nil {
			l.Logger.Errorf("创建微信小程序用户失败: %v", err)
			return nil, err
		}
	}

	if user == nil {
		if oauth == nil {
			return nil, fmt.Errorf("微信小程序关联不存在")
		}
		user, err = l.svcCtx.UserModel.FindById(oauth.UserId)
		if err != nil {
			l.Logger.Errorf("查询微信小程序用户失败: %v", err)
			return nil, err
		}
		if user == nil {
			return nil, fmt.Errorf("用户数据异常")
		}
	}
	if user.Status != 1 {
		return &types.WechatMiniLoginResp{Success: false, Message: loginUnavailableMessage}, nil
	}

	tokenPair, err := issueAuthTokenPair(user.Id, l.svcCtx)
	if err != nil {
		l.Logger.Errorf("生成微信小程序登录令牌失败: %v", err)
		return nil, err
	}

	return &types.WechatMiniLoginResp{
		Success:       true,
		AccessToken:   tokenPair.AccessToken,
		RefreshToken:  tokenPair.RefreshToken,
		ExpiresIn:     tokenPair.ExpiresIn,
		NeedBindPhone: user.Phone == nil || *user.Phone == "",
		UserInfo:      buildAuthUserInfo(user),
	}, nil
}

const wechatMiniProvider = "weixin_mini_program"

var (
	wechatMiniBilliardsNicknames = []string{"旋风清台", "金杆球手", "精准走位"}
	wechatMiniCasualNicknames    = []string{"青柠旅人", "晴日玩家", "云端漫游"}
	wechatMiniMixedNicknames     = []string{"夜航开杆", "流光走位", "星尘清台"}
)

func (l *WechatMiniLoginLogic) createWechatMiniUser(openID, unionID string) (*model.User, error) {
	user := &model.User{
		Nickname:        randomWechatMiniNickname(),
		Status:          1,
		MemberExpiresAt: resolveWelcomeMemberExpiresAt(l.svcCtx),
	}

	err := l.svcCtx.UserModel.Transaction(func(tx *gorm.DB) error {
		if err := l.svcCtx.UserModel.CreateWithTx(tx, user); err != nil {
			return err
		}

		var unionIDPtr *string
		if strings.TrimSpace(unionID) != "" {
			unionIDPtr = &unionID
		}
		return l.svcCtx.OauthModel.CreateWithTx(tx, &model.UserOauth{
			UserId:   user.Id,
			Provider: wechatMiniProvider,
			OpenId:   openID,
			UnionId:  unionIDPtr,
		})
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func randomWechatMiniNickname() string {
	return randomWechatMiniNicknameWith(rand.IntN)
}

func randomWechatMiniNicknameWith(randomInt func(int) int) string {
	styles := [][]string{
		wechatMiniBilliardsNicknames,
		wechatMiniCasualNicknames,
		wechatMiniMixedNicknames,
	}
	words := styles[randomInt(len(styles))]
	return fmt.Sprintf("%s·%04d", words[randomInt(len(words))], randomInt(10000))
}

func isDuplicateOAuthError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate") || strings.Contains(lower, "unique constraint")
}
