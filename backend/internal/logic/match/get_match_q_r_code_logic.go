package match

import (
	"context"
	"time"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

// 匹配码统一用普通链接二维码，微信、端内扫码和系统相机都能识别。
const matchInviteEntryLinkPrefix = "https://www.zhuifen.cn/go/?j="

type GetMatchQRCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取匹配二维码
func NewGetMatchQRCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMatchQRCodeLogic {
	return &GetMatchQRCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *GetMatchQRCodeLogic) GetMatchQRCode() (resp *types.GetMatchQRCodeResp, err error) {
	// 获取用户ID
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMatchQRCodeResp{Success: false}, nil
	}

	// 查询用户信息
	user, err := l.svcCtx.UserModel.FindById(userId)
	if err != nil || user == nil {
		l.Logger.Errorf("查询用户失败: %v", err)
		return &types.GetMatchQRCodeResp{Success: false}, nil
	}

	if user.Status != 1 {
		return &types.GetMatchQRCodeResp{Success: false}, nil
	}
	signer, err := newMatchInviteSigner(l.svcCtx)
	if err != nil {
		l.Logger.Errorf("初始化匹配邀请签名器失败: %v", err)
		return &types.GetMatchQRCodeResp{Success: false}, nil
	}
	inviteToken, claims, err := signer.Issue(user.Id)
	if err != nil {
		l.Logger.Errorf("签发匹配邀请失败: userId=%d err=%v", userId, err)
		return &types.GetMatchQRCodeResp{Success: false}, nil
	}

	// 短码有效期跟随 token，写入失败时不返回无法兑换的码。
	ttl := time.Duration(claims.ExpiresAt-claims.IssuedAt) * time.Second
	code, err := l.svcCtx.MatchInviteCodeStore.Save(l.ctx, inviteToken, ttl)
	if err != nil {
		l.Logger.Errorf("写入匹配邀请短码失败: userId=%d err=%v", userId, err)
		return &types.GetMatchQRCodeResp{Success: false}, nil
	}

	l.Logger.Infof("用户 %d 获取匹配二维码", userId)

	return &types.GetMatchQRCodeResp{
		Success:          true,
		QrcodeData:       matchInviteEntryLinkPrefix + code,
		InviteToken:      inviteToken,
		ExpiresInSeconds: claims.ExpiresAt - claims.IssuedAt,
	}, nil
}
