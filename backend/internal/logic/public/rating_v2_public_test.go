package public

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/ranking"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func publicV2Config(offsetMonths int) config.Config {
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(shanghai)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, shanghai).AddDate(0, offsetMonths, 0)
	return config.Config{
		SeasonLifecycle: config.SeasonLifecycleConfig{Enabled: true, AnchorDate: start.AddDate(0, -6, 0).Format(time.DateOnly), InitialNumber: 1, CycleMonths: 1, Timezone: "Asia/Shanghai"},
		Ranking:         config.RankingConfig{RatingV2StartDate: start.Format(time.DateOnly)},
	}
}

func TestRankConfigsReportRulesVersionOutsideStaticCache(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.RankConfig{}); err != nil {
		t.Fatalf("prepare rank config schema: %v", err)
	}
	if err := db.Create(&model.RankConfig{Level: 1, Name: "青铜球手"}).Error; err != nil {
		t.Fatalf("seed config: %v", err)
	}
	svcCtx := &svc.ServiceContext{RankingModel: model.NewRankingModel(db), Config: publicV2Config(1)}
	scheduled, err := NewGetRankConfigsLogic(context.Background(), svcCtx).GetRankConfigs()
	if err != nil || !scheduled.Success || scheduled.RulesVersion != ranking.RulesVersionLegacy || scheduled.RulesEffectiveAt == "" {
		t.Fatalf("scheduled V2 must keep legacy rules but announce the start: resp=%+v err=%v", scheduled, err)
	}
	svcCtx.Config = publicV2Config(0)
	active, err := NewGetRankConfigsLogic(context.Background(), svcCtx).GetRankConfigs()
	if err != nil || active.RulesVersion != ranking.RulesVersionV2 || active.Version != scheduled.Version {
		t.Fatalf("rules version must switch without changing the cached config version: resp=%+v err=%v", active, err)
	}
	svcCtx.Config = config.Config{}
	legacy, err := NewGetRankConfigsLogic(context.Background(), svcCtx).GetRankConfigs()
	if err != nil || legacy.RulesVersion != ranking.RulesVersionLegacy || legacy.RulesEffectiveAt != "" {
		t.Fatalf("unconfigured V2 must report legacy rules only: resp=%+v err=%v", legacy, err)
	}
}

func TestLeaderboardUsesV2EligibilityAfterStart(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRanking{}, &model.RankConfig{}); err != nil {
		t.Fatalf("prepare leaderboard schema: %v", err)
	}
	if err := db.Create(&[]model.RankConfig{{Level: 1, Name: "青铜球手"}, {Level: 3, Name: "黄金球手", MinScore: 1000}}).Error; err != nil {
		t.Fatalf("seed configs: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "已打V2"}, {Id: 2, Nickname: "旧规则"}, {Id: 3, Nickname: "未开打"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&[]model.UserRanking{
		{UserId: 1, GameType: 3, RankScore: 1300, RankLevel: 3, TotalWins: 2, RatingVersion: model.RatingVersionV2, RatedMatches: 3},
		{UserId: 2, GameType: 3, RankScore: 2600, RankLevel: 6, TotalWins: 90},
		{UserId: 3, GameType: 3, RankScore: 1250, RankLevel: 3, RatingVersion: model.RatingVersionV2},
	}).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	svcCtx := &svc.ServiceContext{RankingModel: model.NewRankingModel(db), UserModel: model.NewUserModel(db), Config: publicV2Config(0)}
	resp, err := NewGetLeaderboardLogic(context.WithValue(context.Background(), "user_id", int64(3)), svcCtx).GetLeaderboard(&types.GetLeaderboardReq{GameType: 3, Page: 1, PageSize: 20})
	if err != nil || !resp.Success || resp.Total != 1 || len(resp.TopThree) != 1 || resp.TopThree[0].UserId != 1 {
		t.Fatalf("V2 leaderboard must only list rows with V2 rated matches: resp=%+v err=%v", resp, err)
	}
	if resp.MyRanking == nil || resp.MyRanking.Rank != 0 || resp.MyRanking.RankLevel != 3 || resp.MyRanking.RankName != "黄金球手" {
		t.Fatalf("unranked viewer must see the initial V2 tier: %+v", resp.MyRanking)
	}
}
