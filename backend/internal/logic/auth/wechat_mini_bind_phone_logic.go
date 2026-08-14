package auth

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type WechatMiniBindPhoneLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 微信小程序绑定手机号
func NewWechatMiniBindPhoneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WechatMiniBindPhoneLogic {
	return &WechatMiniBindPhoneLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *WechatMiniBindPhoneLogic) WechatMiniBindPhone(req *types.WechatMiniBindPhoneReq) (resp *types.WechatMiniBindPhoneResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.WechatMiniBindPhoneResp{Success: false, Message: "用户未登录"}, nil
	}
	if req == nil || strings.TrimSpace(req.Code) == "" {
		return &types.WechatMiniBindPhoneResp{Success: false, Message: "微信手机号授权无效"}, nil
	}
	if l.svcCtx == nil || l.svcCtx.WechatMiniClient == nil || l.svcCtx.Config.Auth.AccessSecret == "" {
		return &types.WechatMiniBindPhoneResp{Success: false, Message: "微信手机号绑定暂不可用"}, nil
	}

	phone, err := l.svcCtx.WechatMiniClient.GetPhoneNumber(l.ctx, req.Code)
	if err != nil {
		return &types.WechatMiniBindPhoneResp{Success: false, Message: err.Error()}, nil
	}
	if !mainlandPhonePattern.MatchString(phone) {
		return &types.WechatMiniBindPhoneResp{Success: false, Message: "手机号格式不正确"}, nil
	}

	var resultUser *model.User
	mergedAccount := false
	err = l.svcCtx.UserModel.Transaction(func(tx *gorm.DB) error {
		currentUser, err := l.svcCtx.UserModel.FindByIdForUpdateWithTx(tx, userID)
		if err != nil {
			return err
		}
		if currentUser == nil {
			return errWechatMiniUserMissing
		}

		existingUser, err := l.svcCtx.UserModel.FindByPhoneForUpdateWithTx(tx, phone)
		if err != nil {
			return err
		}
		if existingUser != nil && existingUser.Id != currentUser.Id {
			if existingUser.Status != 1 {
				return errMergeTargetInactive
			}
			if err := l.svcCtx.OauthModel.UpdateUserIdWithTx(tx, currentUser.Id, existingUser.Id); err != nil {
				return err
			}
			if err := l.svcCtx.UserModel.DeleteByIdWithTx(tx, currentUser.Id); err != nil {
				return err
			}
			mergedAccount = true
			resultUser = existingUser
			return nil
		}

		if err := l.svcCtx.UserModel.UpdatePhoneWithTx(tx, currentUser.Id, phone); err != nil {
			return err
		}
		currentUser.Phone = &phone
		resultUser = currentUser
		return nil
	})
	if err != nil {
		if errors.Is(err, errWechatMiniUserMissing) {
			return &types.WechatMiniBindPhoneResp{Success: false, Message: "用户不存在"}, nil
		}
		l.Logger.Errorf("微信小程序绑定手机号失败: %v", err)
		return &types.WechatMiniBindPhoneResp{Success: false, Message: "绑定失败，请稍后重试"}, nil
	}

	resp = &types.WechatMiniBindPhoneResp{
		Success:       true,
		MergedAccount: mergedAccount,
		NeedBindPhone: false,
		UserInfo:      buildAuthUserInfo(resultUser),
	}
	if !mergedAccount {
		resp.Message = "绑定成功"
		return resp, nil
	}

	tokenPair, err := issueAuthTokenPair(resultUser.Id, l.svcCtx)
	if err != nil {
		l.Logger.Errorf("生成账号合并令牌失败: %v", err)
		return &types.WechatMiniBindPhoneResp{Success: false, Message: "账号合并失败，请稍后重试"}, nil
	}
	resp.Message = "账号已合并"
	resp.AccessToken = tokenPair.AccessToken
	resp.RefreshToken = tokenPair.RefreshToken
	resp.ExpiresIn = tokenPair.ExpiresIn
	return resp, nil
}

var (
	mainlandPhonePattern     = regexp.MustCompile(`^1[3-9]\d{9}$`)
	errWechatMiniUserMissing = errors.New("wechat mini user missing")
)
