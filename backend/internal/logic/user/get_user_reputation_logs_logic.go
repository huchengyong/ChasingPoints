package user

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserReputationLogsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserReputationLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserReputationLogsLogic {
	return &GetUserReputationLogsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserReputationLogsLogic) GetUserReputationLogs(req *types.UserReputationLogsReq) (*types.UserReputationLogsResp, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.UserReputationLogModel == nil {
		return &types.UserReputationLogsResp{Success: false}, nil
	}

	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户信誉日志时解析用户ID失败: %v", err)
		return &types.UserReputationLogsResp{Success: false}, nil
	}

	page := 1
	pageSize := 20
	if req != nil {
		page = req.Page
		pageSize = req.PageSize
	}

	logs, total, err := l.svcCtx.UserReputationLogModel.FindListForUser(userID, page, pageSize)
	if err != nil {
		l.Logger.Errorf("获取用户信誉日志失败: userID=%d err=%v", userID, err)
		return &types.UserReputationLogsResp{Success: false}, nil
	}

	return buildUserReputationLogsResp(logs, total), nil
}
