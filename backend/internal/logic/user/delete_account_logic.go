package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"chasing_points/internal/model"
	oauthverify "chasing_points/internal/pkg/oauth"
	"chasing_points/internal/pkg/ws"
	"chasing_points/internal/sms"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const deleteAccountConfirmText = "注销账号"

var (
	errAccountDeletionMissing       = errors.New("account deletion user missing")
	errAccountDeletionUnavailable   = errors.New("account deletion unavailable")
	errAccountDeletionUserInactive  = errors.New("account deletion user inactive")
	errAccountDeletionAlreadyClosed = errors.New("account deletion already closed")
	errAccountDeletionSmsRequired   = errors.New("account deletion sms verification required")
	errAccountDeletionSmsInvalid    = errors.New("account deletion sms verification invalid")
)

type accountDeletionVerifier func(tx *gorm.DB, user *model.User) error

type DeleteAccountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 注销账号
func NewDeleteAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAccountLogic {
	if svcCtx != nil {
		svcCtx = svcCtx.WithContext(ctx)
	}
	return &DeleteAccountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteAccountLogic) DeleteAccount(req *types.DeleteAccountReq) (resp *types.CommonResp, err error) {
	if req == nil || strings.TrimSpace(req.ConfirmText) != deleteAccountConfirmText {
		return &types.CommonResp{Success: false, Message: "请输入“注销账号”确认不可逆操作"}, nil
	}
	if l == nil || l.svcCtx == nil || l.svcCtx.UserModel == nil || l.svcCtx.UserDataLifecycleModel == nil ||
		l.svcCtx.OauthModel == nil || l.svcCtx.MatchModel == nil {
		return &types.CommonResp{Success: false, Message: "注销服务暂不可用"}, nil
	}

	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.CommonResp{Success: false, Message: "用户未登录"}, nil
	}
	currentUser, err := l.svcCtx.UserModel.FindByIdWithContext(l.ctx, userID)
	if err != nil {
		l.Logger.Errorf("查询待注销用户失败: userId=%d err=%v", userID, err)
		return &types.CommonResp{Success: false, Message: "注销服务暂不可用"}, nil
	}
	if currentUser == nil {
		return &types.CommonResp{Success: true, Message: "账号已注销"}, nil
	}
	if currentUser.Status != 1 {
		return &types.CommonResp{Success: false, Message: "当前账号无法注销"}, nil
	}

	verifyType := strings.ToLower(strings.TrimSpace(req.VerifyType))
	var verification accountDeletionVerifier
	if currentUser.Phone != nil && strings.TrimSpace(*currentUser.Phone) != "" {
		if verifyType != "sms" || strings.TrimSpace(req.SmsCode) == "" || l.svcCtx.CodeManager == nil {
			return &types.CommonResp{Success: false, Message: "请使用注销验证码完成身份复核"}, nil
		}
		verification = func(_ *gorm.DB, user *model.User) error {
			if user.Phone == nil || strings.TrimSpace(*user.Phone) == "" {
				return errAccountDeletionSmsRequired
			}
			valid, verifyErr := l.svcCtx.CodeManager.VerifyCodeForScene(l.ctx, *user.Phone, sms.SceneDeleteAccount, req.SmsCode)
			if verifyErr != nil {
				if errors.Is(verifyErr, sms.ErrCodeExpired) {
					return verifyErr
				}
				return errAccountDeletionUnavailable
			}
			if !valid {
				return errAccountDeletionSmsInvalid
			}
			return nil
		}
	} else {
		if verifyType != "oauth" {
			return &types.CommonResp{Success: false, Message: "请重新完成第三方身份验证"}, nil
		}
		identity, verifyErr := l.verifyOAuthDeletionCredential(userID, req)
		if verifyErr != nil {
			return &types.CommonResp{Success: false, Message: accountDeletionVerificationMessage(verifyErr)}, nil
		}
		verification = l.oauthDeletionAssociationVerifier(userID, identity)
	}

	closedMatches, alreadyDeleted, err := l.deleteAccountInTransaction(userID, verification)
	if err != nil {
		if errors.Is(err, errAccountDeletionAlreadyClosed) {
			return &types.CommonResp{Success: true, Message: "账号已注销"}, nil
		}
		return &types.CommonResp{Success: false, Message: accountDeletionVerificationMessage(err)}, nil
	}
	if alreadyDeleted {
		return &types.CommonResp{Success: true, Message: "账号已注销"}, nil
	}

	l.notifyAccountDeletion(closedMatches, userID)
	if ws.GlobalHub != nil {
		ws.GlobalHub.DisconnectUser(userID)
	}
	return &types.CommonResp{Success: true, Message: "账号已注销"}, nil
}

func (l *DeleteAccountLogic) oauthDeletionAssociationVerifier(userID int64, identity *oauthverify.Identity) accountDeletionVerifier {
	return func(tx *gorm.DB, _ *model.User) error {
		if identity == nil || strings.TrimSpace(identity.Provider) == "" || strings.TrimSpace(identity.Subject) == "" {
			return errDeleteAccountOAuthVerificationFailed
		}
		oauth, err := l.svcCtx.OauthModel.FindByProviderAndOpenIdWithTx(tx, identity.Provider, identity.Subject)
		if err != nil {
			return errDeleteAccountOAuthServiceUnavailable
		}
		if oauth == nil || oauth.UserId != userID {
			return errDeleteAccountOAuthVerificationFailed
		}
		return nil
	}
}

func (l *DeleteAccountLogic) deleteAccountInTransaction(userID int64, verification accountDeletionVerifier) ([]model.Match, bool, error) {
	var closedMatches []model.Match
	alreadyDeleted := false
	err := l.svcCtx.UserModel.TransactionWithContext(l.ctx, func(tx *gorm.DB) error {
		user, findErr := l.svcCtx.UserModel.FindByIdIncludingDeletedForUpdateWithTx(tx, userID)
		if findErr != nil {
			return findErr
		}
		if user == nil {
			return errAccountDeletionMissing
		}
		if user.DeletedAt.Valid {
			alreadyDeleted = true
			return errAccountDeletionAlreadyClosed
		}
		if user.Status != 1 {
			return errAccountDeletionUserInactive
		}
		if verification != nil {
			if verifyErr := verification(tx, user); verifyErr != nil {
				return verifyErr
			}
		}

		var cancelErr error
		closedMatches, cancelErr = l.svcCtx.MatchModel.CancelActiveForAccountDeletionWithTx(tx, userID, time.Now())
		if cancelErr != nil {
			return cancelErr
		}
		if err := l.svcCtx.UserDataLifecycleModel.AnonymizeRetainedAccountFactsWithTx(tx, userID); err != nil {
			return err
		}
		phone := ""
		if user.Phone != nil {
			phone = *user.Phone
		}
		if err := l.svcCtx.UserDataLifecycleModel.DeleteAccountRelationsWithTx(tx, userID, phone); err != nil {
			return err
		}
		if err := l.svcCtx.OauthModel.DeleteByUserIdWithTx(tx, userID); err != nil {
			return err
		}
		return l.svcCtx.UserModel.AnonymizeAndSoftDeleteWithTx(tx, userID)
	})
	return closedMatches, alreadyDeleted, err
}

func accountDeletionVerificationMessage(err error) string {
	switch {
	case errors.Is(err, errDeleteAccountOAuthVerificationRequired):
		return "请重新完成第三方身份验证"
	case errors.Is(err, errDeleteAccountOAuthVerificationFailed):
		return "第三方身份验证失败"
	case errors.Is(err, errDeleteAccountOAuthServiceUnavailable), errors.Is(err, errAccountDeletionUnavailable):
		return "身份复核服务暂不可用"
	case errors.Is(err, sms.ErrCodeExpired):
		return "验证码已过期，请重新获取"
	case errors.Is(err, errAccountDeletionSmsInvalid), errors.Is(err, errAccountDeletionSmsRequired):
		return "注销验证码错误"
	case errors.Is(err, errAccountDeletionUserInactive):
		return "当前账号无法注销"
	case errors.Is(err, errAccountDeletionMissing):
		return "账号不存在或已注销"
	default:
		return "注销服务暂不可用"
	}
}

func (l *DeleteAccountLogic) notifyAccountDeletion(matches []model.Match, deletedUserID int64) {
	for _, match := range matches {
		if ws.GlobalHub != nil {
			ws.GlobalHub.BroadcastToMatch(match.Id, &ws.Message{
				Type: "match_cancelled",
				Data: map[string]any{"match_id": match.Id, "reason": "account_deleted"},
			})
		}

		targets := []int64{match.UserId}
		if match.OpponentId != nil {
			targets = append(targets, *match.OpponentId)
		}
		if match.RefereeUserId != nil {
			targets = append(targets, *match.RefereeUserId)
		}
		for _, targetID := range targets {
			if targetID <= 0 || targetID == deletedUserID || l.svcCtx.NotificationModel == nil {
				continue
			}
			payload, _ := json.Marshal(map[string]any{"match_id": match.Id, "reason": "account_deleted"})
			dedupeKey := fmt.Sprintf("account_deleted:%d:%d", match.Id, targetID)
			_, notifyErr := l.svcCtx.NotificationModel.CreateIfAbsent(&model.Notification{
				UserId:    targetID,
				Type:      "match_cancelled",
				DedupeKey: &dedupeKey,
				Title:     "对局已结束",
				Content:   "因参与者账号注销，对局已结束",
				Data:      stringPointer(string(payload)),
			})
			if notifyErr != nil {
				l.Logger.Errorf("创建账号注销对局通知失败: matchId=%d userId=%d err=%v", match.Id, targetID, notifyErr)
			}
		}
	}
}

func stringPointer(value string) *string {
	return &value
}
