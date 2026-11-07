package svc

import (
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
)

func TestCompetitiveReadModelsEnabledRequiresExplicitEnabledMode(t *testing.T) {
	ctx := &ServiceContext{CompetitiveReadModel: &model.CompetitiveReadModel{}}
	if ctx.CompetitiveReadModelsEnabled() {
		t.Fatal("empty read mode must keep new read models disabled")
	}

	ctx.Config.CompetitiveReadModel.ReadMode = "enabled"
	if !ctx.CompetitiveReadModelsEnabled() {
		t.Fatal("enabled read mode must allow new read models")
	}

	ctx.Config = config.Config{CompetitiveReadModel: config.CompetitiveReadModelConfig{ReadMode: "disabled"}}
	if ctx.CompetitiveReadModelsEnabled() {
		t.Fatal("disabled read mode must keep legacy reads active")
	}
}
