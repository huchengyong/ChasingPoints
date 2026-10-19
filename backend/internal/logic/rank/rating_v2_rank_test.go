package rank

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/types"
)

// rankV2Config 把 V2 起点设为本月初（offsetMonths=0，已生效）或之后某个月初（未生效）。
func rankV2Config(offsetMonths int) config.Config {
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(shanghai)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, shanghai).AddDate(0, offsetMonths, 0)
	return config.Config{
		SeasonLifecycle: config.SeasonLifecycleConfig{Enabled: true, AnchorDate: start.AddDate(0, -6, 0).Format(time.DateOnly), InitialNumber: 1, CycleMonths: 1, Timezone: "Asia/Shanghai"},
		Ranking:         config.RankingConfig{RatingV2StartDate: start.Format(time.DateOnly)},
	}
}

func TestRankInfoUsesV2DefaultsAndPlacementAfterStart(t *testing.T) {
	svcCtx := newRankLogicTestSvc(t)
	svcCtx.Config = rankV2Config(0)
	seedRankLogicRanking(t, svcCtx, &model.UserRanking{UserId: 1002, GameType: 3, RankScore: 1320, RankLevel: 3, RatingVersion: model.RatingVersionV2, RatedMatches: 4})
	seedRankLogicRanking(t, svcCtx, &model.UserRanking{UserId: 1003, GameType: 3, RankScore: 2600, RankLevel: 6})

	fresh, err := NewGetUserRankInfoLogic(rankLogicCtx(1001), svcCtx).GetUserRankInfo(&types.GetUserRankInfoReq{GameType: 3})
	if err != nil || !fresh.Success || fresh.RankInfo.RankScore != ranking.InitialRating || fresh.RankInfo.Level != 3 || fresh.RankInfo.PlacementRemaining != ranking.PlacementMatches {
		t.Fatalf("user without ranking must see the V2 initial rating in placement: resp=%+v err=%v", fresh.RankInfo, err)
	}
	placing, err := NewGetUserRankInfoLogic(rankLogicCtx(1002), svcCtx).GetUserRankInfo(&types.GetUserRankInfoReq{GameType: 3})
	if err != nil || placing.RankInfo.PlacementRemaining != 6 {
		t.Fatalf("placement must count V2 rated matches: resp=%+v err=%v", placing.RankInfo, err)
	}
	pending, err := NewGetUserRankInfoLogic(rankLogicCtx(1003), svcCtx).GetUserRankInfo(&types.GetUserRankInfoReq{GameType: 3})
	if err != nil || pending.RankInfo.PlacementRemaining != 0 {
		t.Fatalf("rows not yet switched must not report placement: resp=%+v err=%v", pending.RankInfo, err)
	}
	all, err := NewGetUserRankInfosLogic(rankLogicCtx(1001), svcCtx).GetUserRankInfos()
	if err != nil || !all.Success || len(all.RankInfos) != len(model.SupportedRankingGameTypes()) {
		t.Fatalf("get all rank infos: resp=%+v err=%v", all, err)
	}
	for _, item := range all.RankInfos {
		if item.RankInfo.RankScore != ranking.InitialRating || item.RankInfo.PlacementRemaining != ranking.PlacementMatches {
			t.Fatalf("missing game types must synthesize the V2 default: %+v", item)
		}
	}
	var rows int64
	if err := svcCtx.DB.Model(&model.UserRanking{}).Where("user_id = ?", 1001).Count(&rows).Error; err != nil || rows != 0 {
		t.Fatalf("rank info reads must not create rows: rows=%d err=%v", rows, err)
	}
}

func TestRankInfoKeepsLegacyDefaultsBeforeStart(t *testing.T) {
	svcCtx := newRankLogicTestSvc(t)
	svcCtx.Config = rankV2Config(1)
	resp, err := NewGetUserRankInfoLogic(rankLogicCtx(1001), svcCtx).GetUserRankInfo(&types.GetUserRankInfoReq{GameType: 3})
	if err != nil || !resp.Success || resp.RankInfo.RankScore != 0 || resp.RankInfo.Level != 1 || resp.RankInfo.PlacementRemaining != 0 {
		t.Fatalf("legacy defaults must apply before the V2 start: resp=%+v err=%v", resp.RankInfo, err)
	}
}

func TestRatingRulesWorkerSwitchesOnlyAfterStart(t *testing.T) {
	svcCtx := newRankLogicTestSvc(t)
	if err := svcCtx.DB.AutoMigrate(&model.RankChangeLog{}, &model.RankRatingSeed{}); err != nil {
		t.Fatalf("prepare switch schema: %v", err)
	}
	seedRankLogicRanking(t, svcCtx, &model.UserRanking{UserId: 1001, GameType: 3, RankScore: 2600, RankLevel: 6})
	seedRankLogicRanking(t, svcCtx, &model.UserRanking{UserId: 1002, GameType: 4, RankScore: 40, RankLevel: 1})

	svcCtx.Config = rankV2Config(1)
	worker := NewRatingRulesWorker(svcCtx)
	if switched, err := worker.RunOnce(context.Background()); err != nil || switched != 0 {
		t.Fatalf("worker must wait for the V2 start: switched=%d err=%v", switched, err)
	}

	svcCtx.Config = rankV2Config(0)
	if switched, err := worker.RunOnce(context.Background()); err != nil || switched != 2 {
		t.Fatalf("worker must switch every legacy row after the start: switched=%d err=%v", switched, err)
	}
	if switched, err := worker.RunOnce(context.Background()); err != nil || switched != 0 {
		t.Fatalf("steady state must be a no-op: switched=%d err=%v", switched, err)
	}
	var row model.UserRanking
	if err := svcCtx.DB.Where("user_id = ? AND game_type = ?", 1001, 3).First(&row).Error; err != nil || row.RankScore != ranking.InitialRating || row.RatingVersion != model.RatingVersionV2 {
		t.Fatalf("switched row: %+v err=%v", row, err)
	}
}
