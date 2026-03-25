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
	eventID := resolveEventNewsDetailID(req)
	if eventID <= 0 {
		return &types.GetEventNewsDetailResp{Success: false}, nil
	}

	item, err := l.svcCtx.EventNewsModel.FindPublishedById(eventID)
	if err != nil {
		l.Logger.Errorf("获取赛事情报详情失败: eventId=%d err=%v", eventID, err)
		return &types.GetEventNewsDetailResp{Success: false}, nil
	}
	if item == nil {
		return &types.GetEventNewsDetailResp{Success: false}, nil
	}

	stages, err := l.svcCtx.EventNewsStageModel.FindByEventId(item.Id)
	if err != nil {
		l.Logger.Errorf("获取赛事阶段失败: eventId=%d err=%v", eventID, err)
		return &types.GetEventNewsDetailResp{Success: false}, nil
	}

	info := mapEventNewsInfo(*item, stages)
	stageInfos := make([]types.EventNewsStageInfo, 0, len(stages))
	for _, stage := range stages {
		stageInfos = append(stageInfos, mapEventNewsStageInfo(stage))
	}

	return &types.GetEventNewsDetailResp{
		Success:   true,
		Event:     &info,
		EventNews: &info,
		Stages:    stageInfos,
	}, nil
}

func resolveEventNewsDetailID(req *types.GetEventNewsDetailReq) int64 {
	if req == nil {
		return 0
	}
	if req.EventId > 0 {
		return req.EventId
	}
	return req.EventNewsId
}
