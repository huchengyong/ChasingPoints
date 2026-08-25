package auth

import (
	"context"
	"errors"
	"regexp"

	"chasing_points/internal/model"
	"chasing_points/internal/sms"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

var (
	errSmsBindUserMissing    = errors.New("sms bind user missing")
	errSmsBindMergeTargetBad = errors.New("sms bind merge target changed")
	errMergeTargetInactive   = errors.New("merge target inactive")
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
		svcCtx: svcCtx.WithContext(ctx),
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
	valid, err := l.svcCtx.CodeManager.VerifyCodeForScene(l.ctx, req.Phone, sms.SceneBind, req.SmsCode)
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
		// 账号合并：事务内迁移 OAuth 归属并删除源用户，成功后签发目标用户会话
		l.Logger.Infof("检测到账号合并场景: OAuth用户 %d -> 手机号用户 %d", userId, existingUser.Id)

		resultUser, err := l.mergeAccounts(userId, req.Phone)
		if err != nil {
			l.Logger.Errorf("账号合并失败: %v", err)
			if errors.Is(err, errSmsBindUserMissing) {
				return &types.BindPhoneResp{Success: false, Message: "用户不存在"}, nil
			}
			return &types.BindPhoneResp{Success: false, Message: "账号合并失败，请联系客服"}, nil
		}

		tokenPair, err := issueAuthTokenPair(resultUser.Id, l.svcCtx)
		if err != nil {
			l.Logger.Errorf("生成账号合并令牌失败: %v", err)
			return &types.BindPhoneResp{Success: false, Message: "账号合并失败，请联系客服"}, nil
		}

		l.Logger.Infof("账号合并成功: OAuth用户 %d 已迁移到手机号用户 %d", userId, resultUser.Id)
		return &types.BindPhoneResp{
			Success:       true,
			Message:       "账号已合并",
			MergedAccount: true,
			AccessToken:   tokenPair.AccessToken,
			RefreshToken:  tokenPair.RefreshToken,
			ExpiresIn:     tokenPair.ExpiresIn,
			NeedBindPhone: false,
			UserInfo:      buildAuthUserInfo(resultUser),
		}, nil
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

	return buildBindPhoneSuccessResp(), nil
}

// mergeAccounts atomically migrates the source OAuth user to the target phone
// user and deletes the source user. It returns the surviving target user.
func (l *BindPhoneLogic) mergeAccounts(sourceUserID int64, phone string) (*model.User, error) {
	var resultUser *model.User
	err := l.svcCtx.UserModel.Transaction(func(tx *gorm.DB) error {
		currentUser, err := l.svcCtx.UserModel.FindByIdForUpdateWithTx(tx, sourceUserID)
		if err != nil {
			return err
		}
		if currentUser == nil {
			return errSmsBindUserMissing
		}

		targetUser, err := l.svcCtx.UserModel.FindByPhoneForUpdateWithTx(tx, phone)
		if err != nil {
			return err
		}
		if targetUser == nil || targetUser.Id == currentUser.Id {
			return errSmsBindMergeTargetBad
		}
		if targetUser.Status != 1 {
			return errMergeTargetInactive
		}

		if err := l.svcCtx.OauthModel.UpdateUserIdWithTx(tx, currentUser.Id, targetUser.Id); err != nil {
			return err
		}
		if err := l.svcCtx.UserModel.DeleteByIdWithTx(tx, currentUser.Id); err != nil {
			return err
		}

		resultUser = targetUser
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resultUser, nil
}
