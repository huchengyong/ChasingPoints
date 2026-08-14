package public

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/middleware"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg"
	"chasing_points/internal/svc"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPublicOptionalJWTTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRanking{}, &model.RankConfig{}, &model.Friend{}, &model.Match{}, &model.MatchRound{}); err != nil {
		t.Fatalf("migrate public HTTP handler schema: %v", err)
	}
	return db
}

func TestPublicHandlersUseOptionalJWTViewerFromHTTPMiddleware(t *testing.T) {
	const secret = "public-handler-optional-jwt-secret"
	db := newPublicOptionalJWTTestDB(t)
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "viewer"}, {Id: 2, Nickname: "friend"}}).Error; err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if err := db.Create(&model.RankConfig{Level: 1, Name: "新手"}).Error; err != nil {
		t.Fatalf("seed rank config: %v", err)
	}
	if err := db.Create(&[]model.UserRanking{{UserId: 1, GameType: 3, RankLevel: 1, RankScore: 1000, TotalWins: 1}, {UserId: 2, GameType: 3, RankLevel: 1, RankScore: 900, TotalWins: 1}}).Error; err != nil {
		t.Fatalf("seed rankings: %v", err)
	}
	if err := db.Create(&model.Friend{UserId: 1, FriendId: 2, Status: 1}).Error; err != nil {
		t.Fatalf("seed friend: %v", err)
	}
	opponentID := int64(1)
	if err := db.Create(&model.Match{Id: 9, UserId: 2, OpponentId: &opponentID, OpponentName: "viewer", GameType: 3, MatchMode: model.MatchModeRanked, Visibility: model.MatchVisibilityPublic, Status: 1, MatchTime: time.Now()}).Error; err != nil {
		t.Fatalf("seed public friend match: %v", err)
	}

	cfg := config.Config{}
	cfg.Auth.AccessSecret = secret
	serviceContext := &svc.ServiceContext{
		DB:           db,
		UserModel:    model.NewUserModel(db),
		MatchModel:   model.NewMatchModel(db),
		RankingModel: model.NewRankingModel(db),
		Config:       cfg,
	}
	optionalJWT := middleware.NewOptionalPublicJWTMiddleware(secret)
	token, err := pkg.GenerateTypedToken(1, secret, 60, pkg.AccessTokenType)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	leaderboard := optionalJWT.Handle(GetLeaderboardHandler(serviceContext))
	leaderboardRequest := httptest.NewRequest(http.MethodGet, "/api/public/rank/leaderboard?game_type=3", nil)
	leaderboardRequest.Header.Set("Authorization", "Bearer "+token)
	leaderboardResponse := httptest.NewRecorder()
	leaderboard(leaderboardResponse, leaderboardRequest)
	var leaderboardPayload struct {
		Success   bool `json:"success"`
		MyRanking *struct {
			UserId int64 `json:"user_id"`
		} `json:"my_ranking"`
	}
	if err := json.Unmarshal(leaderboardResponse.Body.Bytes(), &leaderboardPayload); err != nil {
		t.Fatalf("decode leaderboard response: %v body=%s", err, leaderboardResponse.Body.String())
	}
	if leaderboardResponse.Code != http.StatusOK || !leaderboardPayload.Success || leaderboardPayload.MyRanking == nil || leaderboardPayload.MyRanking.UserId != 1 {
		t.Fatalf("optional JWT must reach leaderboard logic: status=%d payload=%+v", leaderboardResponse.Code, leaderboardPayload)
	}

	summary := optionalJWT.Handle(GetLeaderboardSummaryHandler(serviceContext))
	summaryRequest := httptest.NewRequest(http.MethodGet, "/api/public/rank/leaderboard-summary?game_type=3", nil)
	summaryRequest.Header.Set("Authorization", "Bearer "+token)
	summaryResponse := httptest.NewRecorder()
	summary(summaryResponse, summaryRequest)
	var summaryPayload struct {
		Success   bool `json:"success"`
		MyRanking *struct {
			UserId int64 `json:"user_id"`
		} `json:"my_ranking"`
	}
	if err := json.Unmarshal(summaryResponse.Body.Bytes(), &summaryPayload); err != nil {
		t.Fatalf("decode leaderboard summary response: %v body=%s", err, summaryResponse.Body.String())
	}
	if summaryResponse.Code != http.StatusOK || !summaryPayload.Success || summaryPayload.MyRanking == nil || summaryPayload.MyRanking.UserId != 1 {
		t.Fatalf("optional JWT must reach leaderboard summary logic: status=%d payload=%+v", summaryResponse.Code, summaryPayload)
	}

	matches := optionalJWT.Handle(GetPublicMatchesHandler(serviceContext))
	friendRequest := httptest.NewRequest(http.MethodGet, "/api/public/matches?scope=friends", nil)
	friendRequest.Header.Set("Authorization", "Bearer "+token)
	friendResponse := httptest.NewRecorder()
	matches(friendResponse, friendRequest)
	var friendsPayload struct {
		Success bool  `json:"success"`
		Total   int64 `json:"total"`
	}
	if err := json.Unmarshal(friendResponse.Body.Bytes(), &friendsPayload); err != nil {
		t.Fatalf("decode friends response: %v body=%s", err, friendResponse.Body.String())
	}
	if friendResponse.Code != http.StatusOK || !friendsPayload.Success || friendsPayload.Total != 1 {
		t.Fatalf("optional JWT must reach friends scope logic: status=%d payload=%+v", friendResponse.Code, friendsPayload)
	}
}
