package auth

import (
	"context"
	"regexp"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendSmsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发送短信验证码
func NewSendSmsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendSmsLogic {
	return &SendSmsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendSmsLogic) SendSms(req *types.SendSmsReq) (resp *types.SendSmsResp, err error) {
	// 验证手机号格式
	if !regexp.MustCompile(`^1[3-9]\d{9}$`).MatchString(req.Phone) {
		return &types.SendSmsResp{
			Success: false,
			Message: "手机号格式不正确",
		}, nil
	}

	// 检查发送间隔
	if err := l.svcCtx.CodeManager.CheckSendInterval(l.ctx, req.Phone); err != nil {
		return &types.SendSmsResp{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// 生成验证码
	code := l.svcCtx.CodeManager.GenerateCode()

	// 保存验证码到Redis
	if err := l.svcCtx.CodeManager.SaveCode(l.ctx, req.Phone, code); err != nil {
		l.Logger.Errorf("保存验证码失败: %v", err)
		return &types.SendSmsResp{
			Success: false,
			Message: "系统错误，请稍后重试",
		}, nil
	}

	// 发送短信
	if err := l.svcCtx.SmsClient.SendVerificationCode(req.Phone, code); err != nil {
		l.Logger.Errorf("发送短信失败: %v", err)
		return &types.SendSmsResp{
			Success: false,
			Message: "发送短信失败，请稍后重试",
		}, nil
	}

	// 记录发送时间
	if err := l.svcCtx.CodeManager.RecordSendTime(l.ctx, req.Phone); err != nil {
		l.Logger.Errorf("记录发送时间失败: %v", err)
	}

	return &types.SendSmsResp{
		Success: true,
		Message: "验证码已发送",
	}, nil
}
