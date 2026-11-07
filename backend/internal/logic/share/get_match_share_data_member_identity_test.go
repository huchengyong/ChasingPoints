package share

import (
	"context"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newMemberIdentityShareEnv 准备一场已完成的斯诺克对局：1 号为创建者，2 号为对手。
// 默认成长门槛 10/40/120/300，所以成长值 45 对应 Lv3。
func newMemberIdentityShareEnv(t *testing.T, withGrowthProfiles bool) (*svc.ServiceContext, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	models := []any{
		&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{},
		&model.UserCompetitiveStats{}, &model.UserRanking{}, &model.MemberRightsConfig{},
	}
	if withGrowthProfiles {
		models = append(models, &model.MemberGrowthProfile{})
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("prepare member identity share schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	// 对手在斯诺克（1）的段位是 4，在中式八球（3）是 2：分享只取本场球种的段位。
	if err := db.Create(&[]model.UserRanking{
		{UserId: 1, GameType: 1, RankLevel: 6},
		{UserId: 2, GameType: 1, RankLevel: 4},
		{UserId: 2, GameType: 3, RankLevel: 2},
	}).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	opponentID := int64(2)
	result := 1
	completedAt := time.Date(2026, 10, 20, 19, 0, 0, 0, time.UTC)
	if err := db.Create(&model.Match{Id: 21, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 1, Status: 2, Result: &result, MyScore: 60, OpponentScore: 40, MatchTime: completedAt, CompletedAt: &completedAt}).Error; err != nil {
		t.Fatalf("seed match: %v", err)
	}
	if err := db.Create(&[]model.UserCompetitiveStats{
		{UserId: 1, GameType: 0, TotalMatches: 4, Wins: 2},
		{UserId: 1, GameType: 1, HighestBreak: 71},
		{UserId: 2, GameType: 0, TotalMatches: 4, Wins: 1},
		{UserId: 2, GameType: 1, HighestBreak: 63},
	}).Error; err != nil {
		t.Fatalf("seed profiles: %v", err)
	}
	return &svc.ServiceContext{
		DB:                       db,
		UserModel:                model.NewUserModel(db),
		MatchModel:               model.NewMatchModel(db),
		RankingModel:             model.NewRankingModel(db),
		CompetitiveReadModel:     model.NewCompetitiveReadModel(db),
		MemberGrowthProfileModel: model.NewMemberGrowthProfileModel(db),
		MemberRightsConfigModel:  model.NewMemberRightsConfigModel(db),
		Config:                   config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}, db
}

func makeShareMember(t *testing.T, db *gorm.DB, userID int64, growthPoints int) {
	t.Helper()
	if err := db.Model(&model.User{}).Where("id = ?", userID).Update("member_expires_at", time.Now().Add(24*time.Hour)).Error; err != nil {
		t.Fatalf("set member expiry: %v", err)
	}
	if growthPoints > 0 {
		if err := db.Create(&model.MemberGrowthProfile{UserId: userID, GrowthPoints: growthPoints, GrowthLevel: 1}).Error; err != nil {
			t.Fatalf("seed growth profile: %v", err)
		}
	}
}

func shareDataFor(t *testing.T, svcCtx *svc.ServiceContext, viewerID int64) *types.MatchShareData {
	t.Helper()
	resp, err := NewGetMatchShareDataLogic(context.WithValue(context.Background(), "user_id", viewerID), svcCtx).GetMatchShareData(&types.GetMatchShareDataReq{MatchId: 21})
	if err != nil || !resp.Success || resp.Data == nil {
		t.Fatalf("get share data for user %d: resp=%#v err=%v", viewerID, resp, err)
	}
	return resp.Data
}

func TestMatchShareDataReturnsMemberIdentityFromViewerPerspective(t *testing.T) {
	svcCtx, db := newMemberIdentityShareEnv(t, true)
	makeShareMember(t, db, 2, 45)

	asPlayer1 := shareDataFor(t, svcCtx, 1)
	if asPlayer1.MyVipLevel != 0 || asPlayer1.MyRankLevel != 0 {
		t.Fatalf("free sharer must not carry vip identity: %+v", asPlayer1)
	}
	if asPlayer1.OpponentVipLevel != 3 || asPlayer1.OpponentRankLevel != 4 {
		t.Fatalf("member opponent must be visible on a free user's poster data: vip=%d rank=%d", asPlayer1.OpponentVipLevel, asPlayer1.OpponentRankLevel)
	}

	asPlayer2 := shareDataFor(t, svcCtx, 2)
	if asPlayer2.MyVipLevel != 3 || asPlayer2.MyRankLevel != 4 || asPlayer2.OpponentVipLevel != 0 || asPlayer2.OpponentRankLevel != 0 {
		t.Fatalf("perspective must swap for the opponent viewer: %+v", asPlayer2)
	}
}

func TestMatchShareDataHasNoIdentityForFreeUsers(t *testing.T) {
	svcCtx, _ := newMemberIdentityShareEnv(t, true)

	data := shareDataFor(t, svcCtx, 1)
	if data.MyVipLevel != 0 || data.OpponentVipLevel != 0 || data.MyRankLevel != 0 || data.OpponentRankLevel != 0 {
		t.Fatalf("free users must carry no vip level or frame rank level: %+v", data)
	}
}

func TestMatchShareDataStillSucceedsWhenMemberIdentityLookupFails(t *testing.T) {
	// 不迁移成长档案表：有效会员的成长值查询必然失败。
	svcCtx, db := newMemberIdentityShareEnv(t, false)
	makeShareMember(t, db, 2, 0)

	data := shareDataFor(t, svcCtx, 1)
	if data.MyScore != 60 || data.OpponentScore != 40 || data.OpponentName != "乙" {
		t.Fatalf("share data must still be returned: %+v", data)
	}
	if data.OpponentVipLevel != 0 || data.OpponentRankLevel != 0 {
		t.Fatalf("failed identity lookup must degrade to non-member: %+v", data)
	}
}
