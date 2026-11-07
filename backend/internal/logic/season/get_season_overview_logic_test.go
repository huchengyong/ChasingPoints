package season

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/observability"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeasonOverviewRejectsMissingAuthentication(t *testing.T) {
	resp, err := NewGetSeasonOverviewLogic(context.Background(), &svc.ServiceContext{}).GetSeasonOverview(&types.GetSeasonOverviewReq{})
	if err != nil || resp.Success || resp.Season != nil {
		t.Fatalf("unauthenticated season overview must not return data: resp=%#v err=%v", resp, err)
	}
}

func TestSeasonOverviewReturnsCurrentWindowRecordAndFirstLeaderboardPage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonRecord{}, &model.User{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare season overview schema: %v", err)
	}
	now := time.Now()
	if err := db.Create(&model.Season{Id: 1, Name: "当前赛季", StartDate: now.AddDate(0, -1, 0), EndDate: now.AddDate(0, 1, 0), Status: 1}).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "我"}, {Id: 2, Nickname: "榜首"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&[]model.SeasonRecord{{SeasonId: 1, UserId: 1, GameType: 3, EndRankScore: 1000, PeakRankScore: 1100, MatchesPlayed: 5, Wins: 3}, {SeasonId: 1, UserId: 2, GameType: 3, EndRankScore: 1200, PeakRankScore: 1200, MatchesPlayed: 6, Wins: 4}}).Error; err != nil {
		t.Fatalf("seed season records: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, Revision: 5}).Error; err != nil {
		t.Fatalf("seed revision: %v", err)
	}
	svcCtx := &svc.ServiceContext{SeasonModel: model.NewSeasonModel(db), SeasonRecordModel: model.NewSeasonRecordModel(db), CompetitiveReadModel: model.NewCompetitiveReadModel(db), Config: config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}}}
	resp, err := NewGetSeasonOverviewLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetSeasonOverview(&types.GetSeasonOverviewReq{GameType: 3})
	if err != nil || !resp.Success || resp.SeasonState != "active" || resp.Season == nil || resp.Record == nil {
		t.Fatalf("get season overview: resp=%#v err=%v", resp, err)
	}
	if resp.Record.MatchesPlayed != 5 || len(resp.Leaderboard) != 2 || resp.Leaderboard[0].UserId != 2 || resp.CompetitiveRevision != 5 {
		t.Fatalf("unexpected season overview: %+v", resp)
	}
	legacy, legacyErr := NewGetSeasonLeaderboardLogic(context.Background(), svcCtx).GetSeasonLeaderboard(&types.GetSeasonLeaderboardReq{SeasonId: 1, GameType: 3, Page: 1, PageSize: 20})
	if legacyErr != nil || !legacy.Success || len(legacy.List) != len(resp.Leaderboard) || legacy.List[0].UserId != resp.Leaderboard[0].UserId || legacy.List[0].RankScore != resp.Leaderboard[0].RankScore {
		t.Fatalf("overview and legacy season leaderboard must share records: overview=%+v legacy=%+v err=%v", resp.Leaderboard, legacy, legacyErr)
	}
	legacyPosition := 0
	for _, item := range legacy.List {
		if item.UserId == 1 {
			legacyPosition = item.Rank
		}
	}
	if resp.MyRank != 2 || resp.MyRank != legacyPosition || !resp.Availability["my_rank"] {
		t.Fatalf("my rank must match leaderboard position: my_rank=%d legacy=%d availability=%v", resp.MyRank, legacyPosition, resp.Availability)
	}

	newcomer, err := NewGetSeasonOverviewLogic(context.WithValue(context.Background(), "user_id", int64(3)), svcCtx).GetSeasonOverview(&types.GetSeasonOverviewReq{GameType: 3})
	if err != nil || !newcomer.Success || newcomer.Record != nil || newcomer.MyRank != 0 || !newcomer.Availability["record"] || !newcomer.Availability["my_rank"] {
		t.Fatalf("user without season record must get my_rank 0 without partial: resp=%#v err=%v", newcomer, err)
	}
}

func TestSeasonOverviewKeepsRecordWhenRankCountFails(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonRecord{}, &model.User{}); err != nil {
		t.Fatalf("prepare season overview schema: %v", err)
	}
	now := time.Now()
	if err := db.Create(&model.Season{Id: 1, Name: "当前赛季", StartDate: now.AddDate(0, -1, 0), EndDate: now.AddDate(0, 1, 0), Status: 1}).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	if err := db.Create(&model.SeasonRecord{SeasonId: 1, UserId: 1, GameType: 3, EndRankScore: 1000, MatchesPlayed: 2, Wins: 1}).Error; err != nil {
		t.Fatalf("seed season record: %v", err)
	}
	if err := db.Callback().Query().Before("gorm:query").Register("fail_season_rank_count", func(tx *gorm.DB) {
		if where, ok := tx.Statement.Clauses["WHERE"]; ok && strings.Contains(fmt.Sprintf("%v", where.Expression), "wins >") {
			_ = tx.AddError(errors.New("rank count failed"))
		}
	}); err != nil {
		t.Fatalf("register failing callback: %v", err)
	}
	svcCtx := &svc.ServiceContext{SeasonModel: model.NewSeasonModel(db), SeasonRecordModel: model.NewSeasonRecordModel(db)}
	resp, err := NewGetSeasonOverviewLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetSeasonOverview(&types.GetSeasonOverviewReq{GameType: 3})
	if err != nil || !resp.Success || resp.Record == nil || len(resp.Leaderboard) != 1 || !resp.Availability["record"] || !resp.Availability["leaderboard"] {
		t.Fatalf("rank count failure must keep record and leaderboard: resp=%#v err=%v", resp, err)
	}
	if resp.MyRank != 0 || resp.Availability["my_rank"] || !hasSeasonOverviewPartial(resp, "my_rank") {
		t.Fatalf("rank count failure must be an explicit partial: my_rank=%d availability=%v partial=%+v", resp.MyRank, resp.Availability, resp.PartialErrors)
	}
}

func hasSeasonOverviewPartial(resp *types.GetSeasonOverviewResp, scope string) bool {
	for _, item := range resp.PartialErrors {
		if item.Scope == scope {
			return true
		}
	}
	return false
}

func TestSeasonOverviewReadsSeasonRecordsWhenCompetitiveReadModelDisabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonRecord{}, &model.User{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare season overview schema: %v", err)
	}
	now := time.Now()
	if err := db.Create(&model.Season{Id: 1, Name: "当前赛季", StartDate: now.AddDate(0, -1, 0), EndDate: now.AddDate(0, 1, 0), Status: 1}).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	if err := db.Create(&model.User{Id: 1, Nickname: "我"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&model.SeasonRecord{SeasonId: 1, UserId: 1, GameType: 3, EndRankScore: 1000, PeakRankScore: 1100, MatchesPlayed: 5, Wins: 3}).Error; err != nil {
		t.Fatalf("seed season record: %v", err)
	}
	svcCtx := &svc.ServiceContext{SeasonModel: model.NewSeasonModel(db), SeasonRecordModel: model.NewSeasonRecordModel(db), CompetitiveReadModel: model.NewCompetitiveReadModel(db), Config: config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "disabled"}}}
	resp, err := NewGetSeasonOverviewLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).GetSeasonOverview(&types.GetSeasonOverviewReq{GameType: 3})
	if err != nil || !resp.Success || resp.Season == nil || resp.Record == nil || len(resp.Leaderboard) != 1 {
		t.Fatalf("disabled read mode must keep season records readable: resp=%#v err=%v", resp, err)
	}
	if resp.MyRank != 1 || !resp.Availability["my_rank"] || !resp.Availability["record"] || !resp.Availability["leaderboard"] || resp.Availability["competitive_revision"] || len(resp.PartialErrors) != 1 || resp.PartialErrors[0].Scope != "competitive_revision" {
		t.Fatalf("only competitive revision may be gated by read mode: availability=%v partial=%+v", resp.Availability, resp.PartialErrors)
	}
}

func TestSeasonOverviewKeepsSeasonAvailableWhenOptionalModelsAreMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}); err != nil {
		t.Fatalf("prepare season schema: %v", err)
	}
	now := time.Now()
	if err := db.Create(&model.Season{Id: 1, Name: "当前赛季", StartDate: now.AddDate(0, -1, 0), EndDate: now.AddDate(0, 1, 0), Status: 1}).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	resp, err := NewGetSeasonOverviewLogic(context.WithValue(context.Background(), "user_id", int64(1)), &svc.ServiceContext{SeasonModel: model.NewSeasonModel(db)}).GetSeasonOverview(&types.GetSeasonOverviewReq{GameType: 3})
	if err != nil || !resp.Success || resp.Season == nil || !resp.Availability["season"] || resp.Availability["record"] || resp.Availability["leaderboard"] || resp.Availability["my_rank"] || resp.Availability["competitive_revision"] {
		t.Fatalf("optional season blocks must fail independently: resp=%#v err=%v", resp, err)
	}
}

func TestSeasonOverviewStopsAfterUnavailableSeason(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}); err != nil {
		t.Fatalf("prepare season schema: %v", err)
	}
	resp, err := NewGetSeasonOverviewLogic(context.WithValue(context.Background(), "user_id", int64(1)), &svc.ServiceContext{SeasonModel: model.NewSeasonModel(db)}).GetSeasonOverview(&types.GetSeasonOverviewReq{})
	if err != nil || !resp.Success || resp.Season != nil || len(resp.Leaderboard) != 0 || resp.SeasonState != "not_started" {
		t.Fatalf("unavailable season must stop follow-up reads: resp=%#v err=%v", resp, err)
	}
}

func TestSeasonOverviewUsesFixedQueryCountForOneAndHundredRows(t *testing.T) {
	one := seasonOverviewQueryCount(t, 1)
	hundred := seasonOverviewQueryCount(t, 100)
	if one != 6 || hundred != 6 {
		t.Fatalf("season overview must use six fixed queries: one=%d hundred=%d", one, hundred)
	}
}

func seasonOverviewQueryCount(t *testing.T, rows int) int64 {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+fmt.Sprintf("-%d?mode=memory&cache=shared", rows)), &gorm.Config{Logger: observability.NewGormLogger(time.Hour)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Season{}, &model.SeasonRecord{}, &model.User{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare season overview query schema: %v", err)
	}
	now := time.Now()
	if err := db.Create(&model.Season{Id: 1, Name: "当前赛季", StartDate: now.AddDate(0, -1, 0), EndDate: now.AddDate(0, 1, 0), Status: 1}).Error; err != nil {
		t.Fatalf("seed season: %v", err)
	}
	users := make([]model.User, 0, rows)
	records := make([]model.SeasonRecord, 0, rows)
	for index := 1; index <= rows; index++ {
		users = append(users, model.User{Id: int64(index), Nickname: fmt.Sprintf("用户%d", index)})
		records = append(records, model.SeasonRecord{SeasonId: 1, UserId: int64(index), GameType: 3, EndRankScore: 1000 + index, PeakRankScore: 1000 + index, MatchesPlayed: 1, Wins: 1})
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("seed season records: %v", err)
	}
	if err := db.Create(&model.UserCompetitiveStats{UserId: 1, GameType: 0, Revision: 3}).Error; err != nil {
		t.Fatalf("seed revision: %v", err)
	}
	metrics := observability.NewRequestMetrics(time.Now())
	ctx := observability.WithRequestMetrics(context.WithValue(context.Background(), "user_id", int64(1)), metrics)
	requestDB := db.WithContext(ctx)
	resp, err := NewGetSeasonOverviewLogic(ctx, &svc.ServiceContext{SeasonModel: model.NewSeasonModel(requestDB), SeasonRecordModel: model.NewSeasonRecordModel(requestDB), CompetitiveReadModel: model.NewCompetitiveReadModel(requestDB), Config: config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}}}).GetSeasonOverview(&types.GetSeasonOverviewReq{GameType: 3})
	if err != nil || !resp.Success || resp.Season == nil {
		t.Fatalf("get season overview: resp=%#v err=%v", resp, err)
	}
	return metrics.Snapshot().SQLCount
}
