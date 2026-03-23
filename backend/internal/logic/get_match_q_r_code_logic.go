package logic

import (
	"context"
	"encoding/json"
	"time"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

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
		svcCtx: svcCtx,
	}
}

// QRCodeData 二维码数据结构
type QRCodeData struct {
	UserId   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Ts       int64  `json:"ts"` // 时间戳，用于标识二维码有效期
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

	// 构建二维码数据
	qrData := QRCodeData{
		UserId:   user.Id,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Ts:       time.Now().Unix(),
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(qrData)
	if err != nil {
		l.Logger.Errorf("序列化二维码数据失败: %v", err)
		return &types.GetMatchQRCodeResp{Success: false}, nil
	}

	l.Logger.Infof("用户 %d 获取匹配二维码", userId)

	return &types.GetMatchQRCodeResp{
		Success:    true,
		QrcodeData: string(jsonData),
	}, nil
}
