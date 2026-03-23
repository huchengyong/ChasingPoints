package eventnews

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetEventNewsDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛事情报详情
func NewGetEventNewsDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetEventNewsDetailLogic {
	return &GetEventNewsDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetEventNewsDetailLogic) GetEventNewsDetail(req *types.GetEventNewsDetailReq) (resp *types.GetEventNewsDetailResp, err error) {
	if req == nil || req.EventNewsId <= 0 {
		return &types.GetEventNewsDetailResp{Success: false}, nil
	}

	item, err := l.svcCtx.EventNewsModel.FindPublishedById(req.EventNewsId)
	if err != nil {
		l.Logger.Errorf("获取赛事情报详情失败: eventNewsId=%d err=%v", req.EventNewsId, err)
		return &types.GetEventNewsDetailResp{Success: false}, nil
	}
	if item == nil {
		return &types.GetEventNewsDetailResp{Success: false}, nil
	}

	info := mapEventNewsInfo(*item)
	return &types.GetEventNewsDetailResp{
		Success:   true,
		EventNews: &info,
	}, nil
}
