package config

import "testing"

func TestComplianceFeatureTogglesFollowRestrictedMode(t *testing.T) {
	cfg := Config{}
	cfg.Compliance.RestrictedMode = true

	if cfg.SocialEnabled() {
		t.Fatal("expected social to be disabled in restricted mode")
	}
	if cfg.MemberPaymentEnabled() {
		t.Fatal("expected member payment to be disabled in restricted mode")
	}
	if cfg.UserTournamentEnabled() {
		t.Fatal("expected user tournament actions to be disabled in restricted mode")
	}
}

func TestComplianceFeatureOverridesCanReopenSingleCapability(t *testing.T) {
	cfg := Config{}
	cfg.Compliance.RestrictedMode = true
	cfg.Compliance.AllowSocial = true

	if !cfg.SocialEnabled() {
		t.Fatal("expected social to be enabled when explicitly reopened")
	}
	if cfg.MemberPaymentEnabled() {
		t.Fatal("expected member payment to remain disabled")
	}
}
