package admin

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"chasing_points/internal/logic/eventnews"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUpdateEventNewsPlayerLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新赛讯球员中文名
func NewAdminUpdateEventNewsPlayerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateEventNewsPlayerLogic {
	return &AdminUpdateEventNewsPlayerLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

const adminEventNewsPlayerNameZhMaxRunes = 32

// normalizeAdminPlayerNameZh 去掉首尾空白；空字符串表示清除中文名。拒绝超长与控制字符。
func normalizeAdminPlayerNameZh(value string) (string, string) {
	name := strings.TrimSpace(value)
	if utf8.RuneCountInString(name) > adminEventNewsPlayerNameZhMaxRunes {
		return "", "中文名不能超过 32 个字"
	}
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", "中文名包含非法字符"
	}
	return name, ""
}

func (l *AdminUpdateEventNewsPlayerLogic) AdminUpdateEventNewsPlayer(req *types.AdminEventNewsPlayerUpdateReq) (resp *types.AdminWriteResp, err error) {
	if _, err := utils.GetAdminIDFromCtx(l.ctx); err != nil {
		return &types.AdminWriteResp{Code: 401, Success: false, Message: "未登录或登录已过期"}, nil
	}
	if req == nil || req.PlayerId <= 0 {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: "请求参数错误"}, nil
	}
	nameZh, message := normalizeAdminPlayerNameZh(req.NameZh)
	if message != "" {
		return &types.AdminWriteResp{Code: 400, Success: false, Message: message}, nil
	}

	player, err := l.svcCtx.PlayerModel.FindById(req.PlayerId)
	if err != nil {
		l.Logger.Errorf("查询赛讯球员失败: playerId=%d err=%v", req.PlayerId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "更新中文名失败"}, nil
	}
	if player == nil {
		return &types.AdminWriteResp{Code: 404, Success: false, Message: "球员不存在"}, nil
	}
	if player.NameZh == nameZh {
		return &types.AdminWriteResp{Code: 0, Success: true, Message: "更新成功"}, nil
	}

	if err := l.svcCtx.PlayerModel.UpdateNameZh(player.Id, nameZh); err != nil {
		l.Logger.Errorf("更新赛讯球员中文名失败: playerId=%d err=%v", req.PlayerId, err)
		return &types.AdminWriteResp{Code: 500, Success: false, Message: "更新中文名失败"}, nil
	}
	// 赛讯列表与详情按版本缓存，改名后立即失效；失败只记录，缓存最多 5 分钟后自然过期
	if err := eventnews.BumpEventNewsCacheVersion(l.ctx, l.svcCtx); err != nil {
		l.Logger.Errorf("刷新赛讯缓存版本失败: playerId=%d err=%v", req.PlayerId, err)
	}
	return &types.AdminWriteResp{Code: 0, Success: true, Message: "更新成功"}, nil
}
