package admin

import (
	"context"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetEventNewsPlayersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取赛讯球员列表（维护中文名）
func NewAdminGetEventNewsPlayersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetEventNewsPlayersLogic {
	return &AdminGetEventNewsPlayersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

const (
	adminEventNewsPlayerDefaultPageSize = 20
	adminEventNewsPlayerMaxPageSize     = 100
)

func (l *AdminGetEventNewsPlayersLogic) AdminGetEventNewsPlayers(req *types.AdminEventNewsPlayerListReq) (resp *types.AdminEventNewsPlayerListResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminEventNewsPlayerListResp{Code: 401, Success: false, Message: "未登录或登录已过期", List: []types.AdminEventNewsPlayerInfo{}}, nil
	}
	if req == nil {
		return &types.AdminEventNewsPlayerListResp{Code: 400, Success: false, Message: "请求参数错误", List: []types.AdminEventNewsPlayerInfo{}}, nil
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > adminEventNewsPlayerMaxPageSize {
		pageSize = adminEventNewsPlayerDefaultPageSize
	}
	keyword := []rune(req.Keyword)
	if len(keyword) > adminEventNewsPlayerNameZhMaxRunes*2 {
		keyword = keyword[:adminEventNewsPlayerNameZhMaxRunes*2]
	}

	players, total, err := l.svcCtx.PlayerModel.FindAdminList(model.PlayerAdminFilter{
		Page:      page,
		PageSize:  pageSize,
		Keyword:   string(keyword),
		MissingZh: req.MissingZh == 1,
	})
	if err != nil {
		l.Logger.Errorf("获取赛讯球员列表失败: err=%v", err)
		return &types.AdminEventNewsPlayerListResp{Code: 500, Success: false, Message: "获取球员列表失败", List: []types.AdminEventNewsPlayerInfo{}}, nil
	}

	list := make([]types.AdminEventNewsPlayerInfo, 0, len(players))
	for _, player := range players {
		list = append(list, types.AdminEventNewsPlayerInfo{
			Id:             player.Id,
			SourceType:     player.SourceType,
			SourcePlayerId: player.SourcePlayerId,
			DisplayName:    player.DisplayName,
			NameZh:         player.NameZh,
			CountryCode:    player.CountryCode,
			Avatar:         player.Avatar,
			UpdatedAt:      player.UpdatedAt.Format(time.DateTime),
		})
	}
	return &types.AdminEventNewsPlayerListResp{Code: 0, Success: true, Message: "获取成功", Total: total, List: list}, nil
}
