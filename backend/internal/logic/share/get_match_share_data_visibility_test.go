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

// 海报据 visibility 选择二维码：私密练习对局不能被印成对局链接，旧数据缺省按排位公开归一。
func TestMatchShareDataReturnsNormalizedVisibility(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Match{}, &model.MatchRound{}, &model.MatchAction{}, &model.MatchAchievement{}, &model.UserCompetitiveStats{}); err != nil {
		t.Fatalf("prepare share schema: %v", err)
	}
	svcCtx := &svc.ServiceContext{
		DB:                   db,
		UserModel:            model.NewUserModel(db),
		MatchModel:           model.NewMatchModel(db),
		CompetitiveReadModel: model.NewCompetitiveReadModel(db),
		Config:               config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "enabled"}},
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "甲"}, {Id: 2, Nickname: "乙"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}

	opponentID := int64(2)
	result := 1
	completedAt := time.Date(2026, 10, 20, 19, 0, 0, 0, time.UTC)
	for _, testCase := range []struct {
		id         int64
		mode       string
		visibility string
		want       string
	}{
		{id: 31, mode: model.MatchModePractice, visibility: model.MatchVisibilityPrivate, want: model.MatchVisibilityPrivate},
		{id: 32, mode: model.MatchModeRanked, visibility: model.MatchVisibilityPublic, want: model.MatchVisibilityPublic},
		{id: 33, mode: model.MatchModeRanked, visibility: "", want: model.MatchVisibilityPublic},
	} {
		match := model.Match{
			Id: testCase.id, UserId: 1, OpponentId: &opponentID, OpponentName: "乙", GameType: 1,
			MatchMode: testCase.mode, Visibility: testCase.visibility, Status: 2, Result: &result,
			MyScore: 3, OpponentScore: 1, MatchTime: completedAt, CompletedAt: &completedAt,
		}
		if err := db.Create(&match).Error; err != nil {
			t.Fatalf("seed match %d: %v", testCase.id, err)
		}

		resp, err := NewGetMatchShareDataLogic(context.WithValue(context.Background(), "user_id", int64(1)), svcCtx).
			GetMatchShareData(&types.GetMatchShareDataReq{MatchId: testCase.id})
		if err != nil || !resp.Success || resp.Data == nil {
			t.Fatalf("get share data %d: resp=%#v err=%v", testCase.id, resp, err)
		}
		if resp.Data.Visibility != testCase.want {
			t.Fatalf("match %d visibility = %q, want %q", testCase.id, resp.Data.Visibility, testCase.want)
		}
	}
}
