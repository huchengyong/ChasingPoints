package logic

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSharedResponseCachesStayRestrictedToApprovedReadPaths(t *testing.T) {
	approved := map[string][]string{
		"readcache.LoadJSON": {
			"completed_match_core_summary.go",
			filepath.Join("staticread", "cache.go"),
			filepath.Join("venue", "cache.go"),
			filepath.Join("season", "cache_version.go"),
		},
		"staticread.Load": {
			"runtime_config_cache.go",
			filepath.Join("rules", "get_glossary_logic.go"),
			filepath.Join("rules", "get_rule_content_logic.go"),
			filepath.Join("venue", "get_venue_area_options_logic.go"),
			filepath.Join("public", "get_rank_configs_logic.go"),
			filepath.Join("member", "get_member_plans_logic.go"),
		},
		"loadVenueCache(": {
			filepath.Join("venue", "cache.go"),
			filepath.Join("venue", "get_venue_list_logic.go"),
			filepath.Join("venue", "get_venue_detail_logic.go"),
			filepath.Join("venue", "get_nearby_venues_logic.go"),
		},
		"loadSeasonLeaderboardPage(": {
			filepath.Join("season", "get_season_leaderboard_logic.go"),
			filepath.Join("season", "get_season_overview_logic.go"),
		},
		"loadLeaderboardShared(": {
			filepath.Join("public", "leaderboard_cache.go"),
			filepath.Join("public", "get_leaderboard_logic.go"),
			filepath.Join("public", "get_leaderboard_summary_logic.go"),
		},
	}
	for token, paths := range approved {
		allowed := make(map[string]bool, len(paths))
		for _, path := range paths {
			allowed[path] = true
		}
		err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || strings.HasSuffix(path, "_test.go") {
				return err
			}
			payload, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(string(payload), token) {
				rel := strings.TrimPrefix(filepath.Clean(path), "."+string(filepath.Separator))
				if !allowed[rel] {
					t.Errorf("shared response cache token %q is not allowed in %s", token, rel)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan logic sources: %v", err)
		}
	}
}

func TestCompletedCoreCacheIsGuardedByCompletedStatus(t *testing.T) {
	paths := []string{
		filepath.Join("match", "get_match_detail_logic.go"),
		filepath.Join("share", "get_match_share_data_logic.go"),
		filepath.Join("public", "get_public_match_detail_logic.go"),
	}
	for _, path := range paths {
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(payload)
		guard := strings.Index(text, "match.Status == 2")
		cacheCall := strings.Index(text, "BuildCompletedMatchCoreSummary")
		if guard < 0 || cacheCall < 0 || guard > cacheCall {
			t.Fatalf("%s must guard completed core cache behind match.Status == 2", path)
		}
	}
}
