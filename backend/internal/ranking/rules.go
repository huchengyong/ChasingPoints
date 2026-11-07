package ranking

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"chasing_points/internal/config"
	"chasing_points/internal/model"
	seasonx "chasing_points/internal/season"
)

// Rules 描述当前配置下排位规则 V2 的生效起点与业务时区。零值表示未启用 V2。
type Rules struct {
	v2Start  time.Time
	location *time.Location
	err      error
}

type rulesKey struct {
	startDate string
	startAt   string
	lifecycle config.SeasonLifecycleConfig
}

var rulesCache sync.Map

// RulesFor 解析并缓存配置对应的规则；配置错误时返回未启用 V2 的规则，并通过 Err 报告原因。
func RulesFor(cfg config.Config) Rules {
	key := rulesKey{
		startDate: strings.TrimSpace(cfg.Ranking.RatingV2StartDate),
		startAt:   strings.TrimSpace(cfg.Ranking.RatingV2StartAt),
		lifecycle: cfg.SeasonLifecycle,
	}
	if cached, ok := rulesCache.Load(key); ok {
		return cached.(Rules)
	}
	rules := buildRules(key)
	rulesCache.Store(key, rules)
	return rules
}

func buildRules(key rulesKey) Rules {
	location, err := seasonx.LocationForConfig(key.lifecycle)
	if err != nil {
		location = time.Local
	}
	rules := Rules{location: location}
	if key.startDate == "" && key.startAt == "" {
		return rules
	}
	if key.startDate != "" && key.startAt != "" {
		rules.err = errors.New("排位规则 V2 只能配置生效日期或生效时刻之一")
		return rules
	}
	if err != nil {
		rules.err = fmt.Errorf("排位规则 V2 无法解析赛季时区: %w", err)
		return rules
	}
	if key.startAt != "" {
		// 即时生效：时刻自带时区偏移，不要求对齐赛季起点。
		start, parseErr := time.Parse(time.RFC3339, key.startAt)
		if parseErr != nil {
			rules.err = fmt.Errorf("排位规则 V2 生效时刻格式错误（需 RFC3339，如 2026-10-20T18:00:00+08:00）: %w", parseErr)
			return rules
		}
		rules.v2Start = start
		return rules
	}
	start, err := time.ParseInLocation(time.DateOnly, key.startDate, location)
	if err != nil {
		rules.err = fmt.Errorf("排位规则 V2 生效日期格式错误: %w", err)
		return rules
	}
	if key.lifecycle.Enabled {
		policy, policyErr := seasonx.NewPolicy(key.lifecycle)
		if policyErr != nil {
			rules.err = fmt.Errorf("排位规则 V2 无法校验赛季起点: %w", policyErr)
			return rules
		}
		window, started := policy.WindowAt(start)
		if !started || !window.StartDate.Equal(start) {
			rules.err = fmt.Errorf("排位规则 V2 生效日期 %s 不是赛季起点", key.startDate)
			return rules
		}
	}
	rules.v2Start = start
	return rules
}

// Err 返回配置错误；配置错误时 V2 保持未启用。
func (r Rules) Err() error { return r.err }

// V2Configured 表示已配置合法的 V2 生效起点。
func (r Rules) V2Configured() bool { return !r.v2Start.IsZero() }

// V2StartAt 返回 V2 生效时刻；未配置时为零值。
func (r Rules) V2StartAt() time.Time { return r.v2Start }

// V2ActiveAt 表示完成于 at 的对局是否按 V2 结算。
func (r Rules) V2ActiveAt(at time.Time) bool {
	return r.V2Configured() && !at.Before(r.v2Start)
}

// VersionAt 返回 at 时刻生效的规则版本。
func (r Rules) VersionAt(at time.Time) int {
	if r.V2ActiveAt(at) {
		return RulesVersionV2
	}
	return RulesVersionLegacy
}

// DefaultsAt 返回 at 时刻新建段位行的初始值：V2 生效后直接按 V2 初始分创建。
func (r Rules) DefaultsAt(at time.Time) model.RankingDefaults {
	if r.V2ActiveAt(at) {
		return model.RankingDefaults{RankScore: InitialRating, RankLevel: model.RankLevelForScore(InitialRating), RatingVersion: model.RatingVersionV2}
	}
	return model.LegacyRankingDefaults()
}

// LeaderboardScopeAt 返回 at 时刻全站榜的上榜资格。
func (r Rules) LeaderboardScopeAt(at time.Time) model.LeaderboardScope {
	return model.LeaderboardScope{RatingV2: r.V2ActiveAt(at)}
}

// IsV2Season 表示赛季是否从 V2 起点或之后开始，这类赛季使用达标门槛。
func (r Rules) IsV2Season(season *model.Season) bool {
	if season == nil || !r.V2Configured() {
		return false
	}
	start, _ := seasonx.Bounds(season, r.Location())
	return !start.Before(r.v2Start)
}

// Location 返回排位按日统计使用的业务时区。
func (r Rules) Location() *time.Location {
	if r.location == nil {
		return time.Local
	}
	return r.location
}

// DayRange 返回 at 所在业务日的半开区间，同对手计数与每日上限按它统计。
func (r Rules) DayRange(at time.Time) (time.Time, time.Time) {
	local := at.In(r.Location())
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, r.Location())
	return start, start.AddDate(0, 0, 1)
}

// SeasonQualification 返回赛季的达标门槛；非 V2 赛季没有门槛。
func (r Rules) SeasonQualification(season *model.Season) (minMatches, minOpponents int) {
	if !r.IsV2Season(season) {
		return 0, 0
	}
	return SeasonQualifyMinMatches, SeasonQualifyMinOpponents
}

const (
	// SeasonQualifyMinMatches 与 SeasonQualifyMinOpponents 是 V2 赛季参与名次的门槛；
	// 达标状态已持久化在 season_records.qualified，调整门槛需要迁移重算。
	SeasonQualifyMinMatches   = 5
	SeasonQualifyMinOpponents = 3
)
