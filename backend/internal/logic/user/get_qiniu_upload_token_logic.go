package user

import (
	"context"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetQiniuUploadTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取七牛上传凭证
func NewGetQiniuUploadTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetQiniuUploadTokenLogic {
	return &GetQiniuUploadTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetQiniuUploadTokenLogic) GetQiniuUploadToken(req *types.GetQiniuUploadTokenReq) (resp *types.GetQiniuUploadTokenResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return &types.GetQiniuUploadTokenResp{
			Success: false,
			Message: "用户未登录",
		}, nil
	}

	if l.svcCtx.QiniuUploadService == nil {
		return &types.GetQiniuUploadTokenResp{
			Success: false,
			Message: "上传服务未配置",
		}, nil
	}

	credential, err := l.svcCtx.QiniuUploadService.IssueAvatarUpload(userID, req.FileExt, logicx.NowUTC8())
	if err != nil {
		l.Logger.Errorf("签发七牛上传凭证失败: %v", err)
		return &types.GetQiniuUploadTokenResp{
			Success: false,
			Message: "获取上传凭证失败",
		}, nil
	}

	return &types.GetQiniuUploadTokenResp{
		Success:     true,
		Message:     "ok",
		UploadToken: credential.UploadToken,
		Key:         credential.Key,
		UploadUrl:   credential.UploadURL,
		Domain:      credential.Domain,
	}, nil
}
