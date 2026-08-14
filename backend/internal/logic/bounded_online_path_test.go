package logic

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOnlineReadPathsExcludeUnboundedAndMutatingHelpers(t *testing.T) {
	allowedListAll := map[string]bool{"rank_rebuild_service.go": true}
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || strings.HasSuffix(path, "_test.go") {
			return err
		}
		payload, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(payload)
		rel := strings.TrimPrefix(filepath.Clean(path), "."+string(filepath.Separator))
		if strings.Contains(text, ".ListAll(") && !allowedListAll[rel] {
			t.Errorf("unbounded ListAll is not allowed in online path %s", rel)
		}
		if strings.Contains(text, "COALESCE(end_time, match_time)") || strings.Contains(text, "COALESCE(completed_at") {
			t.Errorf("completion-time hot query must use completed_at directly: %s", rel)
		}
		if strings.HasPrefix(filepath.Base(path), "get_") && strings.HasSuffix(filepath.Base(path), "_logic.go") {
			for _, mutation := range []string{"SeedData(", "ExpireOld(", "FindOrCreate("} {
				if strings.Contains(text, mutation) {
					t.Errorf("GET logic %s contains mutating helper %s", rel, mutation)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan online read paths: %v", err)
	}
}
