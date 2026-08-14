package public

import (
	"context"
	"fmt"
	"testing"
	"time"

	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLeaderboardSummaryRejectsUnavailablePrimaryModel(t *testing.T) {
	resp, err := NewGetLeaderboardSummaryLogic(context.Background(), &svc.ServiceContext{}).GetLeaderboardSummary(&types.GetLeaderboardSummaryReq{GameType: 3})
	if err != nil || resp.Success || len(resp.TopThree) != 0 || resp.MyRanking != nil {
		t.Fatalf("leaderboard summary must fail safely without ranking model: resp=%#v err=%v", resp, err)
	}
}

func TestLeaderboardSummaryOnlyBuildsTopThreeAndViewerRank(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRanking{}, &model.RankConfig{}); err != nil {
		t.Fatalf("prepare leaderboard schema: %v", err)
	}
	if err := db.Create(&model.RankConfig{Level: 1, Name: "新手"}).Error; err != nil {
		t.Fatalf("seed config: %v", err)
	}
	users := []model.User{{Id: 1, Nickname: "我"}, {Id: 2, Nickname: "甲"}, {Id: 3, Nickname: "乙"}, {Id: 4, Nickname: "丙"}, {Id: 5, Nickname: "丁"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	rankings := make([]model.UserRanking, 0, len(users))
	for index, user := range users {
		rankings = append(rankings, model.UserRanking{UserId: user.Id, GameType: 3, RankLevel: 1, RankScore: 1000 - index*10, TotalWins: 1})
	}
	if err := db.Create(&rankings).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	svcCtx := &svc.ServiceContext{RankingModel: model.NewRankingModel(db)}
	resp, err := NewGetLeaderboardSummaryLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetLeaderboardSummary(&types.GetLeaderboardSummaryReq{GameType: 3})
	if err != nil || !resp.Success || len(resp.TopThree) != 3 || resp.MyRanking == nil || resp.MyRanking.UserId != 1 || resp.MyRanking.Rank != 1 || resp.Version == "" {
		t.Fatalf("get leaderboard summary: resp=%#v err=%v", resp, err)
	}
	anonymous, anonymousErr := NewGetLeaderboardSummaryLogic(context.Background(), svcCtx).GetLeaderboardSummary(&types.GetLeaderboardSummaryReq{GameType: 3})
	if anonymousErr != nil || !anonymous.Success || anonymous.MyRanking != nil || len(anonymous.TopThree) != 3 {
		t.Fatalf("public summary must not leak an authenticated viewer ranking: resp=%#v err=%v", anonymous, anonymousErr)
	}
	legacy, legacyErr := NewGetLeaderboardLogic(context.WithValue(context.Background(), "user_id", int64(1)), &svc.ServiceContext{RankingModel: model.NewRankingModel(db), UserModel: model.NewUserModel(db)}).GetLeaderboard(&types.GetLeaderboardReq{GameType: 3, Page: 1, PageSize: 20})
	if legacyErr != nil || !legacy.Success || len(legacy.TopThree) != len(resp.TopThree) || legacy.TopThree[0].UserId != resp.TopThree[0].UserId || legacy.MyRanking == nil || legacy.MyRanking.Rank != resp.MyRanking.Rank {
		t.Fatalf("summary and legacy leaderboard must share ranking results: summary=%+v legacy=%+v err=%v", resp, legacy, legacyErr)
	}
}

func TestLeaderboardSummaryUsesFixedQueryCountForOneAndHundredRows(t *testing.T) {
	one := leaderboardSummaryQueryCount(t, 1)
	hundred := leaderboardSummaryQueryCount(t, 100)
	if one != 5 || hundred != 5 {
		t.Fatalf("leaderboard summary must use five fixed queries: one=%d hundred=%d", one, hundred)
	}
}

func leaderboardSummaryQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRanking{}, &model.RankConfig{}); err != nil {
		t.Fatalf("prepare leaderboard query schema: %v", err)
	}
	if err := db.Create(&model.RankConfig{Level: 1, Name: "新手"}).Error; err != nil {
		t.Fatalf("seed rank config: %v", err)
	}
	users := make([]model.User, 0, rows)
	rankings := make([]model.UserRanking, 0, rows)
	for index := 1; index <= rows; index++ {
		users = append(users, model.User{Id: int64(index), Nickname: fmt.Sprintf("用户%d", index)})
		rankings = append(rankings, model.UserRanking{UserId: int64(index), GameType: 3, RankLevel: 1, RankScore: 10000 - index, TotalWins: 1})
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&rankings).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	requestDB := db.WithContext(ctx)
	resp, err := NewGetLeaderboardSummaryLogic(ctx, &svc.ServiceContext{RankingModel: model.NewRankingModel(requestDB)}).GetLeaderboardSummary(&types.GetLeaderboardSummaryReq{GameType: 3})
	if err != nil || !resp.Success || resp.MyRanking == nil {
		t.Fatalf("get leaderboard summary: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}

func TestRankConfigsHasStableVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.RankConfig{}); err != nil {
		t.Fatalf("prepare rank config schema: %v", err)
	}
	if err := db.Create(&[]model.RankConfig{{Level: 1, Name: "新手", MinScore: 0}, {Level: 2, Name: "进阶", MinScore: 100}}).Error; err != nil {
		t.Fatalf("seed configs: %v", err)
	}
	logic := NewGetRankConfigsLogic(context.Background(), &svc.ServiceContext{RankingModel: model.NewRankingModel(db)})
	first, err := logic.GetRankConfigs()
	if err != nil {
		t.Fatalf("get configs: %v", err)
	}
	second, err := logic.GetRankConfigs()
	if err != nil || !first.Success || len(first.List) != 2 || first.Version == "" || first.Version != second.Version {
		t.Fatalf("unexpected static configs: first=%#v second=%#v err=%v", first, second, err)
	}
}
