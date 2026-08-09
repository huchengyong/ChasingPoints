package main

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"chasing_points/internal/types"
)

func TestHonorWallAndRewardSummaryAPIContract(t *testing.T) {
	apiContent, err := os.ReadFile("chasing_points.api")
	if err != nil {
		t.Fatalf("read api contract: %v", err)
	}
	text := string(apiContent)

	requiredSnippets := []string{
		"type AchievementDef",
		"GameType        int    `json:\"game_type\"`",
		"RewardTitleName string `json:\"reward_title_name,optional\"`",
		"type GetHonorWallReq",
		"UserId          int64 `form:\"user_id,optional\"`",
		"GameType        int   `form:\"game_type,optional,default=3\"`",
		"HistorySeasonId int64 `form:\"history_season_id,optional\"`",
		"type HonorWallSummary",
		"CareerUnlocked    int `json:\"career_unlocked\"`",
		"UniversalUnlocked int `json:\"universal_unlocked\"`",
		"SpecialtyGameType int `json:\"specialty_game_type\"`",
		"SpecialtyUnlocked int `json:\"specialty_unlocked\"`",
		"SpecialtyTotal    int `json:\"specialty_total\"`",
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

func TestGeneratedAchievementPayloadContract(t *testing.T) {
	assertJSONTag := func(value any, fieldName, wantTag string) {
		t.Helper()
		field, ok := reflect.TypeOf(value).FieldByName(fieldName)
		if !ok {
			t.Fatalf("missing generated field %s", fieldName)
		}
		if got := field.Tag.Get("json"); got != wantTag {
			t.Fatalf("field %s json tag = %q, want %q", fieldName, got, wantTag)
		}
	}

	achievement := types.AchievementDef{}
	assertJSONTag(achievement, "Id", "id")
	assertJSONTag(achievement, "Progress", "progress")
	assertJSONTag(achievement, "GameType", "game_type")
	assertJSONTag(achievement, "RewardTitleName", "reward_title_name,optional")

	summary := types.HonorWallSummary{}
	assertJSONTag(summary, "CareerUnlocked", "career_unlocked")
	assertJSONTag(summary, "CareerTotal", "career_total")
	assertJSONTag(summary, "UniversalUnlocked", "universal_unlocked")
	assertJSONTag(summary, "UniversalTotal", "universal_total")
	assertJSONTag(summary, "SpecialtyGameType", "specialty_game_type")
	assertJSONTag(summary, "SpecialtyUnlocked", "specialty_unlocked")
	assertJSONTag(summary, "SpecialtyTotal", "specialty_total")
}

func TestBindPhoneRespAdditiveSessionContract(t *testing.T) {
	apiContent, err := os.ReadFile("chasing_points.api")
	if err != nil {
		t.Fatalf("read api contract: %v", err)
	}
	text := string(apiContent)

	requiredFields := []struct {
		name string
		tag  string
	}{
		{name: "Success", tag: "json:\"success\""},
		{name: "Message", tag: "json:\"message\""},
		{name: "MergedAccount", tag: "json:\"merged_account\""},
		{name: "AccessToken", tag: "json:\"access_token,optional\""},
		{name: "RefreshToken", tag: "json:\"refresh_token,optional\""},
		{name: "ExpiresIn", tag: "json:\"expires_in,optional\""},
		{name: "NeedBindPhone", tag: "json:\"need_bind_phone\""},
		{name: "UserInfo", tag: "json:\"user_info,optional\""},
	}
	block := text[strings.Index(text, "type BindPhoneResp"):]
	block = block[:strings.Index(block, "\n}")]
	for _, field := range requiredFields {
		if !strings.Contains(block, field.name) || !strings.Contains(block, field.tag) {
			t.Fatalf("expected bind phone contract to declare %s with %s", field.name, field.tag)
		}
	}

	resp := types.BindPhoneResp{}
	assertFieldJSONTag(t, resp, "AccessToken", "access_token,optional")
	assertFieldJSONTag(t, resp, "RefreshToken", "refresh_token,optional")
	assertFieldJSONTag(t, resp, "ExpiresIn", "expires_in,optional")
	assertFieldJSONTag(t, resp, "NeedBindPhone", "need_bind_phone")
	assertFieldJSONTag(t, resp, "UserInfo", "user_info,optional")
}

func assertFieldJSONTag(t *testing.T, value any, fieldName, wantTag string) {
	t.Helper()
	field, ok := reflect.TypeOf(value).FieldByName(fieldName)
	if !ok {
		t.Fatalf("missing generated field %s", fieldName)
	}
	if got := field.Tag.Get("json"); got != wantTag {
		t.Fatalf("field %s json tag = %q, want %q", fieldName, got, wantTag)
	}
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
