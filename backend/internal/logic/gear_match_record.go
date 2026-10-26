package logic

import (
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
)

// RecordMatchGearSnapshots 在对局完成后，按两位注册选手在该球种的主力装备设置，各写一条对局用杆记录。
//
// 记录的是完成时的组合：球杆与冲杆存装备 ID，皮头与巧克存这根球杆当时的型号，之后换皮头不影响历史。
// 固定最多 3 条 SQL（主力设置、装备、批量写入），没有任何主力设置时只有 1 条。
// 同一对局同一选手已有记录时不覆盖，重复调用与选手手动修改都是安全的。
// 调用方只记录错误日志，不得因为它失败而影响对局完成与结算。
func RecordMatchGearSnapshots(svcCtx *svc.ServiceContext, match *model.Match) error {
	if svcCtx == nil || svcCtx.GearModel == nil || match == nil || match.Id <= 0 {
		return nil
	}
	userIds := []int64{match.UserId}
	if match.OpponentId != nil && *match.OpponentId > 0 && *match.OpponentId != match.UserId {
		userIds = append(userIds, *match.OpponentId)
	}

	defaults, err := svcCtx.GearModel.ListDefaultsForUsers(userIds, match.GameType)
	if err != nil || len(defaults) == 0 {
		return err
	}
	itemIds := make([]int64, 0, len(defaults))
	for _, d := range defaults {
		itemIds = append(itemIds, d.ItemId)
	}
	items, err := svcCtx.GearModel.FindItemsByIds(itemIds)
	if err != nil {
		return err
	}

	return svcCtx.GearModel.InsertSnapshotsIgnore(buildMatchGearSnapshots(match.Id, match.GameType, defaults, items))
}

// buildMatchGearSnapshots 把主力设置转成记录行。只认本人仍在用的装备，且类别要与槽位匹配；
// 没有任何有效装备的选手不写记录。
func buildMatchGearSnapshots(matchId int64, gameType int, defaults []model.UserGearDefault, items map[int64]model.UserGearItem) []model.MatchGearSnapshot {
	rows := make(map[int64]*model.MatchGearSnapshot)
	var order []int64
	rowFor := func(userId int64) *model.MatchGearSnapshot {
		if row, ok := rows[userId]; ok {
			return row
		}
		row := &model.MatchGearSnapshot{MatchId: matchId, UserId: userId, Source: model.GearSnapshotSourceAuto}
		rows[userId] = row
		order = append(order, userId)
		return row
	}
	usable := func(userId int64, item model.UserGearItem, category string) bool {
		return item.Id > 0 && item.UserId == userId && item.Status == model.GearItemStatusActive && item.Category == category
	}

	for _, d := range defaults {
		item, ok := items[d.ItemId]
		if !ok {
			continue
		}
		switch d.Slot {
		case model.GearSlotCue:
			if usable(d.UserId, item, model.GearCategoryCue) {
				row := rowFor(d.UserId)
				row.CueItemId, row.TipProductId, row.ChalkProductId = item.Id, item.TipProductId, item.ChalkProductId
			}
		case model.GearSlotBreak:
			if model.GearGameTypeRecordsBreak(gameType) && usable(d.UserId, item, model.GearCategoryBreakJump) {
				rowFor(d.UserId).BreakItemId = item.Id
			}
		}
	}

	result := make([]model.MatchGearSnapshot, 0, len(order))
	for _, userId := range order {
		result = append(result, *rows[userId])
	}
	return result
}

// MatchCompletedAt 返回对局的完成时间，取值顺序与竞技读模型一致：completed_at、end_time、match_time。
func MatchCompletedAt(match *model.Match) time.Time {
	return resolveMatchCompletedAt(match)
}
