package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chasing_points/internal/config"
)

func TestHuaweiVerifierReturnsVerifiedIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("credential") != "credential-1" || r.Form.Get("client_id") != "huawei-client" {
			t.Fatalf("unexpected verifier form: %s", r.Form.Encode())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"huawei-openid-1","union_id":"union-1","aud":"huawei-client"}`))
	}))
	defer server.Close()

	verifier := NewHuaweiVerifier(config.HuaweiOAuthConfig{
		Enabled:          true,
		ClientID:         "huawei-client",
		ClientSecret:     "huawei-secret",
		VerifyURL:        server.URL,
		RequestTimeoutMs: 1000,
	})

	identity, err := verifier.Verify(context.Background(), VerifyRequest{
		Provider:       ProviderHuawei,
		Credential:     "credential-1",
		CredentialType: "authorization_code",
		Platform:       "harmony",
	})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if identity.Subject != "huawei-openid-1" || identity.UnionID != "union-1" || identity.Provider != ProviderHuawei {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestHuaweiVerifierRejectsFailedAndInvalidResponses(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		category ErrorCategory
	}{
		{name: "rejected", status: http.StatusUnauthorized, body: `{"error":"invalid_token"}`, category: ErrorRejected},
		{name: "invalid json", status: http.StatusOK, body: `not-json`, category: ErrorInvalidResponse},
		{name: "audience mismatch", status: http.StatusOK, body: `{"sub":"huawei-openid-1","aud":"other-client"}`, category: ErrorRejected},
		{name: "upstream unavailable", status: http.StatusBadGateway, body: `{}`, category: ErrorUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			verifier := NewHuaweiVerifier(config.HuaweiOAuthConfig{
				Enabled:          true,
				ClientID:         "huawei-client",
				ClientSecret:     "huawei-secret",
				VerifyURL:        server.URL,
				RequestTimeoutMs: 1000,
			})
			_, err := verifier.Verify(context.Background(), VerifyRequest{Provider: ProviderHuawei, Credential: "credential-1"})
			if CategoryOf(err) != tt.category {
				t.Fatalf("expected category %s, got %v", tt.category, err)
			}
			if err != nil && strings.Contains(err.Error(), "credential-1") {
				t.Fatalf("verifier error leaked credential: %v", err)
			}
		})
	}
}
