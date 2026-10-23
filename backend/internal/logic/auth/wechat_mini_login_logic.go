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
		svcCtx: svcCtx.WithContext(ctx),
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

	user, err := l.findOAuthUser(identity.OpenID)
	if err != nil {
		return nil, err
	}
	if user != nil && user.Status != 1 {
		return &types.WechatMiniLoginResp{Success: false, Message: loginUnavailableMessage}, nil
	}

	// 已绑定手机号的老用户不再向微信换取手机号，避免多余的授权调用。
	phone := ""
	if (user == nil || user.Phone == nil || *user.Phone == "") && strings.TrimSpace(req.PhoneCode) != "" {
		phone, err = l.svcCtx.WechatMiniClient.GetPhoneNumber(l.ctx, req.PhoneCode)
		if err != nil {
			return &types.WechatMiniLoginResp{Success: false, Message: err.Error()}, nil
		}
		if !mainlandPhonePattern.MatchString(phone) {
			return &types.WechatMiniLoginResp{Success: false, Message: "手机号格式不正确"}, nil
		}
	}

	if user == nil {
		user, err = l.createWechatMiniUser(identity.OpenID, identity.UnionID, phone)
		if err != nil && isDuplicateOAuthError(err) {
			user, err = l.findOAuthUser(identity.OpenID)
			if err == nil && user == nil {
				err = fmt.Errorf("微信小程序关联不存在")
			}
		}
		if err != nil {
			if errors.Is(err, errMergeTargetInactive) {
				return &types.WechatMiniLoginResp{Success: false, Message: loginUnavailableMessage}, nil
			}
			l.Logger.Errorf("创建微信小程序用户失败: %v", err)
			return nil, err
		}
	}
	if user.Status != 1 {
		return &types.WechatMiniLoginResp{Success: false, Message: loginUnavailableMessage}, nil
	}
	if phone != "" && (user.Phone == nil || *user.Phone == "") {
		user, _, err = bindWechatMiniPhone(l.svcCtx, user.Id, phone)
		if err != nil {
			if errors.Is(err, errMergeTargetInactive) {
				return &types.WechatMiniLoginResp{Success: false, Message: loginUnavailableMessage}, nil
			}
			l.Logger.Errorf("微信小程序登录绑定手机号失败: %v", err)
			return nil, err
		}
	}
	if err := ensureUserRankingProfiles(l.svcCtx, user.Id); err != nil {
		l.Logger.Errorf("初始化用户段位失败: %v", err)
		return nil, err
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

// findOAuthUser 返回 openid 已关联的用户，没有关联时返回 nil。
func (l *WechatMiniLoginLogic) findOAuthUser(openID string) (*model.User, error) {
	oauth, err := l.svcCtx.OauthModel.FindByProviderAndOpenId(wechatMiniProvider, openID)
	if err != nil {
		l.Logger.Errorf("查询微信小程序关联失败: %v", err)
		return nil, err
	}
	if oauth == nil {
		return nil, nil
	}
	user, err := l.svcCtx.UserModel.FindById(oauth.UserId)
	if err != nil {
		l.Logger.Errorf("查询微信小程序用户失败: %v", err)
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("用户数据异常")
	}
	return user, nil
}

// createWechatMiniUser 为新 openid 建立用户关联。phone 非空时，手机号已有账号则直接关联到该账号，否则带手机号建号。
func (l *WechatMiniLoginLogic) createWechatMiniUser(openID, unionID, phone string) (*model.User, error) {
	var user *model.User
	err := l.svcCtx.UserModel.Transaction(func(tx *gorm.DB) error {
		if phone != "" {
			existingUser, err := l.svcCtx.UserModel.FindByPhoneWithTx(tx, phone)
			if err != nil {
				return err
			}
			if existingUser != nil {
				if existingUser.Status != 1 {
					return errMergeTargetInactive
				}
				user = existingUser
			}
		}
		if user == nil {
			user = &model.User{
				Nickname:        randomWechatMiniNickname(),
				Status:          1,
				MemberExpiresAt: resolveWelcomeMemberExpiresAt(l.svcCtx),
			}
			if phone != "" {
				user.Phone = &phone
			}
			if err := l.svcCtx.UserModel.CreateWithTx(tx, user); err != nil {
				return err
			}
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
