package main

import (
	"os"
	"strings"
	"testing"
)

func TestHonorWallAndRewardSummaryAPIContract(t *testing.T) {
	apiContent, err := os.ReadFile("chasing_points.api")
	if err != nil {
		t.Fatalf("read api contract: %v", err)
	}
	text := string(apiContent)

	requiredSnippets := []string{
		"type GetHonorWallReq",
		"UserId          int64 `form:\"user_id,optional\"`",
		"GameType        int   `form:\"game_type,optional,default=3\"`",
		"HistorySeasonId int64 `form:\"history_season_id,optional\"`",
		"type GetHonorWallResp",
		"ViewerScope",
		"RecentHonors",
		"CareerAchievements",
		"CurrentSeason",
		"type GetMatchRewardSummaryReq",
		"MatchId int64 `form:\"match_id\"`",
		"type GetMatchRewardSummaryResp",
		"Status  string",
		"List    []MatchRewardItem",
		"get /honor-wall (GetHonorWallReq) returns (GetHonorWallResp)",
		"get /reward-summary (GetMatchRewardSummaryReq) returns (GetMatchRewardSummaryResp)",
		"get /list (GetAchievementListReq) returns (GetAchievementListResp)",
		"get /titles returns (GetUserTitlesResp)",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected api contract to contain %q", snippet)
		}
	}

	assertJWTServerBlock(t, text, "prefix: /api/achievement", "get /honor-wall")
	assertJWTServerBlock(t, text, "prefix: /api/match", "get /reward-summary")
}

func TestGeneratedRoutesContainHonorWallAndRewardSummary(t *testing.T) {
	content, err := os.ReadFile("internal/handler/routes.go")
	if err != nil {
		t.Fatalf("read generated routes: %v", err)
	}
	text := string(content)
	requiredSnippets := []string{
		"Path:    \"/honor-wall\"",
		"Handler: achievement.GetHonorWallHandler(serverCtx)",
		"rest.WithPrefix(\"/api/achievement\")",
		"Path:    \"/reward-summary\"",
		"Handler: match.GetMatchRewardSummaryHandler(serverCtx)",
		"rest.WithPrefix(\"/api/match\")",
		"rest.WithJwt(serverCtx.Config.Auth.AccessSecret)",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("expected generated routes to contain %q", snippet)
		}
	}
}

func assertJWTServerBlock(t *testing.T, text, prefix, route string) {
	t.Helper()
	prefixIndex := strings.Index(text, prefix)
	if prefixIndex < 0 {
		t.Fatalf("missing server prefix %s", prefix)
	}
	serverStart := strings.LastIndex(text[:prefixIndex], "@server (")
	if serverStart < 0 {
		t.Fatalf("missing server block for %s", prefix)
	}
	blockEndRelative := strings.Index(text[prefixIndex:], "}\n")
	if blockEndRelative < 0 {
		t.Fatalf("missing service block end for %s", prefix)
	}
	block := text[serverStart : prefixIndex+blockEndRelative+2]
	if !strings.Contains(block, "jwt:    Auth") {
		t.Fatalf("server block %s must require JWT", prefix)
	}
	if !strings.Contains(block, route) {
		t.Fatalf("server block %s missing route %s", prefix, route)
	}
}
