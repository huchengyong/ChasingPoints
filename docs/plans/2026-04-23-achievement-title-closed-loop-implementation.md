# Achievement Title Closed Loop Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a real closed loop for special match records, account achievements, and title rewards so finished matches, tournament settlement, and season rebuild can all update user progress and grant traceable titles.

**Architecture:** Keep the existing public API surface mostly stable and fix the closed loop behind the scenes. Add actor-aware match special-record storage, introduce an idempotent achievement progress event table, centralize progress aggregation and title granting in the achievement domain, then integrate that domain into match finish, tournament join/finish, and season rebuild flows.

**Tech Stack:** go-zero API, Gorm/MySQL migrations via goose, Go unit/integration-style tests with sqlite, existing UniApp frontend pages and API wrappers, `node --test` for frontend logic verification.

---

### Task 1: Add the schema needed for the closed loop

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/migrations/20260423110000_achievement_title_closed_loop.sql`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/migrations/migration_files_test.go`

**Step 1: Write the failing migration test**

Add assertions in `migration_files_test.go` that the new migration contains:

- `achievement_progress_events`
- `actor`
- `source_ref_id`
- `source_ref_name`
- `reward_granted`
- `uk_match_actor_achievement`
- `uk_user_source_metric`
- `uk_user_title_source`

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./migrations
```

Expected: FAIL because the migration file does not exist yet.

**Step 3: Write the migration**

Create a migration that:

- alters `match_achievements` to add `actor`
- replaces the old unique key with `(match_id, actor, achievement_type)`
- alters `achievements` to add `game_type`, `metric_key`, `progress_mode`, reward-title metadata, `sort`, `status`, `updated_at`
- alters `user_achievements` to add `reward_granted`, `reward_granted_at`, `updated_at`
- alters `user_titles` to add `title_key`, `source_type`, `source_ref_id`, `source_ref_name`, `granted_by_achievement_id`, `equipped_at`, `granted_at`
- creates `achievement_progress_events`
- uses MySQL-compatible SQL only

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./migrations
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/migrations/20260423110000_achievement_title_closed_loop.sql backend/migrations/migration_files_test.go
git commit -m "feat: 增加成就称号闭环表结构"
```

### Task 2: Upgrade the model layer for actor-based records, progress events, and traceable titles

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/model/achievement.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/model/match.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/model/achievement_closed_loop_test.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/svc/service_context.go`

**Step 1: Write the failing model tests**

Cover:

- `MatchAchievement` includes `Actor`
- `Achievement` includes `metric_key`, `progress_mode`, reward-title fields
- `UserAchievement` includes reward-grant fields
- `UserTitle` includes source-ref and grant fields
- progress event model supports duplicate-safe lookup/upsert
- `ServiceContext` wires any new models without auto-migrating

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/model -run Achievement
```

Expected: FAIL because the new fields and helpers do not exist yet.

**Step 3: Write minimal implementation**

Implement:

- upgraded structs in `achievement.go`
- `AchievementProgressEvent` model and helpers
- actor-aware `SaveAchievementWithTx`
- source-traceable `UserTitle` model methods
- any required `ServiceContext` fields or constructor changes

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/model -run Achievement
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/model/achievement.go backend/internal/model/match.go backend/internal/model/achievement_closed_loop_test.go backend/internal/svc/service_context.go
git commit -m "feat: 补齐成就称号领域模型"
```

### Task 3: Add the achievement progress and title grant services

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/achievement/progress_service.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/achievement/title_grant_service.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/achievement/metric_defs.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/achievement/progress_service_test.go`

**Step 1: Write the failing service tests**

Cover:

- sum-mode metrics aggregate by total
- max-mode metrics aggregate by maximum
- duplicate `(user, source_type, source_id, metric_key)` events are idempotent
- unlocking a threshold flips `user_achievements.unlocked`
- reward titles are granted once
- title grant uses `source_type`, `source_ref_id`, `source_ref_name`

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/logic/achievement -run Progress
```

Expected: FAIL because the new services do not exist yet.

**Step 3: Write minimal implementation**

Implement:

- metric-key constants
- progress event append helper
- user progress refresh helper
- unlock detection helper
- achievement-title grant helper
- category and game-type validation kept minimal

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/logic/achievement -run Progress
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/logic/achievement/progress_service.go backend/internal/logic/achievement/title_grant_service.go backend/internal/logic/achievement/metric_defs.go backend/internal/logic/achievement/progress_service_test.go
git commit -m "feat: 增加成就进度与称号发放服务"
```

### Task 4: Seed the minimum achievement definitions

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/migrations/20260423113000_seed_minimum_achievements.sql`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/migrations/migration_files_test.go`

**Step 1: Write the failing migration test**

Assert the seed migration contains:

- `match_10`
- `wins_100`
- `streak_10`
- `break_147_1`
- `tournament_champion_1`

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./migrations
```

Expected: FAIL because the seed migration does not exist yet.

**Step 3: Write the seed migration**

Insert the 12 minimum achievement definitions agreed in the design doc, using canonical category codes and reward-title metadata.

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./migrations
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/migrations/20260423113000_seed_minimum_achievements.sql backend/migrations/migration_files_test.go
git commit -m "feat: 初始化最小成就定义"
```

### Task 5: Integrate actor-aware special records and achievement progress into match flows

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/match/end_round_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/match/finish_match_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/model/match.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/match/finish_match_logic_test.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/match/finish_match_achievement_closed_loop_test.go`

**Step 1: Write the failing match-flow tests**

Cover:

- saved special records include the correct `actor`
- invalid matches do not produce progress events
- both players receive `matches_total`
- winner receives `wins_total`
- `max_win_streak` is refreshed from ranking snapshot
- special-record metrics are written for the correct player only
- re-finishing or replaying the same match does not double-count

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/logic/match -run Achievement
```

Expected: FAIL because the closed-loop integration is missing.

**Step 3: Write minimal implementation**

Implement:

- actor-aware `SaveAchievementWithTx` call sites in `end_round_logic.go`
- effective-match guards in `finish_match_logic.go`
- post-settlement progress-event writes
- one refresh call per affected user after all event writes

Keep the match finish transaction focused: write progress events and refresh snapshots only after the base match settlement is stable.

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/logic/match -run Achievement
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/logic/match/end_round_logic.go backend/internal/logic/match/finish_match_logic.go backend/internal/model/match.go backend/internal/logic/match/finish_match_logic_test.go backend/internal/logic/match/finish_match_achievement_closed_loop_test.go
git commit -m "feat: 打通比赛成就进度闭环"
```

### Task 6: Integrate tournament join and finish into achievement and title granting

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/tournament/join_tournament_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/tournament/finish_tournament_logic.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/tournament/achievement_bridge.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/tournament/tournament_achievement_test.go`

**Step 1: Write the failing tournament tests**

Cover:

- successful join produces `tournament_join_total`
- duplicate join stays idempotent
- tournament finish writes `tournament_finish_total`
- champion also receives `tournament_champion_total`
- final-rank-based titles are granted once
- replaying finish does not duplicate titles

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/logic/tournament -run Achievement
```

Expected: FAIL because tournament achievement integration is missing.

**Step 3: Write minimal implementation**

Implement:

- progress event write after join succeeds
- finish-time event generation based on `final_rank`
- tournament-title grant helper calls
- title naming based on tournament name and final rank

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/logic/tournament -run Achievement
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/logic/tournament/join_tournament_logic.go backend/internal/logic/tournament/finish_tournament_logic.go backend/internal/logic/tournament/tournament_achievement_test.go
git commit -m "feat: 打通赛事成就与称号发放"
```

### Task 7: Integrate season-title granting into season rebuild

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/rank_rebuild_service.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/model/user_ranking.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/rank_rebuild_service_test.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/achievement/season_title_service.go`

**Step 1: Write the failing rebuild tests**

Cover:

- season records still compute `final_rank` correctly
- after season-record creation, rank rebuild grants season titles for rank 1/2/3/top 10
- rerunning rebuild does not duplicate the same season titles
- source references point to the correct `season_id`

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/logic -run RankRebuild
```

Expected: FAIL because season-title grant integration is missing.

**Step 3: Write minimal implementation**

Implement:

- a season-title grant helper based on `season_records.final_rank`
- one rebuild hook after `SeasonRecordModel.CreateBatchWithTx`
- season title naming using season name and game-type label
- tx-aware rank-change queries so season records can read the just-replayed ranking logs before the rebuild transaction commits

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/logic -run RankRebuild
```

Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/logic/rank_rebuild_service.go backend/internal/logic/rank_rebuild_service_test.go backend/internal/logic/achievement/season_title_service.go
git commit -m "feat: 增加赛季称号发放链路"
```

### Task 8: Fix frontend naming, category mapping, and title refresh behavior

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/achievement/index.vue`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/achievement/detail.vue`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/achievement/titles.vue`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/match/matchResult.vue`
- Create: `/Users/wisesearch/Projects/ChasingPoints/app/utils/achievement-page.js`
- Create: `/Users/wisesearch/Projects/ChasingPoints/app/tests/achievement-page.test.mjs`

**Step 1: Write the failing frontend tests**

Cover:

- backend category codes map to the expected Chinese labels
- category tabs filter correctly by code instead of Chinese raw value
- match result empty copy says “暂无特殊战绩”
- returning from titles page refreshes the current equipped title on the index page

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/achievement-page.test.mjs
```

Expected: FAIL because the current pages still use mixed naming and raw category values.

**Step 3: Write minimal implementation**

Implement:

- a small shared category-code-to-label map
- display mapping in both list and detail pages
- `onShow` refresh on the achievement index page so equipped title stays current
- copy cleanup on the match result page

**Step 4: Run test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/achievement-page.test.mjs
```

Expected: PASS

**Step 5: Commit**

```bash
git add app/subPages/achievement/index.vue app/subPages/achievement/detail.vue app/subPages/achievement/titles.vue app/subPages/match/matchResult.vue app/utils/achievement-page.js app/tests/achievement-page.test.mjs
git commit -m "feat: 统一成就称号页面命名与分类映射"
```

### Task 9: Run regression verification across the closed loop

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-04-23-achievement-title-closed-loop-design.md`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-04-23-achievement-title-closed-loop-implementation.md`

**Step 1: Run backend migration tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./migrations
```

Expected: PASS

**Step 2: Run backend model and logic tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/gocache-achievement-plan go test ./internal/model ./internal/logic/achievement ./internal/logic/match ./internal/logic/tournament ./internal/logic -run 'Achievement|RankRebuild|Season|Tournament'
```

Expected: PASS

**Step 3: Run frontend tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/*.test.mjs
```

Expected: PASS

**Step 4: Update the design docs if reality changed**

If implementation forced any deviation from the design, update both plan docs before closing the branch.

**Step 5: Commit**

```bash
git add docs/plans/2026-04-23-achievement-title-closed-loop-design.md docs/plans/2026-04-23-achievement-title-closed-loop-implementation.md
git commit -m "docs: 校准成就称号闭环方案与实现计划"
```

Plan complete and saved to `docs/plans/2026-04-23-achievement-title-closed-loop-implementation.md`. Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
