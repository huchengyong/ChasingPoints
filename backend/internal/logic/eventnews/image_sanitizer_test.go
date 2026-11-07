package eventnews

import (
	"testing"

	"chasing_points/internal/model"
)

func TestSanitizePublicImageURLFiltersWSTHost(t *testing.T) {
	for _, value := range []string{
		"https://images.gc.wstservices.co.uk/player.png",
		"https://IMAGES.GC.WSTSERVICES.CO.UK/cover.jpg",
		"//images.gc.wstservices.co.uk/cover.webp",
	} {
		if got := sanitizePublicImageURL(value); got != "" {
			t.Fatalf("expected WST URL to be filtered: input=%q output=%q", value, got)
		}
	}

	for _, value := range []string{
		"https://cdn.example.com/wst/player.png",
		"https://images.gc.wstservices.co.uk.evil.example/cover.png",
	} {
		if got := sanitizePublicImageURL(value); got != value {
			t.Fatalf("expected non-WST URL to remain: input=%q output=%q", value, got)
		}
	}
}

func TestMapEventNewsInfoPrefersMirroredTournamentCover(t *testing.T) {
	info := mapEventNewsInfo(model.EventNews{
		Title:      "WST Event",
		SourceType: "official",
		SourceName: "WST",
		CoverImage: "https://images.gc.wstservices.co.uk/event.png",
	}, &model.Tournament{
		Name:       "WST Tournament",
		CoverImage: "https://cdn.example.com/wst/tournaments/cover.png",
	}, nil)

	if info.CoverImage != "https://cdn.example.com/wst/tournaments/cover.png" {
		t.Fatalf("expected mirrored tournament cover, got %#v", info)
	}
}

func TestEventNewsMappingsDoNotReturnResidualWSTImages(t *testing.T) {
	info := mapEventNewsInfo(model.EventNews{
		Title:      "WST Event",
		SourceType: "official",
		SourceName: "WST",
		CoverImage: "https://images.gc.wstservices.co.uk/event.png",
	}, &model.Tournament{
		Name:       "WST Tournament",
		CoverImage: "https://images.gc.wstservices.co.uk/tournament.png",
	}, nil)
	if info.CoverImage != "" {
		t.Fatalf("expected residual WST event cover to be empty, got %#v", info)
	}

	tournament := mapTournamentInfo(&model.Tournament{
		Name:       "WST Tournament",
		CoverImage: "https://images.gc.wstservices.co.uk/tournament.png",
	})
	if tournament.CoverImage != "" {
		t.Fatalf("expected residual WST tournament cover to be empty, got %#v", tournament)
	}

	players := map[int64]model.Player{
		1: {Id: 1, Avatar: "https://images.gc.wstservices.co.uk/player.png"},
		2: {Id: 2, Avatar: "https://cdn.example.com/wst/players/player.png"},
	}
	if got := resolvePlayerAvatar(players, 1); got != "" {
		t.Fatalf("expected residual WST player avatar to be empty, got %q", got)
	}
	if got := resolvePlayerAvatar(players, 2); got != "https://cdn.example.com/wst/players/player.png" {
		t.Fatalf("expected mirrored player avatar, got %q", got)
	}
}
