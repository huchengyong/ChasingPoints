package gear

import (
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/types"
)

func (e *gearEnv) getMatch(userId, matchId int64) (*types.GetGearMatchResp, int) {
	e.t.Helper()
	ctx, svcCtx, metrics := e.request(userId)
	resp, err := NewGetGearMatchLogic(ctx, svcCtx).GetGearMatch(&types.GetGearMatchReq{MatchId: matchId})
	if err != nil {
		e.t.Fatalf("get gear match returned an error: %v", err)
	}
	return resp, int(metrics.Snapshot().SQLCount)
}

func (e *gearEnv) updateMatch(userId int64, req types.UpdateGearMatchReq) *types.GetGearMatchResp {
	e.t.Helper()
	ctx, svcCtx, _ := e.request(userId)
	resp, err := NewUpdateGearMatchLogic(ctx, svcCtx).UpdateGearMatch(&req)
	if err != nil {
		e.t.Fatalf("update gear match returned an error: %v", err)
	}
	return resp
}

func (e *gearEnv) snapshotRows(matchId int64) []model.MatchGearSnapshot {
	e.t.Helper()
	rows, err := model.NewGearModel(e.db).FindSnapshotsByMatch(matchId)
	if err != nil {
		e.t.Fatalf("find snapshots: %v", err)
	}
	return rows
}

func (e *gearEnv) putSnapshot(matchId, userId, cue, tip, chalk, breakItem int64, source string) {
	e.t.Helper()
	row := model.MatchGearSnapshot{MatchId: matchId, UserId: userId, CueItemId: cue, TipProductId: tip, ChalkProductId: chalk, BreakItemId: breakItem, Source: source}
	if err := model.NewGearModel(e.db).InsertSnapshotsIgnore([]model.MatchGearSnapshot{row}); err != nil {
		e.t.Fatalf("seed snapshot: %v", err)
	}
}

func TestGetGearMatchShowsBothSidesFromEachViewersPerspective(t *testing.T) {
	env := newGearEnv(t, "")
	match := env.createMatch(500, 3, 1, 2, timeAgo(time.Hour))
	cue1 := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	break1 := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.breakCue})
	cue2 := env.mustCreate(2, types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "私人品牌", ModelName: "私人型号", CueStyle: model.GearCueStyleSnooker})
	env.putSnapshot(match.Id, 1, cue1, env.c.tipAlpha, env.c.chalkAlpha, break1, model.GearSnapshotSourceAuto)
	env.putSnapshot(match.Id, 2, cue2, 0, 0, 0, model.GearSnapshotSourceAuto)

	asOne, _ := env.getMatch(1, match.Id)
	if !asOne.Success || asOne.Me.UserId != 1 || asOne.Opponent.UserId != 2 || !asOne.HasOpponent || !asOne.SupportsBreak || asOne.GameType != 3 {
		t.Fatalf("player 1's view: %+v", asOne)
	}
	if asOne.Me.Cue == nil || asOne.Me.Cue.ModelName != "A-100" || asOne.Me.BreakCue == nil || asOne.Me.BreakCue.ModelName != "A-Break" {
		t.Fatalf("my cue and break cue: %+v", asOne.Me)
	}
	if asOne.Me.TipProduct == nil || asOne.Me.TipProduct.Name != "A-Tip" || asOne.Me.ChalkProduct == nil || asOne.Me.Source != model.GearSnapshotSourceAuto {
		t.Fatalf("my tip and chalk at the time of the match: %+v", asOne.Me)
	}
	if asOne.Opponent.Cue == nil || !asOne.Opponent.Cue.Unlisted || asOne.Opponent.Cue.BrandName != "其他品牌" || asOne.Opponent.Cue.ModelName != "未收录型号" {
		t.Fatalf("an opponent's typed names must not reach me: %+v", asOne.Opponent.Cue)
	}

	asTwo, _ := env.getMatch(2, match.Id)
	if asTwo.Me.UserId != 2 || asTwo.Opponent.UserId != 1 {
		t.Fatalf("the view must be oriented to the viewer: %+v", asTwo)
	}
	if asTwo.Me.Cue == nil || asTwo.Me.Cue.ModelName != "私人型号" || asTwo.Me.Cue.BrandName != "私人品牌" {
		t.Fatalf("I see my own typed names: %+v", asTwo.Me.Cue)
	}
	if asTwo.Opponent.Cue == nil || asTwo.Opponent.Cue.ModelName != "A-100" {
		t.Fatalf("catalog models are public: %+v", asTwo.Opponent.Cue)
	}
	if asTwo.Me.TipProduct != nil || asTwo.Me.BreakCue != nil {
		t.Fatalf("a record without tip or break cue shows none: %+v", asTwo.Me)
	}
}

func TestGetGearMatchKeepsTipFromTheTimeOfTheMatch(t *testing.T) {
	env := newGearEnv(t, "")
	match := env.createMatch(501, 3, 1, 2, timeAgo(time.Hour))
	cue := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool, TipProductId: env.c.tipAlpha})
	env.putSnapshot(match.Id, 1, cue, env.c.tipAlpha, 0, 0, model.GearSnapshotSourceAuto)

	// 对局之后换皮头、装上巧克：历史记录不受影响
	ctx, svcCtx, _ := env.request(1)
	if resp, _ := NewUpdateGearItemLogic(ctx, svcCtx).UpdateGearItem(&types.UpdateGearItemReq{ItemId: cue, UpdateTip: true, TipProductId: 0, UpdateChalk: true, ChalkProductId: env.c.chalkAlpha}); !resp.Success {
		t.Fatalf("change gear: %+v", resp)
	}
	view, _ := env.getMatch(1, match.Id)
	if view.Me.Cue == nil || view.Me.Cue.TipProduct == nil || view.Me.Cue.TipProduct.Name != "A-Tip" || view.Me.Cue.ChalkProduct != nil {
		t.Fatalf("the cue inside a match record must show the combination of that time: %+v", view.Me.Cue)
	}
}

func TestGetGearMatchResolvesDeletedAndLinkedItems(t *testing.T) {
	env := newGearEnv(t, "")
	match := env.createMatch(502, 3, 1, 2, timeAgo(time.Hour))
	unlisted := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, BrandName: "Brand X", ModelName: "Pro 1", CueStyle: model.GearCueStylePool})
	env.putSnapshot(match.Id, 1, unlisted, 0, 0, 0, model.GearSnapshotSourceAuto)

	if view, _ := env.getMatch(2, match.Id); view.Opponent.Cue == nil || view.Opponent.Cue.ModelName != "未收录型号" {
		t.Fatalf("before linking, others see the placeholder: %+v", view.Opponent.Cue)
	}
	// 后台把这个未收录型号关联到正式型号：历史对局双方都显示为该型号
	key := model.BuildGearCustomKey(model.GearCategoryCue, 0, "Brand X", "Pro 1")
	if _, _, err := model.NewGearModel(env.db).LinkPending(key, env.c.cueSnooker, model.GearPendingLinkBatchSize); err != nil {
		t.Fatalf("link pending: %v", err)
	}
	for _, viewer := range []int64{1, 2} {
		view, _ := env.getMatch(viewer, match.Id)
		cue := view.Me.Cue
		if viewer == 2 {
			cue = view.Opponent.Cue
		}
		if cue == nil || cue.Unlisted || cue.ModelName != "A-200" || cue.BrandName != "Alpha" {
			t.Fatalf("viewer %d should now see the real model: %+v", viewer, cue)
		}
	}

	// 删除这根杆（软删除）：历史对局仍然能解析它
	ctx, svcCtx, _ := env.request(1)
	if del, _ := NewDeleteGearItemLogic(ctx, svcCtx).DeleteGearItem(&types.DeleteGearItemReq{ItemId: unlisted}); !del.Success {
		t.Fatalf("delete: %+v", del)
	}
	view, _ := env.getMatch(1, match.Id)
	if view.Me.Cue == nil || view.Me.Cue.ModelName != "A-200" || view.Me.Cue.Status != model.GearItemStatusDeleted {
		t.Fatalf("a deleted item must still resolve for history: %+v", view.Me.Cue)
	}
}

func TestGetGearMatchHandlesUnrecordedUnregisteredAndRefusals(t *testing.T) {
	env := newGearEnv(t, "")
	done := env.createMatch(510, 3, 1, 2, timeAgo(time.Hour))
	soloDone := env.createMatch(511, 3, 1, 0, timeAgo(time.Hour))
	running := env.createMatch(512, 3, 1, 2, nil)
	refereed := env.createMatch(513, 3, 1, 2, timeAgo(time.Hour))
	if err := env.db.Model(&model.Match{}).Where("id = ?", refereed.Id).Update("referee_user_id", int64(3)).Error; err != nil {
		t.Fatalf("set referee: %v", err)
	}

	if view, _ := env.getMatch(1, done.Id); !view.Success || view.Me.Recorded || view.Opponent.Recorded || view.Me.Cue != nil {
		t.Fatalf("without records both sides read as unrecorded: %+v", view)
	}
	if view, _ := env.getMatch(1, soloDone.Id); !view.Success || view.HasOpponent || view.Opponent.UserId != 0 {
		t.Fatalf("an unregistered opponent has no side: %+v", view)
	}

	for name, tc := range map[string]struct {
		viewer int64
		match  int64
		reason string
	}{
		"referee":         {3, refereed.Id, reasonMatchNotParticipant},
		"stranger":        {4, done.Id, reasonMatchNotParticipant},
		"in progress":     {1, running.Id, reasonMatchNotCompleted},
		"missing match":   {1, 99999, reasonMatchNotFound},
		"zero match id":   {1, 0, reasonMatchNotFound},
		"in progress for": {2, running.Id, reasonMatchNotCompleted},
	} {
		view, _ := env.getMatch(tc.viewer, tc.match)
		if view.Success || view.Reason != tc.reason {
			t.Fatalf("%s: expected reason %s, got %+v", name, tc.reason, view)
		}
	}
	if view, _ := env.getMatch(-5, done.Id); view.Success {
		t.Fatalf("an admin token is not a player: %+v", view)
	}
	var count int64
	env.db.Model(&model.MatchGearSnapshot{}).Count(&count)
	if count != 0 {
		t.Fatalf("reading must never create records, found %d", count)
	}
}

func TestGetGearMatchUsesFiveQueriesAndReportsEditWindow(t *testing.T) {
	env := newGearEnv(t, "")
	match := env.createMatch(520, 3, 1, 2, timeAgo(2*time.Hour))
	cue1 := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	cue2 := env.mustCreate(2, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cueSnooker})
	env.putSnapshot(match.Id, 1, cue1, env.c.tipAlpha, env.c.chalkAlpha, 0, model.GearSnapshotSourceAuto)
	env.putSnapshot(match.Id, 2, cue2, env.c.tipAlpha, 0, 0, model.GearSnapshotSourceAuto)

	view, queries := env.getMatch(1, match.Id)
	if !view.Success || queries != 5 {
		t.Fatalf("expected match, records, items, products and brands (5 queries), got %d: %+v", queries, view)
	}
	deadline, err := time.Parse(time.RFC3339, view.EditDeadline)
	if err != nil || !view.Editable {
		t.Fatalf("a match finished two hours ago is editable until a RFC3339 deadline: %+v err=%v", view, err)
	}
	if want := match.CompletedAt.Add(7 * 24 * time.Hour); deadline.Sub(want).Abs() > time.Second {
		t.Fatalf("the deadline is completion plus 7 days: got %v want %v", deadline, want)
	}

	old := env.createMatch(521, 3, 1, 2, timeAgo(8*24*time.Hour))
	if view, _ := env.getMatch(1, old.Id); !view.Success || view.Editable {
		t.Fatalf("an 8-day-old match is readable but not editable: %+v", view)
	}
}

func TestUpdateGearMatchAddsAndReplacesTheCallersOwnRecord(t *testing.T) {
	env := newGearEnv(t, "")
	match := env.createMatch(530, 3, 1, 2, timeAgo(3*24*time.Hour))
	cue := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	otherCue := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cueSnooker})
	breakCue := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.breakJumpBoth})
	opponentCue := env.mustCreate(2, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	env.putSnapshot(match.Id, 2, opponentCue, env.c.tipAlpha, 0, 0, model.GearSnapshotSourceAuto)

	// 补记一场当时没有记录的对局
	added := env.updateMatch(1, types.UpdateGearMatchReq{MatchId: match.Id, CueItemId: cue, TipProductId: env.c.tipAlpha, ChalkProductId: env.c.chalkAlpha, BreakItemId: breakCue})
	if !added.Success || !added.Me.Recorded || added.Me.Source != model.GearSnapshotSourceManual || added.Me.Cue == nil || added.Me.BreakCue == nil {
		t.Fatalf("add a record: %+v", added)
	}
	// 改用另一根杆
	changed := env.updateMatch(1, types.UpdateGearMatchReq{MatchId: match.Id, CueItemId: otherCue, TipProductId: env.c.tipAlpha})
	if !changed.Success || changed.Me.Cue.ModelName != "A-200" || changed.Me.ChalkProduct != nil || changed.Me.BreakCue != nil {
		t.Fatalf("the update replaces the whole combination: %+v", changed)
	}
	// 对手的记录不受影响
	var opponentRow model.MatchGearSnapshot
	env.db.Where("match_id = ? AND user_id = ?", match.Id, 2).First(&opponentRow)
	if opponentRow.CueItemId != opponentCue || opponentRow.Source != model.GearSnapshotSourceAuto {
		t.Fatalf("the opponent's record must not change: %+v", opponentRow)
	}
	// 清空：删除本人记录
	cleared := env.updateMatch(1, types.UpdateGearMatchReq{MatchId: match.Id})
	if !cleared.Success || cleared.Me.Recorded {
		t.Fatalf("clearing every field removes the record: %+v", cleared)
	}
	if rows := env.snapshotRows(match.Id); len(rows) != 1 || rows[0].UserId != 2 {
		t.Fatalf("only the opponent's record remains: %+v", rows)
	}
}

func TestUpdateGearMatchRejectsInvalidChanges(t *testing.T) {
	env := newGearEnv(t, "")
	match := env.createMatch(540, 3, 1, 2, timeAgo(time.Hour))
	snooker := env.createMatch(541, 1, 1, 2, timeAgo(time.Hour))
	old := env.createMatch(542, 3, 1, 2, timeAgo(8*24*time.Hour))
	justInside := env.createMatch(543, 3, 1, 2, timeAgo(6*24*time.Hour+23*time.Hour))
	running := env.createMatch(544, 3, 1, 2, nil)

	cue := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	breakCue := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.breakCue})
	jumpOnly := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryBreakJump, ProductId: env.c.jumpCue})
	opponentCue := env.mustCreate(2, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	deleted := env.mustCreate(1, types.CreateGearItemReq{Category: model.GearCategoryCue, ProductId: env.c.cuePool})
	ctx, svcCtx, _ := env.request(1)
	if del, _ := NewDeleteGearItemLogic(ctx, svcCtx).DeleteGearItem(&types.DeleteGearItemReq{ItemId: deleted}); !del.Success {
		t.Fatalf("delete: %+v", del)
	}

	rejected := map[string]types.UpdateGearMatchReq{
		"opponent's cue":          {MatchId: match.Id, CueItemId: opponentCue},
		"deleted cue":             {MatchId: match.Id, CueItemId: deleted},
		"missing cue":             {MatchId: match.Id, CueItemId: 99999},
		"break cue as cue":        {MatchId: match.Id, CueItemId: breakCue},
		"cue as break cue":        {MatchId: match.Id, BreakItemId: cue},
		"jump-only as break cue":  {MatchId: match.Id, BreakItemId: jumpOnly},
		"break cue in snooker":    {MatchId: snooker.Id, BreakItemId: breakCue},
		"chalk as tip":            {MatchId: match.Id, TipProductId: env.c.chalkAlpha},
		"tip as chalk":            {MatchId: match.Id, ChalkProductId: env.c.tipAlpha},
		"unknown tip":             {MatchId: match.Id, TipProductId: 99999},
		"past the 7 day window":   {MatchId: old.Id, CueItemId: cue},
		"match still in progress": {MatchId: running.Id, CueItemId: cue},
		"unknown match":           {MatchId: 99999, CueItemId: cue},
	}
	for name, req := range rejected {
		if resp := env.updateMatch(1, req); resp.Success {
			t.Fatalf("%s should be rejected: %+v", name, resp)
		}
	}
	if resp := env.updateMatch(4, types.UpdateGearMatchReq{MatchId: match.Id, CueItemId: cue}); resp.Success || resp.Reason != reasonMatchNotParticipant {
		t.Fatalf("a stranger cannot edit: %+v", resp)
	}
	if resp := env.updateMatch(-1, types.UpdateGearMatchReq{MatchId: match.Id, CueItemId: cue}); resp.Success {
		t.Fatalf("an admin token cannot edit: %+v", resp)
	}
	if rows := env.snapshotRows(match.Id); len(rows) != 0 {
		t.Fatalf("rejected updates must not write anything: %+v", rows)
	}

	// 窗口内最后一刻仍可修改；停用的皮头型号也接受，用来保留原值
	if resp := env.updateMatch(1, types.UpdateGearMatchReq{MatchId: justInside.Id, CueItemId: cue}); !resp.Success {
		t.Fatalf("inside the window the edit is accepted: %+v", resp)
	}
	if resp := env.updateMatch(1, types.UpdateGearMatchReq{MatchId: match.Id, CueItemId: cue, TipProductId: env.c.tipOff}); !resp.Success {
		t.Fatalf("a disabled tip model may be kept on a historical record: %+v", resp)
	}
	// 退役的杆也可以补记到历史对局里
	ctx, svcCtx, _ = env.request(1)
	if resp, _ := NewUpdateGearItemLogic(ctx, svcCtx).UpdateGearItem(&types.UpdateGearItemReq{ItemId: cue, Status: model.GearItemStatusRetired}); !resp.Success {
		t.Fatalf("retire: %+v", resp)
	}
	if resp := env.updateMatch(1, types.UpdateGearMatchReq{MatchId: match.Id, CueItemId: cue}); !resp.Success {
		t.Fatalf("a retired cue is a valid record for a past match: %+v", resp)
	}
}
