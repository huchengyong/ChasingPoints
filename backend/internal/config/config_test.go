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
