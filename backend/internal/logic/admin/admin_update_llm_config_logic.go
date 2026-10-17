package admin

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUpdateLlmConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新大模型调用配置
func NewAdminUpdateLlmConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateLlmConfigLogic {
	return &AdminUpdateLlmConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminUpdateLlmConfigLogic) AdminUpdateLlmConfig(req *types.AdminLlmConfigUpdateReq) (resp *types.AdminWriteResp, err error) {
	existing, err := l.svcCtx.LlmConfigModel.FindByKey(model.DefaultLlmConfigKey)
	if err != nil {
		l.Logger.Errorf("读取大模型调用配置失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "保存配置失败",
		}, nil
	}
	storedAPIKey := ""
	if existing != nil {
		storedAPIKey = existing.APIKey
	}

	adminID, _ := utils.GetAdminIDFromCtx(l.ctx)
	cfg, err := buildLlmConfigModel(req, storedAPIKey, int64(adminID))
	if err != nil {
		return &types.AdminWriteResp{
			Code:    400,
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// 日志只记录错误，不输出请求体，避免 API Key 进入日志。
	if err := l.svcCtx.LlmConfigModel.Upsert(cfg); err != nil {
		l.Logger.Errorf("保存大模型调用配置失败: %v", err)
		return &types.AdminWriteResp{
			Code:    500,
			Success: false,
			Message: "保存配置失败",
		}, nil
	}

	return &types.AdminWriteResp{
		Code:    0,
		Success: true,
		Message: "保存成功",
	}, nil
}
