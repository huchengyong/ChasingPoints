package svc

import (
	"testing"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	"chasing_points/internal/pkg/geocode"
)

func TestNewGeocodeClientProviderSelection(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		amapKey      string
		wantErr      bool
		wantAmap     bool
		wantApihz    bool
		wantProvider string
	}{
		{name: "amap 配置了 key", provider: "amap", amapKey: "amap-key-value", wantAmap: true, wantProvider: "amap"},
		{name: "amap 未配置 key", provider: "amap", wantErr: true},
		{name: "apihz", provider: "apihz", wantApihz: true, wantProvider: "apihz"},
		{name: "未知 provider", provider: "unknown", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var c config.Config
			c.Geocode.Provider = test.provider
			c.Geocode.AmapKey = test.amapKey

			client, provider, err := newGeocodeClient(c)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error, got client %T", client)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if provider != test.wantProvider {
				t.Fatalf("expected provider %s, got %s", test.wantProvider, provider)
			}
			if _, ok := client.(*geocode.AmapClient); ok != test.wantAmap {
				t.Fatalf("expected amap client %v, got %T", test.wantAmap, client)
			}
			if _, ok := client.(*geocode.ApihzClient); ok != test.wantApihz {
				t.Fatalf("expected apihz client %v, got %T", test.wantApihz, client)
			}
		})
	}
}

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
