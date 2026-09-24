package migrations

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const careerAchievementMigration = "20260730154000_redesign_career_achievements.sql"

func TestCareerAchievementMigrationContainsV2Catalog(t *testing.T) {
	content, err := os.ReadFile(careerAchievementMigration)
	if err != nil {
		t.Fatalf("read career achievement migration: %v", err)
	}
	text := string(content)
	downIndex := strings.Index(text, "-- +goose Down")
	if downIndex < 0 {
		t.Fatal("expected goose down block")
	}
	upBlock := text[:downIndex]
	downBlock := text[downIndex:]

	allKeys := []string{
		"match_1", "match_10", "match_50", "match_100",
		"wins_1", "wins_10", "wins_50", "wins_100",
		"streak_3", "streak_5", "streak_10",
		"tournament_join_1", "tournament_finish_1", "tournament_finish_10", "tournament_champion_1",
		"snooker_break_50_1", "snooker_break_100_1", "break_147_1",
		"chasing_golden_break_1", "chasing_nine_on_break_1",
		"continue_clear_1", "break_clear_1", "break_clear_20",
		"american_golden_break_1", "american_nine_on_break_1",
	}
	for _, key := range allKeys {
		if !strings.Contains(upBlock, "'"+key+"'") {
			t.Fatalf("career catalog missing %s", key)
		}
		icon := fmt.Sprintf("'/static/images/achievements/%s.png'", key)
		if !strings.Contains(upBlock, icon) {
			t.Fatalf("career catalog missing icon %s", icon)
		}
		assetPath := filepath.Join("..", "..", "app", "static", "images", "achievements", key+".png")
		if _, err := os.Stat(assetPath); err != nil {
			t.Fatalf("career icon asset %s is unavailable: %v", assetPath, err)
		}
	}
	chasingIcon, err := os.ReadFile(filepath.Join("..", "..", "app", "static", "images", "achievements", "chasing_golden_break_1.png"))
	if err != nil {
		t.Fatalf("read chasing nine-ball icon: %v", err)
	}
	americanIcon, err := os.ReadFile(filepath.Join("..", "..", "app", "static", "images", "achievements", "american_golden_break_1.png"))
	if err != nil {
		t.Fatalf("read american nine-ball icon: %v", err)
	}
	if bytes.Equal(chasingIcon, americanIcon) {
		t.Fatal("chasing and american nine-ball icons must be visually distinct assets")
	}
	if got := strings.Count(upBlock, "\n  ('"); got != 25 {
		t.Fatalf("career catalog should contain 25 value rows, got %d", got)
	}

	newKeys := []string{
		"match_1", "match_50", "wins_1", "wins_50", "streak_3", "tournament_finish_1",
		"snooker_break_50_1", "snooker_break_100_1", "continue_clear_1",
		"chasing_golden_break_1", "chasing_nine_on_break_1",
		"american_golden_break_1", "american_nine_on_break_1",
	}
	for _, key := range newKeys {
		if !strings.Contains(downBlock, "'"+key+"'") {
			t.Fatalf("safe down block must disable new key %s", key)
		}
	}

	requiredSnippets := []string{
		"ON DUPLICATE KEY UPDATE",
		"('break_147_1', '满分时刻', '在斯诺克对局中打出 1 次单杆 147', '/static/images/achievements/break_147_1.png', 'special', 1",
		"('chasing_golden_break_1', '小金初现', '在九球追分对局中打出 1 次小金', '/static/images/achievements/chasing_golden_break_1.png', 'special', 2",
		"('break_clear_20', '全台掌控', '在中式八球对局中累计打出 20 次炸清', '/static/images/achievements/break_clear_20.png', 'special', 3",
		"('american_nine_on_break_1', '大金降临', '在美式九球对局中打出 1 次大金', '/static/images/achievements/american_nine_on_break_1.png', 'special', 4",
		"'title_wins_100', '百胜名将'",
		"'title_streak_10', '连胜主宰'",
		"UPDATE `user_titles` AS `ut`",
		"`ut`.`title_name` = `a`.`reward_title_name`",
		"`ut`.`source_ref_name` = `a`.`name`",
		"SET `status` = 0",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("career migration should contain %q", snippet)
		}
	}
	if strings.Contains(downBlock, "DELETE FROM `achievements`") {
		t.Fatal("career migration down must not delete achievement definitions referenced by user assets")
	}
}
