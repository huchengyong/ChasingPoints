package rank

import (
	"context"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/testsupport"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRankLogicTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.UserRanking{}, &model.RankConfig{}); err != nil {
		t.Fatalf("prepare rank schema: %v", err)
	}

	configs := []model.RankConfig{
		{Level: 1, Name: "青铜球手", Icon: "/static/images/ranks/rank_bronze.png", MinScore: 0},
		{Level: 2, Name: "白银球手", Icon: "/static/images/ranks/rank_silver.png", MinScore: 500},
		{Level: 3, Name: "黄金球手", Icon: "/static/images/ranks/rank_gold.png", MinScore: 1000},
		{Level: 4, Name: "铂金大师", Icon: "/static/images/ranks/rank_platinum.png", MinScore: 1500},
		{Level: 5, Name: "钻石", Icon: "/static/images/ranks/rank_diamond.png", MinScore: 2000},
		{Level: 6, Name: "王者", Icon: "/static/images/ranks/rank_king.png", MinScore: 2500},
	}
	if err := db.Create(&configs).Error; err != nil {
		t.Fatalf("seed rank configs: %v", err)
	}

	return &svc.ServiceContext{
		DB:           db,
		RankingModel: model.NewRankingModel(db),
	}
}

func rankLogicCtx(userID int64) context.Context {
	return context.WithValue(context.Background(), "user_id", userID)
}

func seedRankLogicRanking(t *testing.T, svcCtx *svc.ServiceContext, ranking *model.UserRanking) {
	t.Helper()
	if err := svcCtx.DB.Create(ranking).Error; err != nil {
		t.Fatalf("seed user ranking: %v", err)
	}
}

func TestGetUserRankInfoReturnsKingPromotionForDiamond(t *testing.T) {
	svcCtx := newRankLogicTestSvc(t)
	seedRankLogicRanking(t, svcCtx, &model.UserRanking{
		UserId:    1001,
		GameType:  3,
		RankScore: 2250,
		RankLevel: 5,
	})

	resp, err := NewGetUserRankInfoLogic(rankLogicCtx(1001), svcCtx).GetUserRankInfo(&types.GetUserRankInfoReq{GameType: 3})
	if err != nil {
		t.Fatalf("get diamond rank info: %v", err)
	}
	if !resp.Success || resp.RankInfo == nil {
		t.Fatalf("expected successful rank info, got %#v", resp)
	}
	if resp.RankInfo.Level != 5 || resp.RankInfo.Name != "钻石" {
		t.Fatalf("unexpected current rank: %#v", resp.RankInfo)
	}
	if resp.RankInfo.NextLevel != 6 || resp.RankInfo.NextName != "王者" || resp.RankInfo.NextScore != 2500 || resp.RankInfo.Progress != 50 {
		t.Fatalf("unexpected diamond promotion target: %#v", resp.RankInfo)
	}
}

func TestGetUserRankInfoReturnsKingAsMaxRank(t *testing.T) {
	svcCtx := newRankLogicTestSvc(t)
	seedRankLogicRanking(t, svcCtx, &model.UserRanking{
		UserId:    1002,
		GameType:  3,
		RankScore: 2600,
		RankLevel: 6,
	})

	resp, err := NewGetUserRankInfoLogic(rankLogicCtx(1002), svcCtx).GetUserRankInfo(&types.GetUserRankInfoReq{GameType: 3})
	if err != nil {
		t.Fatalf("get king rank info: %v", err)
	}
	if !resp.Success || resp.RankInfo == nil {
		t.Fatalf("expected successful rank info, got %#v", resp)
	}
	if resp.RankInfo.Level != 6 || resp.RankInfo.Name != "王者" || resp.RankInfo.NextLevel != 6 || resp.RankInfo.NextName != "王者" || resp.RankInfo.NextScore != 2500 || resp.RankInfo.Progress != 100 {
		t.Fatalf("unexpected max rank response: %#v", resp.RankInfo)
	}
}

func TestGetRankListReturnsSixConfigsInAscendingOrder(t *testing.T) {
	svcCtx := newRankLogicTestSvc(t)
	seedRankLogicRanking(t, svcCtx, &model.UserRanking{
		UserId:    1003,
		GameType:  3,
		RankScore: 2250,
		RankLevel: 5,
	})

	resp, err := NewGetRankListLogic(rankLogicCtx(1003), svcCtx).GetRankList(&types.GetRankListReq{GameType: 3})
	if err != nil {
		t.Fatalf("get rank list: %v", err)
	}
	if !resp.Success || len(resp.List) != 6 {
		t.Fatalf("expected six rank configs, got %#v", resp)
	}
	for index, item := range resp.List {
		if item.Level != index+1 {
			t.Fatalf("rank list index %d: expected level %d, got %d", index, index+1, item.Level)
		}
	}
	if resp.List[4].Name != "钻石" || resp.List[5].Name != "王者" || !resp.List[4].IsCurrent {
		t.Fatalf("unexpected diamond and king configs: %#v", resp.List[4:])
	}
}

func TestGetUserRankInfosReturnsFourOrderedGameTypesWithoutWritingMissingRows(t *testing.T) {
	svcCtx := newRankLogicTestSvc(t)
	seedRankLogicRanking(t, svcCtx, &model.UserRanking{UserId: 1004, GameType: 2, RankScore: 2500, RankLevel: 6})
	recorder := testsupport.NewSQLWriteRecorder()
	svcCtx.RankingModel = model.NewRankingModel(svcCtx.DB.Session(&gorm.Session{Logger: recorder}))

	resp, err := NewGetUserRankInfosLogic(rankLogicCtx(1004), svcCtx).GetUserRankInfos()
	if err != nil || !resp.Success {
		t.Fatalf("get aggregate rank info: resp=%#v err=%v", resp, err)
	}
	if len(resp.RankInfos) != 4 {
		t.Fatalf("expected four rank infos, got %#v", resp.RankInfos)
	}
	for index, item := range resp.RankInfos {
		if item.GameType != index+1 || item.RankInfo == nil {
			t.Fatalf("unexpected aggregate item %d: %#v", index, item)
		}
	}
	if resp.RankInfos[1].RankInfo.Progress != 100 || resp.RankInfos[1].RankInfo.NextName != "王者" {
		t.Fatalf("expected king max-rank semantics: %#v", resp.RankInfos[1])
	}
	var count int64
	if err := svcCtx.DB.Model(&model.UserRanking{}).Where("user_id = ?", 1004).Count(&count).Error; err != nil {
		t.Fatalf("count rank snapshots: %v", err)
	}
	if count != 1 {
		t.Fatalf("rank GET must not create missing rows, got %d", count)
	}
	if writes := recorder.Writes(); len(writes) != 0 {
		t.Fatalf("rank GET must not issue writes: %q", writes)
	}
}

func TestGetUserRankInfosRejectsMissingAuthentication(t *testing.T) {
	resp, err := NewGetUserRankInfosLogic(context.Background(), newRankLogicTestSvc(t)).GetUserRankInfos()
	if err != nil || resp.Success {
		t.Fatalf("expected authentication failure, resp=%#v err=%v", resp, err)
	}
}

func TestGetUserRankInfoKeepsLegacyZeroPromotionWhenNextConfigIsMissing(t *testing.T) {
	svcCtx := newRankLogicTestSvc(t)
	seedRankLogicRanking(t, svcCtx, &model.UserRanking{UserId: 1005, GameType: 3, RankScore: 100, RankLevel: 1})
	if err := svcCtx.DB.Where("level = ?", 2).Delete(&model.RankConfig{}).Error; err != nil {
		t.Fatalf("remove next rank config: %v", err)
	}

	resp, err := NewGetUserRankInfoLogic(rankLogicCtx(1005), svcCtx).GetUserRankInfo(&types.GetUserRankInfoReq{GameType: 3})
	if err != nil || !resp.Success || resp.RankInfo == nil {
		t.Fatalf("get rank info with missing next config: resp=%#v err=%v", resp, err)
	}
	if resp.RankInfo.NextLevel != 0 || resp.RankInfo.NextName != "" || resp.RankInfo.NextScore != 0 || resp.RankInfo.Progress != 0 {
		t.Fatalf("expected legacy empty promotion values, got %#v", resp.RankInfo)
	}
}

func TestGetUserRankInfosFailsWhenRankConfigIsIncomplete(t *testing.T) {
	svcCtx := newRankLogicTestSvc(t)
	if err := svcCtx.DB.Where("level = ?", 1).Delete(&model.RankConfig{}).Error; err != nil {
		t.Fatalf("remove current rank config: %v", err)
	}

	resp, err := NewGetUserRankInfosLogic(rankLogicCtx(1006), svcCtx).GetUserRankInfos()
	if err != nil || resp.Success {
		t.Fatalf("expected incomplete config failure, resp=%#v err=%v", resp, err)
	}
}
