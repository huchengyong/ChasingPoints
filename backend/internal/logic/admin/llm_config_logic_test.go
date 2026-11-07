package admin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"chasing_points/internal/model"
	"chasing_points/internal/svc"
	"chasing_points/internal/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const testLlmAPIKey = "sk-test-0123456789abcdef"

func newLlmAdminTestSvc(t *testing.T) *svc.ServiceContext {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.LlmConfig{}); err != nil {
		t.Fatalf("prepare llm config schema: %v", err)
	}

	return &svc.ServiceContext{
		DB:             db,
		LlmConfigModel: model.NewLlmConfigModel(db),
	}
}

func validLlmUpdateReq() *types.AdminLlmConfigUpdateReq {
	return &types.AdminLlmConfigUpdateReq{
		ApiFormat:   "anthropic",
		EndpointUrl: "https://opencode.ai/inference/anthropic/v1/messages",
		ApiKey:      testLlmAPIKey,
		Model:       "claude-haiku-4-5",
		MaxTokens:   1024,
	}
}

func storedLlmConfig(t *testing.T, svcCtx *svc.ServiceContext) *model.LlmConfig {
	t.Helper()

	cfg, err := svcCtx.LlmConfigModel.FindByKey(model.DefaultLlmConfigKey)
	if err != nil {
		t.Fatalf("find llm config: %v", err)
	}
	return cfg
}

func TestAdminGetLlmConfigReturnsEmptyWhenMissing(t *testing.T) {
	svcCtx := newLlmAdminTestSvc(t)

	resp, err := NewAdminGetLlmConfigLogic(context.Background(), svcCtx).AdminGetLlmConfig()
	if err != nil {
		t.Fatalf("get llm config: %v", err)
	}
	if !resp.Success || resp.Code != 0 {
		t.Fatalf("expected success response, got %#v", resp)
	}
	if resp.ApiFormat != "" || resp.HasApiKey || resp.ApiKeyMasked != "" || resp.EndpointUrl != "" || resp.Model != "" || resp.MaxTokens != 0 {
		t.Fatalf("expected empty config, got %#v", resp)
	}
	if storedLlmConfig(t, svcCtx) != nil {
		t.Fatal("reading config must not create a default record")
	}
}

func TestAdminUpdateLlmConfigSavesAndGetNeverReturnsPlainAPIKey(t *testing.T) {
	svcCtx := newLlmAdminTestSvc(t)

	updateResp, err := NewAdminUpdateLlmConfigLogic(newAdminContextWithID(9527), svcCtx).AdminUpdateLlmConfig(validLlmUpdateReq())
	if err != nil {
		t.Fatalf("update llm config: %v", err)
	}
	if !updateResp.Success {
		t.Fatalf("expected update success, got %#v", updateResp)
	}

	stored := storedLlmConfig(t, svcCtx)
	if stored == nil || stored.APIKey != testLlmAPIKey || stored.UpdatedBy != 9527 {
		t.Fatalf("unexpected stored config: %+v", stored)
	}

	getResp, err := NewAdminGetLlmConfigLogic(context.Background(), svcCtx).AdminGetLlmConfig()
	if err != nil {
		t.Fatalf("get llm config: %v", err)
	}
	if getResp.ApiFormat != "anthropic" || getResp.EndpointUrl != "https://opencode.ai/inference/anthropic/v1/messages" || getResp.Model != "claude-haiku-4-5" || getResp.MaxTokens != 1024 {
		t.Fatalf("unexpected config response: %#v", getResp)
	}
	if !getResp.HasApiKey || getResp.ApiKeyMasked != "****cdef" {
		t.Fatalf("expected masked api key, got %#v", getResp)
	}

	body, err := json.Marshal(getResp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if strings.Contains(string(body), testLlmAPIKey) || strings.Contains(string(body), "0123456789") {
		t.Fatalf("response must not contain the plain api key: %s", body)
	}
}

func TestAdminUpdateLlmConfigBlankAPIKeyKeepsStoredKey(t *testing.T) {
	svcCtx := newLlmAdminTestSvc(t)
	logic := NewAdminUpdateLlmConfigLogic(newAdminContextWithID(1), svcCtx)

	if resp, _ := logic.AdminUpdateLlmConfig(validLlmUpdateReq()); resp == nil || !resp.Success {
		t.Fatalf("seed config failed: %#v", resp)
	}

	req := validLlmUpdateReq()
	req.ApiKey = "   "
	req.Model = "claude-sonnet-4-5"
	req.MaxTokens = 4096
	resp, err := logic.AdminUpdateLlmConfig(req)
	if err != nil || !resp.Success {
		t.Fatalf("expected success when keeping key, resp=%#v err=%v", resp, err)
	}

	stored := storedLlmConfig(t, svcCtx)
	if stored.APIKey != testLlmAPIKey {
		t.Fatalf("blank api key must keep the stored key, got %q", stored.APIKey)
	}
	if stored.Model != "claude-sonnet-4-5" || stored.MaxTokens != 4096 {
		t.Fatalf("other fields should be updated, got %+v", stored)
	}
}

func TestAdminUpdateLlmConfigReplacesAPIKeyWhenProvided(t *testing.T) {
	svcCtx := newLlmAdminTestSvc(t)
	logic := NewAdminUpdateLlmConfigLogic(newAdminContextWithID(1), svcCtx)

	if resp, _ := logic.AdminUpdateLlmConfig(validLlmUpdateReq()); resp == nil || !resp.Success {
		t.Fatalf("seed config failed: %#v", resp)
	}

	req := validLlmUpdateReq()
	req.ApiKey = "  sk-rotated-9876543210zyxw  "
	if resp, err := logic.AdminUpdateLlmConfig(req); err != nil || !resp.Success {
		t.Fatalf("expected success, resp=%#v err=%v", resp, err)
	}

	if stored := storedLlmConfig(t, svcCtx); stored.APIKey != "sk-rotated-9876543210zyxw" {
		t.Fatalf("expected trimmed rotated key, got %q", stored.APIKey)
	}
}

func TestAdminUpdateLlmConfigSwitchesAPIFormatAndKeepsStoredKey(t *testing.T) {
	svcCtx := newLlmAdminTestSvc(t)
	logic := NewAdminUpdateLlmConfigLogic(newAdminContextWithID(1), svcCtx)

	if resp, _ := logic.AdminUpdateLlmConfig(validLlmUpdateReq()); resp == nil || !resp.Success {
		t.Fatalf("seed config failed: %#v", resp)
	}

	req := validLlmUpdateReq()
	req.ApiFormat = " openai "
	req.ApiKey = ""
	req.EndpointUrl = "https://api.openai.com/v1/chat/completions"
	if resp, err := logic.AdminUpdateLlmConfig(req); err != nil || !resp.Success {
		t.Fatalf("expected success, resp=%#v err=%v", resp, err)
	}

	stored := storedLlmConfig(t, svcCtx)
	if stored.APIFormat != model.LlmAPIFormatOpenAI || stored.APIKey != testLlmAPIKey {
		t.Fatalf("expected openai format with the original key, got %+v", stored)
	}

	getResp, err := NewAdminGetLlmConfigLogic(context.Background(), svcCtx).AdminGetLlmConfig()
	if err != nil || getResp.ApiFormat != "openai" {
		t.Fatalf("expected openai format in response, resp=%#v err=%v", getResp, err)
	}
}

func TestAdminUpdateLlmConfigRejectsInvalidValues(t *testing.T) {
	testCases := []struct {
		name   string
		mutate func(req *types.AdminLlmConfigUpdateReq)
	}{
		{"missing api format", func(r *types.AdminLlmConfigUpdateReq) { r.ApiFormat = "  " }},
		{"unknown api format", func(r *types.AdminLlmConfigUpdateReq) { r.ApiFormat = "gemini" }},
		{"api format is case sensitive", func(r *types.AdminLlmConfigUpdateReq) { r.ApiFormat = "OpenAI" }},
		{"missing endpoint", func(r *types.AdminLlmConfigUpdateReq) { r.EndpointUrl = "  " }},
		{"http endpoint", func(r *types.AdminLlmConfigUpdateReq) { r.EndpointUrl = "http://opencode.ai/v1/messages" }},
		{"non http scheme", func(r *types.AdminLlmConfigUpdateReq) { r.EndpointUrl = "ftp://opencode.ai/v1/messages" }},
		{"endpoint without host", func(r *types.AdminLlmConfigUpdateReq) { r.EndpointUrl = "https:///v1/messages" }},
		{"endpoint without scheme", func(r *types.AdminLlmConfigUpdateReq) { r.EndpointUrl = "opencode.ai/v1/messages" }},
		{"endpoint with credentials", func(r *types.AdminLlmConfigUpdateReq) { r.EndpointUrl = "https://user:pass@opencode.ai/v1/messages" }},
		{"endpoint too long", func(r *types.AdminLlmConfigUpdateReq) { r.EndpointUrl = "https://a.com/" + strings.Repeat("x", 600) }},
		{"api key with space", func(r *types.AdminLlmConfigUpdateReq) { r.ApiKey = "sk-abc def" }},
		{"api key with newline", func(r *types.AdminLlmConfigUpdateReq) { r.ApiKey = "sk-abc\r\nX-Evil: 1" }},
		{"api key too long", func(r *types.AdminLlmConfigUpdateReq) { r.ApiKey = strings.Repeat("k", 513) }},
		{"missing model", func(r *types.AdminLlmConfigUpdateReq) { r.Model = "" }},
		{"model with space", func(r *types.AdminLlmConfigUpdateReq) { r.Model = "claude haiku" }},
		{"model too long", func(r *types.AdminLlmConfigUpdateReq) { r.Model = strings.Repeat("m", 129) }},
		{"zero max tokens", func(r *types.AdminLlmConfigUpdateReq) { r.MaxTokens = 0 }},
		{"negative max tokens", func(r *types.AdminLlmConfigUpdateReq) { r.MaxTokens = -1 }},
		{"max tokens too large", func(r *types.AdminLlmConfigUpdateReq) { r.MaxTokens = 1000001 }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svcCtx := newLlmAdminTestSvc(t)
			req := validLlmUpdateReq()
			tc.mutate(req)

			resp, err := NewAdminUpdateLlmConfigLogic(newAdminContextWithID(1), svcCtx).AdminUpdateLlmConfig(req)
			if err != nil {
				t.Fatalf("update llm config: %v", err)
			}
			if resp.Success || resp.Code != 400 {
				t.Fatalf("expected 400 rejection, got %#v", resp)
			}
			if storedLlmConfig(t, svcCtx) != nil {
				t.Fatal("rejected request must not be persisted")
			}
		})
	}
}

func TestAdminUpdateLlmConfigRequiresAPIKeyOnFirstSave(t *testing.T) {
	svcCtx := newLlmAdminTestSvc(t)
	req := validLlmUpdateReq()
	req.ApiKey = ""

	resp, err := NewAdminUpdateLlmConfigLogic(newAdminContextWithID(1), svcCtx).AdminUpdateLlmConfig(req)
	if err != nil {
		t.Fatalf("update llm config: %v", err)
	}
	if resp.Success || resp.Code != 400 || resp.Message != "请填写 API Key" {
		t.Fatalf("expected api key required rejection, got %#v", resp)
	}
	if storedLlmConfig(t, svcCtx) != nil {
		t.Fatal("rejected request must not be persisted")
	}
}

func TestMaskLlmAPIKey(t *testing.T) {
	testCases := []struct {
		key  string
		want string
	}{
		{"", ""},
		{"short", "****"},
		{"sk-12345678", "****"},
		{"sk-123456789", "****6789"},
		{testLlmAPIKey, "****cdef"},
	}
	for _, tc := range testCases {
		if got := maskLlmAPIKey(tc.key); got != tc.want {
			t.Fatalf("maskLlmAPIKey(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}
