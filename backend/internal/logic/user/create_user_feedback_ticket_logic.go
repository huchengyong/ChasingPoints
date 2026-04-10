package user

import (
	"context"

	sharedlogic "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateUserFeedbackTicketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 提交投诉举报与意见反馈
func NewCreateUserFeedbackTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserFeedbackTicketLogic {
	return &CreateUserFeedbackTicketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateUserFeedbackTicketLogic) CreateUserFeedbackTicket(req *types.CreateFeedbackTicketReq) (resp *types.CreateFeedbackTicketResp, err error) {
	userID, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.CreateFeedbackTicketResp{Success: false, Message: "请先完成登录"}, nil
	}

	resp, err = sharedlogic.CreateFeedbackTicket(l.ctx, l.svcCtx, sharedlogic.FeedbackTicketSubmission{
		UserId:   userID,
		Source:   model.FeedbackTicketSourceApp,
		Category: req.Category,
		Content:  req.Content,
		Contact:  req.Contact,
	})
	if err != nil {
		l.Logger.Errorf("创建用户反馈工单失败: userId=%d err=%v", userID, err)
	}
	return resp, err
}
