package match

import (
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"
)

func hideTargetMatchRecord(t *testing.T, svcCtx *svc.ServiceContext, userID int64) {
	t.Helper()
	if err := svcCtx.DB.Model(&model.User{}).Where("id = ?", userID).Update("hide_match_record", true).Error; err != nil {
		t.Fatalf("hide match record: %v", err)
	}
}

func TestH2HEndpointsRespectTargetHideMatchRecord(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)
	hideTargetMatchRecord(t, svcCtx, 202)

	stats, err := NewGetH2HStatsLogic(matchLogicCtx(101), svcCtx).GetH2HStats(&types.H2HStatsReq{TargetUserId: 202, OpponentId: 404})
	if err != nil || !stats.Success || !stats.Hidden || stats.Stats != nil || stats.Opponent != nil || stats.Message != "对方已隐藏战绩" {
		t.Fatalf("hidden target stats: resp=%#v err=%v", stats, err)
	}

	history, err := NewGetH2HHistoryLogic(matchLogicCtx(101), svcCtx).GetH2HHistory(&types.H2HHistoryReq{TargetUserId: 202, OpponentId: 404, Page: 1, PageSize: 20})
	if err != nil || !history.Success || !history.Hidden || history.Total != 0 || len(history.List) != 0 || history.Message != "对方已隐藏战绩" {
		t.Fatalf("hidden target history: resp=%#v err=%v", history, err)
	}

	overview, err := NewGetH2HOverviewLogic(matchLogicCtx(101), svcCtx).GetH2HOverview(&types.H2HOverviewReq{TargetUserId: 202, OpponentId: 404})
	if err != nil || !overview.Success || !overview.Hidden || overview.Stats != nil || overview.Opponent != nil || overview.Total != 0 || len(overview.List) != 0 || overview.Message != "对方已隐藏战绩" {
		t.Fatalf("hidden target overview: resp=%#v err=%v", overview, err)
	}
}

func TestH2HEndpointsUnchangedWhenTargetDoesNotHide(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)

	stats, err := NewGetH2HStatsLogic(matchLogicCtx(101), svcCtx).GetH2HStats(&types.H2HStatsReq{TargetUserId: 202, OpponentId: 404})
	if err != nil || !stats.Success || stats.Hidden || stats.Stats == nil || stats.Stats.TotalMatches != 2 {
		t.Fatalf("visible target stats: resp=%#v err=%v", stats, err)
	}
	overview, err := NewGetH2HOverviewLogic(matchLogicCtx(101), svcCtx).GetH2HOverview(&types.H2HOverviewReq{TargetUserId: 202, OpponentId: 404})
	if err != nil || !overview.Success || overview.Hidden || overview.Stats == nil {
		t.Fatalf("visible target overview: resp=%#v err=%v", overview, err)
	}
}

func TestLegacyH2HEndpointsRespectTargetHideMatchRecord(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	svcCtx.Config = config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "disabled"}}
	seedTargetH2HFixtures(t, svcCtx)
	hideTargetMatchRecord(t, svcCtx, 202)

	stats, err := NewGetH2HStatsLogic(matchLogicCtx(101), svcCtx).GetH2HStats(&types.H2HStatsReq{TargetUserId: 202, OpponentId: 404})
	if err != nil || !stats.Success || !stats.Hidden || stats.Stats != nil {
		t.Fatalf("legacy hidden target stats: resp=%#v err=%v", stats, err)
	}
	history, err := NewGetH2HHistoryLogic(matchLogicCtx(101), svcCtx).GetH2HHistory(&types.H2HHistoryReq{TargetUserId: 202, OpponentId: 404, Page: 1, PageSize: 20})
	if err != nil || !history.Success || !history.Hidden || history.Total != 0 || len(history.List) != 0 {
		t.Fatalf("legacy hidden target history: resp=%#v err=%v", history, err)
	}
	overview, err := NewGetH2HOverviewLogic(matchLogicCtx(101), svcCtx).GetH2HOverview(&types.H2HOverviewReq{TargetUserId: 202, OpponentId: 404})
	if err != nil || !overview.Success || !overview.Hidden || overview.Stats != nil || len(overview.List) != 0 {
		t.Fatalf("legacy hidden target overview: resp=%#v err=%v", overview, err)
	}
}

func TestH2HHiddenDoesNotBypassFriendCheck(t *testing.T) {
	svcCtx := newMatchLogicTestSvc(t)
	seedTargetH2HFixtures(t, svcCtx)
	hideTargetMatchRecord(t, svcCtx, 909)

	resp, err := NewGetH2HStatsLogic(matchLogicCtx(101), svcCtx).GetH2HStats(&types.H2HStatsReq{TargetUserId: 909, OpponentId: 404})
	if err != nil || resp.Success || resp.Hidden || resp.Message != "仅可查看好友的对方战绩" {
		t.Fatalf("non-friend target must still be rejected: resp=%#v err=%v", resp, err)
	}
}
