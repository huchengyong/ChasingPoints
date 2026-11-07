package oauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const ProviderHuawei = "huawei"

type ErrorCategory string

const (
	ErrorUnsupportedProvider ErrorCategory = "unsupported_provider"
	ErrorRejected            ErrorCategory = "rejected"
	ErrorUnavailable         ErrorCategory = "unavailable"
	ErrorInvalidResponse     ErrorCategory = "invalid_response"
)

type VerifyRequest struct {
	Provider       string
	Credential     string
	CredentialType string
	Platform       string
}

type Identity struct {
	Provider string
	Subject  string
	UnionID  string
}

type Verifier interface {
	Verify(ctx context.Context, req VerifyRequest) (*Identity, error)
}

type VerifyError struct {
	Category ErrorCategory
	Message  string
	Err      error
}

func (e *VerifyError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return string(e.Category)
}

func (e *VerifyError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewVerifyError(category ErrorCategory, message string, err error) error {
	return &VerifyError{Category: category, Message: message, Err: err}
}

func CategoryOf(err error) ErrorCategory {
	var verifyErr *VerifyError
	if errors.As(err, &verifyErr) {
		return verifyErr.Category
	}
	return ""
}

type ProviderVerifier struct {
	Huawei Verifier
}

func (v ProviderVerifier) Verify(ctx context.Context, req VerifyRequest) (*Identity, error) {
	provider := normalizeProvider(req.Provider)
	switch provider {
	case ProviderHuawei:
		if v.Huawei == nil {
			return nil, NewVerifyError(ErrorUnavailable, "Huawei verifier is not configured", nil)
		}
		req.Provider = provider
		return v.Huawei.Verify(ctx, req)
	default:
		return nil, NewVerifyError(ErrorUnsupportedProvider, "unsupported OAuth provider", nil)
	}
}

type StaticVerifier struct {
	Identity *Identity
	Err      error
	Delay    func(context.Context) error
}

func (v StaticVerifier) Verify(ctx context.Context, req VerifyRequest) (*Identity, error) {
	if v.Delay != nil {
		if err := v.Delay(ctx); err != nil {
			return nil, err
		}
	}
	if v.Err != nil {
		return nil, v.Err
	}
	if v.Identity == nil {
		return nil, NewVerifyError(ErrorInvalidResponse, "missing identity", nil)
	}
	identity := *v.Identity
	if identity.Provider == "" {
		identity.Provider = normalizeProvider(req.Provider)
	}
	identity.Provider = normalizeProvider(identity.Provider)
	identity.Subject = strings.TrimSpace(identity.Subject)
	identity.UnionID = strings.TrimSpace(identity.UnionID)
	if identity.Provider == "" || identity.Subject == "" {
		return nil, NewVerifyError(ErrorInvalidResponse, "invalid identity", nil)
	}
	return &identity, nil
}

func NormalizeIdentity(provider, subject, unionID string) (*Identity, error) {
	identity := &Identity{
		Provider: normalizeProvider(provider),
		Subject:  strings.TrimSpace(subject),
		UnionID:  strings.TrimSpace(unionID),
	}
	if identity.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if identity.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}
	return identity, nil
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
