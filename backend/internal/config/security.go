package config

import (
	"fmt"
	"net/url"
	"strings"
)

type SecurityConfig struct {
	HTTPAllowedOrigins       string `json:",env=HTTP_ALLOWED_ORIGINS,optional"`
	WebSocketAllowedOrigins  string `json:",env=WEBSOCKET_ALLOWED_ORIGINS,optional"`
	AllowNativeMissingOrigin bool   `json:",env=ALLOW_NATIVE_MISSING_ORIGIN,default=false"`

	HuaweiOAuth HuaweiOAuthConfig
	MatchInvite MatchInviteConfig
}

type HuaweiOAuthConfig struct {
	Enabled          bool   `json:",env=HUAWEI_OAUTH_ENABLED,default=false"`
	ClientID         string `json:",env=HUAWEI_OAUTH_CLIENT_ID,optional"`
	ClientSecret     string `json:",env=HUAWEI_OAUTH_CLIENT_SECRET,optional"`
	VerifyURL        string `json:",env=HUAWEI_OAUTH_VERIFY_URL,optional"`
	RequestTimeoutMs int    `json:",env=HUAWEI_OAUTH_REQUEST_TIMEOUT_MS,default=5000"`
}

type MatchInviteConfig struct {
	SigningSecret string `json:",env=MATCH_INVITE_SIGNING_SECRET,optional"`
	TTLSeconds    int    `json:",env=MATCH_INVITE_TTL_SECONDS,default=300"`
}

func (c Config) ValidateProductionSecurity() error {
	return c.Security.ValidateForEnvironment(c.AppEnv)
}

func (s SecurityConfig) ValidateForEnvironment(appEnv string) error {
	if !IsProductionEnv(appEnv) {
		return nil
	}

	if err := validateProductionOrigins("HTTP allowed origins", s.HTTPOriginList()); err != nil {
		return err
	}
	if err := validateProductionOrigins("WebSocket allowed origins", s.WebSocketOriginList()); err != nil {
		return err
	}
	if s.HuaweiOAuth.Enabled {
		if strings.TrimSpace(s.HuaweiOAuth.ClientID) == "" ||
			strings.TrimSpace(s.HuaweiOAuth.ClientSecret) == "" ||
			strings.TrimSpace(s.HuaweiOAuth.VerifyURL) == "" {
			return fmt.Errorf("Huawei OAuth verifier is enabled but required config is missing")
		}
		if s.HuaweiOAuth.RequestTimeoutMs <= 0 {
			return fmt.Errorf("Huawei OAuth verifier timeout must be positive")
		}
	}
	if strings.TrimSpace(s.MatchInvite.SigningSecret) == "" {
		return fmt.Errorf("match invite signing secret is required in production")
	}
	if s.MatchInvite.TTLSeconds <= 0 {
		return fmt.Errorf("match invite ttl must be positive")
	}
	return nil
}

func IsProductionEnv(appEnv string) bool {
	switch strings.ToLower(strings.TrimSpace(appEnv)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

func (s SecurityConfig) HTTPOriginList() []string {
	return parseOriginList(s.HTTPAllowedOrigins)
}

func (s SecurityConfig) WebSocketOriginList() []string {
	return parseOriginList(s.WebSocketAllowedOrigins)
}

func (s SecurityConfig) HTTPOriginAllowed(origin string) bool {
	return OriginAllowed(origin, s.HTTPOriginList())
}

func OriginAllowed(origin string, allowedOrigins []string) bool {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	if origin == "" {
		return false
	}
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(origin, strings.TrimRight(strings.TrimSpace(allowed), "/")) {
			return true
		}
	}
	return false
}

func parseOriginList(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	seen := make(map[string]struct{}, len(parts))
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := strings.TrimRight(strings.TrimSpace(part), "/")
		if origin == "" {
			continue
		}
		key := strings.ToLower(origin)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		origins = append(origins, origin)
	}
	return origins
}

func validateProductionOrigins(name string, origins []string) error {
	if len(origins) == 0 {
		return fmt.Errorf("%s must not be empty in production", name)
	}
	for _, origin := range origins {
		if origin == "*" {
			return fmt.Errorf("%s must not contain wildcard origin", name)
		}
		if err := validateOrigin(origin); err != nil {
			return fmt.Errorf("%s contains invalid origin %q: %w", name, origin, err)
		}
	}
	return nil
}

func validateOrigin(origin string) error {
	parsed, err := url.Parse(origin)
	if err != nil {
		return err
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("scheme must be http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("host is required")
	}
	if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("origin must not contain path, query or fragment")
	}
	return nil
}
