package eventnews

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFeaturedEventNewsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取首页焦点赛事情报
func NewGetFeaturedEventNewsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFeaturedEventNewsLogic {
	return &GetFeaturedEventNewsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFeaturedEventNewsLogic) GetFeaturedEventNews() (resp *types.GetFeaturedEventNewsResp, err error) {
	items, err := l.svcCtx.EventNewsModel.FindPublishedMatching(-1, -1, "")
	if err != nil {
		l.Logger.Errorf("获取首页焦点赛事情报失败: err=%v", err)
		return &types.GetFeaturedEventNewsResp{Success: false}, nil
	}
	if len(items) == 0 {
		return &types.GetFeaturedEventNewsResp{Success: false}, nil
	}

	item := pickBestFeaturedEventNews(items, eventNewsNow())
	if item == nil {
		return &types.GetFeaturedEventNewsResp{Success: false}, nil
	}

	stages, err := l.svcCtx.EventNewsStageModel.FindByEventId(item.Id)
	if err != nil {
		l.Logger.Errorf("获取焦点赛事阶段失败: eventId=%d err=%v", item.Id, err)
		return &types.GetFeaturedEventNewsResp{Success: false}, nil
	}

	info := mapEventNewsInfo(*item, stages)
	return &types.GetFeaturedEventNewsResp{
		Success: true,
		Event:   &info,
	}, nil
}
