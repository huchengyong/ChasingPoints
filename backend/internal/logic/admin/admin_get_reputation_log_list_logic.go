package admin

import (
	"context"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetReputationLogListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取信誉变更日志列表
func NewAdminGetReputationLogListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetReputationLogListLogic {
	return &AdminGetReputationLogListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *AdminGetReputationLogListLogic) AdminGetReputationLogList(req *types.AdminReputationLogListReq) (resp *types.AdminReputationLogListResp, err error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.UserReputationLogModel == nil {
		if l != nil {
			l.Logger.Errorf("获取信誉日志列表失败: 依赖未初始化")
		}
		return adminReputationLogListFailedResp("获取信誉日志失败"), nil
	}

	if req == nil {
		req = &types.AdminReputationLogListReq{}
	}

	logs, total, err := l.svcCtx.UserReputationLogModel.FindListForAdmin(model.UserReputationLogAdminFilter{
		Page:       req.Page,
		PageSize:   req.PageSize,
		UserID:     req.UserId,
		ChangeType: req.ChangeType,
		ReasonCode: req.ReasonCode,
	})
	if err != nil {
		l.Logger.Errorf("获取信誉日志列表失败: err=%v", err)
		return adminReputationLogListFailedResp("获取信誉日志失败"), nil
	}

	nicknameByUserID := l.buildNicknameByUserID(logs)

	return buildAdminReputationLogListResp(logs, nicknameByUserID, total), nil
}

func (l *AdminGetReputationLogListLogic) buildNicknameByUserID(logs []model.UserReputationLog) map[int64]string {
	nicknameByUserID := make(map[int64]string, len(logs))
	seen := make(map[int64]struct{}, len(logs))
	userIDs := make([]int64, 0, len(logs))

	for i := range logs {
		userID := logs[i].UserID
		if userID <= 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}

	if len(userIDs) == 0 || l == nil || l.svcCtx == nil || l.svcCtx.DB == nil {
		return nicknameByUserID
	}

	var users []model.User
	if err := l.svcCtx.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		l.Logger.Errorf("补充信誉日志用户昵称失败，已降级为空昵称: err=%v", err)
		return nicknameByUserID
	}
	for i := range users {
		nicknameByUserID[users[i].Id] = users[i].Nickname
	}
	return nicknameByUserID
}

func adminReputationLogListFailedResp(message string) *types.AdminReputationLogListResp {
	return &types.AdminReputationLogListResp{
		Code:    500,
		Success: false,
		Message: message,
		List:    []types.AdminReputationLogItem{},
	}
}
