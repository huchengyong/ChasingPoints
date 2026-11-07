package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

const seasonLifecycleConfigYAML = `SeasonLifecycle:
  Enabled: false
  AnchorDate: ""
  InitialNumber: 1
  CycleMonths: 1
  Timezone: Asia/Shanghai
`

func TestCompetitiveReadModelDefaultsToDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("CompetitiveReadModel: {}\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	var cfg struct {
		CompetitiveReadModel CompetitiveReadModelConfig
	}
	if err := conf.Load(path, &cfg); err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.CompetitiveReadModel.ReadMode != "disabled" {
		t.Fatalf("competitive read mode default = %q, want disabled", cfg.CompetitiveReadModel.ReadMode)
	}
}

func TestObservabilityConfigLoadsEnvironmentValue(t *testing.T) {
	t.Setenv("OBSERVABILITY_SLOW_SQL_THRESHOLD_MS", "750")

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("Observability:\n  SlowSQLThresholdMs: 500\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var cfg struct {
		Observability ObservabilityConfig
	}
	if err := conf.Load(path, &cfg, conf.UseEnv()); err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Observability.SlowSQLThresholdMs != 750 {
		t.Fatalf("slow SQL threshold = %d", cfg.Observability.SlowSQLThresholdMs)
	}
}

func TestSeasonLifecycleConfigLoadsEnvironmentValues(t *testing.T) {
	t.Setenv("SEASON_LIFECYCLE_ENABLED", "true")
	t.Setenv("SEASON_LIFECYCLE_ANCHOR_DATE", "2026-08-01")
	t.Setenv("SEASON_LIFECYCLE_INITIAL_NUMBER", "8")
	t.Setenv("SEASON_LIFECYCLE_CYCLE_MONTHS", "2")
	t.Setenv("SEASON_LIFECYCLE_TIMEZONE", "Asia/Shanghai")

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(seasonLifecycleConfigYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var cfg struct {
		SeasonLifecycle SeasonLifecycleConfig
	}
	if err := conf.Load(path, &cfg, conf.UseEnv()); err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.SeasonLifecycle.Enabled || cfg.SeasonLifecycle.AnchorDate != "2026-08-01" ||
		cfg.SeasonLifecycle.InitialNumber != 8 || cfg.SeasonLifecycle.CycleMonths != 2 ||
		cfg.SeasonLifecycle.Timezone != "Asia/Shanghai" {
		t.Fatalf("unexpected lifecycle config: %+v", cfg.SeasonLifecycle)
	}
}
