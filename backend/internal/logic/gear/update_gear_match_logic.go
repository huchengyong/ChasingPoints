package gear

import (
	"context"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
	"chasing_points/internal/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateGearMatchLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 修改或补记本场用杆
func NewUpdateGearMatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateGearMatchLogic {
	return &UpdateGearMatchLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx.WithContext(ctx),
	}
}

func (l *UpdateGearMatchLogic) UpdateGearMatch(req *types.UpdateGearMatchReq) (resp *types.GetGearMatchResp, err error) {
	userId, err := utils.GetUserIDFromCtx(l.ctx)
	if err != nil {
		return failedMatchResp("", "用户未登录"), nil
	}
	if req == nil {
		return failedMatchResp(reasonMatchNotFound, "对局不存在"), nil
	}

	match, refused, err := loadViewableMatch(l.svcCtx, userId, req.MatchId)
	if err != nil {
		l.Logger.Errorf("读取对局失败: matchId=%d err=%v", req.MatchId, err)
		return failedMatchResp("", "保存失败"), nil
	}
	if refused != nil {
		return refused, nil
	}

	now := time.Now()
	if !now.Before(matchEditDeadline(match)) {
		return failedMatchResp("", "对局完成已超过 7 天，本场用杆不能再修改"), nil
	}

	if err := l.validate(userId, match, req); err != nil {
		message := userMessage(err, "保存失败")
		if message == "保存失败" {
			l.Logger.Errorf("校验本场用杆失败: matchId=%d userId=%d err=%v", match.Id, userId, err)
		}
		return failedMatchResp("", message), nil
	}

	row := &model.MatchGearSnapshot{
		MatchId: match.Id, UserId: userId,
		CueItemId: req.CueItemId, TipProductId: req.TipProductId, ChalkProductId: req.ChalkProductId, BreakItemId: req.BreakItemId,
	}
	if row.IsEmpty() {
		// 四项都清空等于「这场没用记录的装备」，直接删除这位选手的记录。
		err = l.svcCtx.GearModel.DeleteSnapshot(match.Id, userId)
	} else {
		err = l.svcCtx.GearModel.UpsertManualSnapshot(row)
	}
	if err != nil {
		l.Logger.Errorf("保存本场用杆失败: matchId=%d userId=%d err=%v", match.Id, userId, err)
		return failedMatchResp("", "保存失败"), nil
	}

	resp, err = buildMatchGearResp(l.svcCtx, userId, match, now)
	if err != nil {
		// 已经保存成功，只是刷新视图失败，让客户端重新读取即可。
		l.Logger.Errorf("保存后读取本场用杆失败: matchId=%d userId=%d err=%v", match.Id, userId, err)
		return &types.GetGearMatchResp{Success: true, Message: "已保存", MatchId: match.Id}, nil
	}
	resp.Message = "已保存"
	return resp, nil
}

// validate 校验本场用杆的四个字段：装备必须是本人未删除、类别与身份正确的装备，
// 皮头与巧克必须是目录里对应类别的型号（已停用的也接受，用来保留原值）。
func (l *UpdateGearMatchLogic) validate(userId int64, match *model.Match, req *types.UpdateGearMatchReq) error {
	gears := l.svcCtx.GearModel
	ids := make([]int64, 0, 2)
	for _, id := range []int64{req.CueItemId, req.BreakItemId} {
		if id != 0 {
			ids = append(ids, id)
		}
	}
	items, err := gears.FindItemsByIds(ids)
	if err != nil {
		return err
	}
	ownedItem := func(id int64, category string) (*model.UserGearItem, error) {
		item, ok := items[id]
		if !ok || item.UserId != userId || item.Status == model.GearItemStatusDeleted {
			return nil, newUserError("装备不存在")
		}
		if item.Category != category {
			return nil, newUserError("装备类别不符")
		}
		return &item, nil
	}

	if req.CueItemId != 0 {
		if _, err := ownedItem(req.CueItemId, model.GearCategoryCue); err != nil {
			return err
		}
	}
	if req.BreakItemId != 0 {
		if !model.GearGameTypeRecordsBreak(match.GameType) {
			return newUserError("该球种不记录冲杆")
		}
		item, err := ownedItem(req.BreakItemId, model.GearCategoryBreakJump)
		if err != nil {
			return err
		}
		roles, err := itemRoles(gears, item)
		if err != nil {
			return err
		}
		if !model.GearRolesContain(roles, model.GearRoleBreak) {
			return newUserError("这件装备不是冲杆")
		}
	}
	if err := checkSubProduct(gears, req.TipProductId, model.GearCategoryTip, false); err != nil {
		return err
	}
	return checkSubProduct(gears, req.ChalkProductId, model.GearCategoryChalk, false)
}
