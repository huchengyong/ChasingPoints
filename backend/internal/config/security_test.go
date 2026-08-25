package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

const securityConfigYAML = `Security:
  HTTPAllowedOrigins: ""
  WebSocketAllowedOrigins: ""
  AllowNativeMissingOrigin: false
  HuaweiOAuth:
    Enabled: false
    ClientID: ""
    ClientSecret: ""
    VerifyURL: ""
    RequestTimeoutMs: 5000
  MatchInvite:
    SigningSecret: ""
    TTLSeconds: 300
`

func TestSecurityConfigLoadsEnvironmentValues(t *testing.T) {
	t.Setenv("HTTP_ALLOWED_ORIGINS", "https://app.example.com, https://admin.example.com")
	t.Setenv("WEBSOCKET_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("ALLOW_NATIVE_MISSING_ORIGIN", "true")
	t.Setenv("HUAWEI_OAUTH_ENABLED", "true")
	t.Setenv("HUAWEI_OAUTH_CLIENT_ID", "huawei-client")
	t.Setenv("HUAWEI_OAUTH_CLIENT_SECRET", "huawei-secret")
	t.Setenv("HUAWEI_OAUTH_VERIFY_URL", "https://oauth-login.cloud.huawei.com/oauth2/v3/tokeninfo")
	t.Setenv("HUAWEI_OAUTH_REQUEST_TIMEOUT_MS", "3000")
	t.Setenv("MATCH_INVITE_SIGNING_SECRET", "invite-secret")
	t.Setenv("MATCH_INVITE_TTL_SECONDS", "180")

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(securityConfigYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var cfg struct {
		Security SecurityConfig
	}
	if err := conf.Load(path, &cfg, conf.UseEnv()); err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.Security.HTTPOriginList(); len(got) != 2 || got[0] != "https://app.example.com" || got[1] != "https://admin.example.com" {
		t.Fatalf("unexpected http origins: %#v", got)
	}
	if !cfg.Security.AllowNativeMissingOrigin || !cfg.Security.HuaweiOAuth.Enabled {
		t.Fatalf("expected env booleans to load: %+v", cfg.Security)
	}
	if cfg.Security.HuaweiOAuth.RequestTimeoutMs != 3000 || cfg.Security.MatchInvite.TTLSeconds != 180 {
		t.Fatalf("expected env numeric values to load: %+v", cfg.Security)
	}
}

func TestValidateProductionSecurityRejectsUnsafeOrigins(t *testing.T) {
	cfg := validProductionConfig()

	cfg.Security.HTTPAllowedOrigins = ""
	if err := cfg.ValidateProductionSecurity(); err == nil || !strings.Contains(err.Error(), "HTTP allowed origins") {
		t.Fatalf("expected empty HTTP origins to fail, got %v", err)
	}

	cfg = validProductionConfig()
	cfg.Security.WebSocketAllowedOrigins = "*"
	if err := cfg.ValidateProductionSecurity(); err == nil || !strings.Contains(err.Error(), "wildcard") {
		t.Fatalf("expected wildcard websocket origin to fail, got %v", err)
	}
}

func TestSecurityHTTPOriginAllowedUsesExactConfiguredOrigins(t *testing.T) {
	security := SecurityConfig{
		HTTPAllowedOrigins: "https://app.example.com,https://admin-bm.dianzaozao.com",
	}

	if !security.HTTPOriginAllowed("https://admin-bm.dianzaozao.com") {
		t.Fatal("expected configured admin origin to be allowed")
	}
	if security.HTTPOriginAllowed("https://evil.example.com") {
		t.Fatal("expected unconfigured origin to be rejected")
	}
	if security.HTTPOriginAllowed("https://app.example.com.evil.test") {
		t.Fatal("expected suffix lookalike origin to be rejected")
	}
}

func TestValidateProductionSecurityRejectsMissingProviderConfig(t *testing.T) {
	cfg := validProductionConfig()
	cfg.Security.HuaweiOAuth.Enabled = true
	cfg.Security.HuaweiOAuth.ClientSecret = ""

	if err := cfg.ValidateProductionSecurity(); err == nil || !strings.Contains(err.Error(), "Huawei OAuth") {
		t.Fatalf("expected missing Huawei config to fail, got %v", err)
	}
}

func TestValidateProductionSecurityRejectsMissingMatchInviteSecret(t *testing.T) {
	cfg := validProductionConfig()
	cfg.Security.MatchInvite.SigningSecret = ""

	if err := cfg.ValidateProductionSecurity(); err == nil || !strings.Contains(err.Error(), "match invite") {
		t.Fatalf("expected missing match invite secret to fail, got %v", err)
	}
}

func TestValidateProductionSecurityAllowsExactOriginsAndRequiredSecrets(t *testing.T) {
	cfg := validProductionConfig()

	if err := cfg.ValidateProductionSecurity(); err != nil {
		t.Fatalf("expected valid production config, got %v", err)
	}
}

func TestValidateProductionSecuritySkipsLocalEnvironment(t *testing.T) {
	cfg := Config{}
	cfg.AppEnv = "local"
	cfg.Security.HTTPAllowedOrigins = "*"

	if err := cfg.ValidateProductionSecurity(); err != nil {
		t.Fatalf("expected local config to skip production security validation, got %v", err)
	}
}

func validProductionConfig() Config {
	cfg := Config{}
	cfg.AppEnv = "prod"
	cfg.Security.HTTPAllowedOrigins = "https://app.example.com,https://admin-bm.dianzaozao.com"
	cfg.Security.WebSocketAllowedOrigins = "https://app.example.com"
	cfg.Security.HuaweiOAuth.Enabled = true
	cfg.Security.HuaweiOAuth.ClientID = "huawei-client"
	cfg.Security.HuaweiOAuth.ClientSecret = "huawei-secret"
	cfg.Security.HuaweiOAuth.VerifyURL = "https://oauth-login.cloud.huawei.com/oauth2/v3/tokeninfo"
	cfg.Security.HuaweiOAuth.RequestTimeoutMs = 5000
	cfg.Security.MatchInvite.SigningSecret = "invite-secret"
	cfg.Security.MatchInvite.TTLSeconds = 300
	return cfg
}
