package gear

import (
	"time"

	logicx "chasing_points/internal/logic"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

// 对局用杆接口里客户端用来区分状态的原因代码。
const (
	reasonMatchNotFound       = "not_found"
	reasonMatchNotParticipant = "not_participant"
	reasonMatchNotCompleted   = "not_completed"
)

const matchCompletedStatus = 2

func failedMatchResp(reason, message string) *types.GetGearMatchResp {
	return &types.GetGearMatchResp{Success: false, Message: message, Reason: reason}
}

// loadViewableMatch 读取对局并校验查看者是已完成对局的选手。
// 裁判、观战者和其他用户都按"不是选手"处理；进行中的对局返回不可用，也不会提前生成记录。
func loadViewableMatch(svcCtx *svc.ServiceContext, viewerId, matchId int64) (*model.Match, *types.GetGearMatchResp, error) {
	if matchId <= 0 {
		return nil, failedMatchResp(reasonMatchNotFound, "对局不存在"), nil
	}
	match, err := svcCtx.MatchModel.FindById(matchId)
	if err != nil {
		return nil, nil, err
	}
	if match == nil {
		return nil, failedMatchResp(reasonMatchNotFound, "对局不存在"), nil
	}
	if !isMatchPlayer(match, viewerId) {
		return nil, failedMatchResp(reasonMatchNotParticipant, "只有对局双方可以查看本场用杆"), nil
	}
	if match.Status != matchCompletedStatus {
		return nil, failedMatchResp(reasonMatchNotCompleted, "对局结束后才能查看本场用杆"), nil
	}
	return match, nil, nil
}

func isMatchPlayer(match *model.Match, userId int64) bool {
	if userId <= 0 {
		return false
	}
	return match.UserId == userId || (match.OpponentId != nil && *match.OpponentId == userId)
}

// matchOpponentOf 返回对局中另一位注册选手，对手不是注册用户时为 0。
func matchOpponentOf(match *model.Match, userId int64) int64 {
	if match.UserId == userId {
		if match.OpponentId != nil && *match.OpponentId > 0 && *match.OpponentId != match.UserId {
			return *match.OpponentId
		}
		return 0
	}
	return match.UserId
}

func matchEditDeadline(match *model.Match) time.Time {
	return logicx.MatchCompletedAt(match).Add(model.GearSnapshotEditWindow)
}

// buildMatchGearResp 读取双方的本场用杆并按查看者组装，固定 4 条 SQL（记录、装备、型号、品牌），加上读取对局共 5 条。
func buildMatchGearResp(svcCtx *svc.ServiceContext, viewerId int64, match *model.Match, now time.Time) (*types.GetGearMatchResp, error) {
	opponentId := matchOpponentOf(match, viewerId)
	snapshots, err := svcCtx.GearModel.FindSnapshotsByMatch(match.Id)
	if err != nil {
		return nil, err
	}
	byUser := make(map[int64]model.MatchGearSnapshot, len(snapshots))
	itemIds := make([]int64, 0, len(snapshots)*2)
	productIds := make([]int64, 0, len(snapshots)*2)
	for _, snapshot := range snapshots {
		byUser[snapshot.UserId] = snapshot
		itemIds = append(itemIds, snapshot.CueItemId, snapshot.BreakItemId)
		productIds = append(productIds, snapshot.TipProductId, snapshot.ChalkProductId)
	}
	items, err := svcCtx.GearModel.FindItemsByIds(itemIds)
	if err != nil {
		return nil, err
	}
	itemList := make([]model.UserGearItem, 0, len(items))
	for _, item := range items {
		itemList = append(itemList, item)
	}
	catalog, err := loadGearCatalog(svcCtx.GearModel, itemList, productIds...)
	if err != nil {
		return nil, err
	}

	side := func(userId int64) types.GearMatchSide {
		return catalog.matchSide(userId, viewerId, byUser, items)
	}
	deadline := matchEditDeadline(match)
	resp := &types.GetGearMatchResp{
		Success:       true,
		Message:       "ok",
		MatchId:       match.Id,
		GameType:      match.GameType,
		SupportsBreak: model.GearGameTypeRecordsBreak(match.GameType),
		Me:            side(viewerId),
		HasOpponent:   opponentId > 0,
		Editable:      now.Before(deadline),
		EditDeadline:  deadline.Format(time.RFC3339),
	}
	if opponentId > 0 {
		resp.Opponent = side(opponentId)
	}
	return resp, nil
}

// matchSide 组装一位选手的本场用杆：球杆与冲杆是装备，皮头与巧克是记录当时的型号。
func (c *gearCatalog) matchSide(userId, viewerId int64, snapshots map[int64]model.MatchGearSnapshot, items map[int64]model.UserGearItem) types.GearMatchSide {
	side := types.GearMatchSide{UserId: userId}
	snapshot, ok := snapshots[userId]
	if !ok {
		return side
	}
	side.Recorded = true
	side.Source = snapshot.Source
	side.TipProduct = c.productInfoByID(snapshot.TipProductId)
	side.ChalkProduct = c.productInfoByID(snapshot.ChalkProductId)

	if item, found := items[snapshot.CueItemId]; found {
		info := c.itemInfo(item, item.UserId == viewerId)
		// 展示记录当时的皮头与巧克，而不是这根杆现在装的。
		info.TipProduct, info.ChalkProduct = side.TipProduct, side.ChalkProduct
		side.Cue = &info
	}
	if item, found := items[snapshot.BreakItemId]; found {
		info := c.itemInfo(item, item.UserId == viewerId)
		side.BreakCue = &info
	}
	return side
}
