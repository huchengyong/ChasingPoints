package feedback

import (
	"context"

	sharedlogic "chasing_points/internal/logic"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateFeedbackTicketLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 公开提交投诉举报与意见反馈
func NewCreateFeedbackTicketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateFeedbackTicketLogic {
	return &CreateFeedbackTicketLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *CreateFeedbackTicketLogic) CreateFeedbackTicket(req *types.CreateFeedbackTicketReq) (resp *types.CreateFeedbackTicketResp, err error) {
	resp, err = sharedlogic.CreateFeedbackTicket(l.ctx, l.svcCtx, sharedlogic.FeedbackTicketSubmission{
		UserId:   0,
		Source:   req.Source,
		Category: req.Category,
		Content:  req.Content,
		Contact:  req.Contact,
	})
	if err != nil {
		l.Logger.Errorf("创建公开反馈工单失败: err=%v", err)
	}
	return resp, err
}
