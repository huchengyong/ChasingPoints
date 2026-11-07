package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"chasing_points/internal/config"
)

const maxVerifierResponseBytes = 1 << 20

type HuaweiVerifier struct {
	cfg    config.HuaweiOAuthConfig
	client *http.Client
}

func NewHuaweiVerifier(cfg config.HuaweiOAuthConfig) *HuaweiVerifier {
	timeout := time.Duration(cfg.RequestTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &HuaweiVerifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (v *HuaweiVerifier) Verify(ctx context.Context, req VerifyRequest) (*Identity, error) {
	if v == nil || !v.cfg.Enabled {
		return nil, NewVerifyError(ErrorUnavailable, "Huawei verifier is not enabled", nil)
	}
	if strings.TrimSpace(v.cfg.ClientID) == "" || strings.TrimSpace(v.cfg.ClientSecret) == "" || strings.TrimSpace(v.cfg.VerifyURL) == "" {
		return nil, NewVerifyError(ErrorUnavailable, "Huawei verifier is not configured", nil)
	}
	credential := strings.TrimSpace(req.Credential)
	if credential == "" {
		return nil, NewVerifyError(ErrorRejected, "OAuth credential is required", nil)
	}

	form := url.Values{}
	form.Set("client_id", strings.TrimSpace(v.cfg.ClientID))
	form.Set("client_secret", strings.TrimSpace(v.cfg.ClientSecret))
	form.Set("credential", credential)
	form.Set("credential_type", strings.TrimSpace(req.CredentialType))
	form.Set("platform", strings.TrimSpace(req.Platform))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(v.cfg.VerifyURL), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, NewVerifyError(ErrorUnavailable, "Huawei verifier request setup failed", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := v.client.Do(httpReq)
	if err != nil {
		return nil, NewVerifyError(ErrorUnavailable, "Huawei verifier request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxVerifierResponseBytes))
	if err != nil {
		return nil, NewVerifyError(ErrorUnavailable, "Huawei verifier response read failed", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusBadRequest {
		return nil, NewVerifyError(ErrorRejected, "Huawei rejected OAuth credential", nil)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, NewVerifyError(ErrorUnavailable, "Huawei verifier returned unavailable status", nil)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, NewVerifyError(ErrorInvalidResponse, "Huawei verifier response is not valid JSON", err)
	}
	if message := firstString(payload, "error", "error_description"); message != "" {
		return nil, NewVerifyError(ErrorRejected, "Huawei rejected OAuth credential", fmt.Errorf("%s", message))
	}

	audience := firstString(payload, "aud", "client_id", "azp")
	if audience != "" && audience != strings.TrimSpace(v.cfg.ClientID) {
		return nil, NewVerifyError(ErrorRejected, "Huawei credential audience mismatch", nil)
	}
	subject := firstString(payload, "sub", "open_id", "openid", "user_id")
	identity, err := NormalizeIdentity(ProviderHuawei, subject, firstString(payload, "union_id", "unionid"))
	if err != nil {
		return nil, NewVerifyError(ErrorInvalidResponse, "Huawei verifier response missing identity", err)
	}
	return identity, nil
}

func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if trimmed := strings.TrimSpace(typed); trimmed != "" {
				return trimmed
			}
		case []any:
			for _, item := range typed {
				if text, ok := item.(string); ok {
					if trimmed := strings.TrimSpace(text); trimmed != "" {
						return trimmed
					}
				}
			}
		}
	}
	return ""
}
