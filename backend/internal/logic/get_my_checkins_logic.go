package logic

import (
	"context"

	"billiard_master/internal/svc"
	"billiard_master/internal/types"
	"billiard_master/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyCheckinsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取我的打卡记录
func NewGetMyCheckinsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyCheckinsLogic {
	return &GetMyCheckinsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMyCheckinsLogic) GetMyCheckins(req *types.GetMyCheckinsReq) (resp *types.GetMyCheckinsResp, err error) {
	userIdInt, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		l.Logger.Errorf("获取用户ID失败: %v", err)
		return &types.GetMyCheckinsResp{Success: false}, nil
	}

	list, total, err := l.svcCtx.VenueCheckinModel.FindByUser(userIdInt, req.Page, req.PageSize)
	if err != nil {
		l.Logger.Errorf("获取我的签到记录失败: userId=%d err=%v", userIdInt, err)
		return &types.GetMyCheckinsResp{Success: false}, nil
	}

	items := make([]types.VenueCheckinInfo, 0, len(list))
	for _, checkin := range list {
		venueName := ""
		venue, venueErr := l.svcCtx.VenueModel.FindById(checkin.VenueId)
		if venueErr != nil {
			l.Logger.Errorf("查询签到球馆失败: venueId=%d err=%v", checkin.VenueId, venueErr)
		}
		if venue != nil {
			venueName = venue.Name
		}

		items = append(items, types.VenueCheckinInfo{
			Id:        checkin.Id,
			VenueId:   checkin.VenueId,
			VenueName: venueName,
			CreatedAt: checkin.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	if items == nil {
		items = []types.VenueCheckinInfo{}
	}

	return &types.GetMyCheckinsResp{Success: true, Total: total, List: items}, nil
}
