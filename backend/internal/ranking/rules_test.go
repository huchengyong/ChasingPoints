package ranking

import (
	"testing"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
)

func rulesConfig(startDate string, lifecycleEnabled bool) config.Config {
	return config.Config{
		Ranking: config.RankingConfig{RatingV2StartDate: startDate},
		SeasonLifecycle: config.SeasonLifecycleConfig{
			Enabled:       lifecycleEnabled,
			AnchorDate:    "2026-08-01",
			InitialNumber: 1,
			CycleMonths:   1,
			Timezone:      "Asia/Shanghai",
		},
	}
}

func TestRulesWithoutStartDateKeepLegacy(t *testing.T) {
	rules := RulesFor(rulesConfig("", true))
	if rules.V2Configured() || rules.Err() != nil || rules.VersionAt(time.Now()) != RulesVersionLegacy {
		t.Fatalf("empty start date must keep legacy rules: %+v", rules)
	}
}

func TestRulesSwitchAtSeasonStartInBusinessTimezone(t *testing.T) {
	rules := RulesFor(rulesConfig("2026-11-01", true))
	if !rules.V2Configured() || rules.Err() != nil {
		t.Fatalf("season start must enable V2: err=%v", rules.Err())
	}
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	before := time.Date(2026, 10, 31, 23, 59, 59, 0, shanghai)
	at := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)
	if rules.V2ActiveAt(before) || !rules.V2ActiveAt(at) {
		t.Fatalf("V2 must start at 00:00 business time: before=%v at=%v", rules.V2ActiveAt(before), rules.V2ActiveAt(at))
	}
	if rules.VersionAt(at) != RulesVersionV2 {
		t.Fatal("version must be V2 after the start")
	}

	s3 := &model.Season{StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)}
	s4 := &model.Season{StartDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)}
	if rules.IsV2Season(s3) || !rules.IsV2Season(s4) {
		t.Fatal("only seasons starting at or after the V2 start use qualification")
	}
	if matches, opponents := rules.SeasonQualification(s4); matches != SeasonQualifyMinMatches || opponents != SeasonQualifyMinOpponents {
		t.Fatalf("V2 season qualification = %d/%d", matches, opponents)
	}
	if matches, opponents := rules.SeasonQualification(s3); matches != 0 || opponents != 0 {
		t.Fatal("legacy seasons must not have qualification thresholds")
	}
}

func TestRulesRejectStartDateInsideSeason(t *testing.T) {
	rules := RulesFor(rulesConfig("2026-11-15", true))
	if rules.V2Configured() || rules.Err() == nil {
		t.Fatal("a start date inside a season must be rejected and keep legacy rules")
	}
	if RulesFor(rulesConfig("2026/11/01", true)).Err() == nil {
		t.Fatal("malformed start date must be rejected")
	}
}

func TestRulesInstantStartSwitchesAtTheInstantWithoutSeasonAlignment(t *testing.T) {
	cfg := rulesConfig("", true)
	cfg.Ranking.RatingV2StartAt = "2026-10-20T18:00:00+08:00"
	rules := RulesFor(cfg)
	if !rules.V2Configured() || rules.Err() != nil {
		t.Fatalf("instant start must enable V2: err=%v", rules.Err())
	}
	at := time.Date(2026, 10, 20, 18, 0, 0, 0, time.FixedZone("CST", 8*3600))
	if rules.V2ActiveAt(at.Add(-time.Second)) || !rules.V2ActiveAt(at) {
		t.Fatal("V2 must switch exactly at the configured instant")
	}
	// 10 月赛季早于切换时刻开始：当前赛季保持旧赛季口径，名次不设门槛。
	october := &model.Season{StartDate: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)}
	if rules.IsV2Season(october) {
		t.Fatal("a season that started before the instant must stay legacy")
	}
	november := &model.Season{StartDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)}
	if !rules.IsV2Season(november) {
		t.Fatal("a season that starts after the instant must use V2 qualification")
	}
}

func TestRulesRejectMalformedOrConflictingStart(t *testing.T) {
	malformed := rulesConfig("", true)
	malformed.Ranking.RatingV2StartAt = "2026-10-20 18:00"
	if rules := RulesFor(malformed); rules.V2Configured() || rules.Err() == nil {
		t.Fatalf("instant start without offset must be rejected: err=%v", rules.Err())
	}
	conflicting := rulesConfig("2026-11-01", true)
	conflicting.Ranking.RatingV2StartAt = "2026-10-20T18:00:00+08:00"
	if rules := RulesFor(conflicting); rules.V2Configured() || rules.Err() == nil {
		t.Fatal("setting both start date and instant must be rejected")
	}
}

func TestRulesWithoutLifecycleAcceptAnyDate(t *testing.T) {
	rules := RulesFor(rulesConfig("2026-11-15", false))
	if !rules.V2Configured() || rules.Err() != nil {
		t.Fatalf("manual seasons cannot be validated, start date must be accepted: %v", rules.Err())
	}
}

func TestRulesDayRangeUsesBusinessTimezone(t *testing.T) {
	rules := RulesFor(rulesConfig("2026-11-01", true))
	start, end := rules.DayRange(time.Date(2026, 11, 1, 17, 30, 0, 0, time.UTC)) // 北京时间 11-02 01:30
	shanghai, _ := time.LoadLocation("Asia/Shanghai")
	if !start.Equal(time.Date(2026, 11, 2, 0, 0, 0, 0, shanghai)) || !end.Equal(time.Date(2026, 11, 3, 0, 0, 0, 0, shanghai)) {
		t.Fatalf("day range = [%v, %v)", start, end)
	}
}
