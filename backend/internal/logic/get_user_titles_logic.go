package logic

import (
	"context"

	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserTitlesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户称号列表
func NewGetUserTitlesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserTitlesLogic {
	return &GetUserTitlesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserTitlesLogic) GetUserTitles() (resp *types.GetUserTitlesResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetUserTitlesResp{Success: false}, nil
	}

	titles, err := l.svcCtx.UserTitleModel.FindByUserId(userIdInt)
	if err != nil {
		l.Logger.Errorf("查询用户称号失败: %v", err)
		return &types.GetUserTitlesResp{Success: false}, nil
	}

	list := make([]types.TitleInfo, 0, len(titles))
	for _, title := range titles {
		list = append(list, types.TitleInfo{
			Id:        title.Id,
			TitleName: title.TitleName,
			Source:    title.Source,
			Equipped:  title.Equipped == 1,
			CreatedAt: title.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &types.GetUserTitlesResp{
		Success: true,
		List:    list,
	}, nil
}
