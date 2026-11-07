package public_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/handler"
	"chasing_points/internal/middleware"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg"
	"chasing_points/internal/svc"

	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRegisteredPublicRoutesUseOptionalJWTMiddleware(t *testing.T) {
	const secret = "registered-public-route-secret"
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserRanking{}, &model.RankConfig{}, &model.Friend{}, &model.Match{}, &model.MatchRound{}); err != nil {
		t.Fatalf("migrate public route schema: %v", err)
	}
	if err := db.Create(&[]model.User{{Id: 1, Nickname: "viewer", Status: 1}, {Id: 2, Nickname: "friend", Status: 1}}).Error; err != nil {
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
	cfg.RestConf.ServiceConf = service.ServiceConf{Name: "public-route-test", Mode: service.TestMode}
	cfg.RestConf.Host = "127.0.0.1"
	cfg.RestConf.Port = 1
	svcCtx := &svc.ServiceContext{
		DB:           db,
		UserModel:    model.NewUserModel(db),
		MatchModel:   model.NewMatchModel(db),
		RankingModel: model.NewRankingModel(db),
		FriendModel:  model.NewFriendModel(db),
		Config:       cfg,
	}

	server, err := rest.NewServer(cfg.RestConf)
	if err != nil {
		t.Fatalf("create rest server: %v", err)
	}
	t.Cleanup(server.Stop)
	handler.RegisterHandlers(server, svcCtx)
	server.Use(middleware.RequestContextMiddleware)
	server.Use(middleware.RequestObservabilityMiddleware)
	server.Use(middleware.NewOptionalPublicJWTMiddleware(secret).Handle)
	server.Use(middleware.NewActiveUserSessionMiddleware(svcCtx.UserModel).Handle)
	serverless, err := rest.NewServerless(server)
	if err != nil {
		t.Fatalf("build registered routes: %v", err)
	}
	httpServer := httptest.NewServer(http.HandlerFunc(serverless.Serve))
	t.Cleanup(httpServer.Close)

	accessToken, err := pkg.GenerateTypedToken(1, secret, 60, pkg.AccessTokenType)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	refreshToken, err := pkg.GenerateTypedToken(1, secret, 60, pkg.RefreshTokenType)
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	t.Run("anonymous leaderboard remains public", func(t *testing.T) {
		response := getRegisteredPublicRoute(t, httpServer.URL+"/api/public/rank/leaderboard?game_type=3", "")
		defer response.Body.Close()
		var payload struct {
			Success   bool        `json:"success"`
			MyRanking interface{} `json:"my_ranking"`
		}
		decodeRegisteredPublicResponse(t, response, &payload)
		if response.StatusCode != http.StatusOK || !payload.Success || payload.MyRanking != nil {
			t.Fatalf("anonymous public route mismatch: status=%d payload=%+v", response.StatusCode, payload)
		}
	})

	t.Run("access token personalizes registered routes", func(t *testing.T) {
		response := getRegisteredPublicRoute(t, httpServer.URL+"/api/public/rank/leaderboard-summary?game_type=3", accessToken)
		defer response.Body.Close()
		var payload struct {
			Success   bool `json:"success"`
			MyRanking *struct {
				UserId int64 `json:"user_id"`
			} `json:"my_ranking"`
		}
		decodeRegisteredPublicResponse(t, response, &payload)
		if response.StatusCode != http.StatusOK || !payload.Success || payload.MyRanking == nil || payload.MyRanking.UserId != 1 {
			t.Fatalf("registered summary did not receive viewer: status=%d payload=%+v", response.StatusCode, payload)
		}

		friendsResponse := getRegisteredPublicRoute(t, httpServer.URL+"/api/public/matches?scope=friends", accessToken)
		defer friendsResponse.Body.Close()
		var friendsPayload struct {
			Success bool  `json:"success"`
			Total   int64 `json:"total"`
		}
		decodeRegisteredPublicResponse(t, friendsResponse, &friendsPayload)
		if friendsResponse.StatusCode != http.StatusOK || !friendsPayload.Success || friendsPayload.Total != 1 {
			t.Fatalf("registered friends scope mismatch: status=%d payload=%+v", friendsResponse.StatusCode, friendsPayload)
		}
	})

	for name, token := range map[string]string{
		"malformed":     "not-a-jwt",
		"refresh token": refreshToken,
	} {
		t.Run(name+" is rejected", func(t *testing.T) {
			response := getRegisteredPublicRoute(t, httpServer.URL+"/api/public/rank/leaderboard?game_type=3", token)
			defer response.Body.Close()
			if response.StatusCode != http.StatusUnauthorized {
				t.Fatalf("invalid token status=%d, want 401", response.StatusCode)
			}
		})
	}
}

func getRegisteredPublicRoute(t *testing.T, url, token string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build public request: %v", err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("execute public request: %v", err)
	}
	return response
}

func decodeRegisteredPublicResponse(t *testing.T, response *http.Response, target interface{}) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode public response: %v", err)
	}
}
