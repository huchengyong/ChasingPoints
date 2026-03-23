# 赛事情报改造 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a new content-oriented event intelligence module for snooker, Chinese eight-ball, and Chinese nine-ball, and switch the current user-facing tournament entry points to this new module.

**Architecture:** Keep the existing `tournament` domain intact, then add a parallel `event-news` domain with its own database table, go-zero API contract, handlers, logic, and frontend pages. Reuse the existing home/list/detail page shells where it is cheap, but change data mapping and interaction semantics from “joinable tournament” to “readable event content”.

**Tech Stack:** UniApp Vue3 + Pinia + SCSS frontend, go-zero API service, Gorm + MySQL backend, existing admin frontend, goose migrations, goctl code generation.

---

### Task 1: Prepare backend contract for event intelligence

**Files:**
- Modify: `backend/chasing_points.api`
- Reference: `backend/internal/handler/routes.go`
- Reference: `app/api/tournament.js`

**Step 1: Add event-news types to `.api`**

Add new request and response types for:

- `EventNewsInfo`
- `GetEventNewsListReq`
- `GetEventNewsListResp`
- `GetEventNewsDetailReq`
- `GetEventNewsDetailResp`
- `GetFeaturedEventNewsResp`
- Admin create/update payloads

Mirror the project’s existing list/detail conventions as closely as possible without reusing tournament-specific payloads.

**Step 2: Add public routes**

Add public routes under a new prefix:

```api
@server (
    prefix: /api/event-news
    group:  eventnews
)
service chasing_points-api {
    @handler GetEventNewsList
    get /list (GetEventNewsListReq) returns (GetEventNewsListResp)

    @handler GetEventNewsDetail
    get /detail (GetEventNewsDetailReq) returns (GetEventNewsDetailResp)

    @handler GetFeaturedEventNews
    get /featured returns (GetFeaturedEventNewsResp)
}
```

**Step 3: Add admin routes**

Add admin-protected routes for create, update, publish, and delete or offline actions.

**Step 4: Generate go-zero code**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
goctl api go --api chasing_points.api --dir . --style go_zero --home ~/.goctl/default
```

Expected: generated handler, logic, and types compile without manual edits to generated files.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/chasing_points.api /Users/wisesearch/Projects/ChasingPoints/backend/internal
git commit -m "feat: add event news api contract"
```

### Task 2: Add event intelligence database model and migration

**Files:**
- Create: `backend/migrations/20260323xxxxxx_add_event_news_tables.sql`
- Modify: `backend/internal/model/`
- Modify: `backend/internal/svc/service_context.go`

**Step 1: Write the migration**

Create a goose migration with `Up` and `Down` sections for a new table, for example `event_news`.

Suggested columns:

- `id`
- `title`
- `slug`
- `game_type`
- `source_type`
- `source_name`
- `source_url`
- `cover_image`
- `summary`
- `content`
- `country`
- `city`
- `venue`
- `start_time`
- `end_time`
- `status`
- `stage_text`
- `result_text`
- `featured`
- `sort_time`
- `published`
- `published_at`
- `created_at`
- `updated_at`
- `deleted_at`

**Step 2: Add the Gorm model**

Create a new backend model file with JSON tags aligned to SQL columns and helper methods for:

- paginated list
- detail by id
- featured item lookup
- create
- update
- publish toggle
- delete or soft delete

**Step 3: Wire it into `ServiceContext`**

Register the new model in `backend/internal/svc/service_context.go`.

**Step 4: Run migration locally**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
./goose.sh up
```

Expected: migration applies successfully.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/migrations /Users/wisesearch/Projects/ChasingPoints/backend/internal/model /Users/wisesearch/Projects/ChasingPoints/backend/internal/svc/service_context.go
git commit -m "feat: add event news persistence layer"
```

### Task 3: Implement backend public list, detail, and featured logic

**Files:**
- Modify: generated `backend/internal/logic/eventnews/*.go`
- Modify: generated `backend/internal/handler/eventnews/*.go`
- Test: `backend/internal/logic/*_test.go`

**Step 1: Write failing tests for model-backed list behavior**

Cover at least:

- filter by `game_type`
- filter by `status`
- only published items appear in public list
- featured endpoint prefers `featured = 1`, then nearest active event

**Step 2: Run failing tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/... 
```

Expected: tests fail because event news logic is incomplete.

**Step 3: Implement list logic**

Map request params into model filters, normalize empty list responses, and ensure sorting is stable:

- featured first only where appropriate for featured endpoint
- list ordered by `sort_time DESC` or business-approved ordering

**Step 4: Implement detail logic**

Return a single published event item with all display fields needed by frontend.

**Step 5: Implement featured logic**

Return the single best homepage candidate.

**Step 6: Run tests again**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/...
```

Expected: event news tests pass.

**Step 7: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic /Users/wisesearch/Projects/ChasingPoints/backend/internal/handler
git commit -m "feat: implement public event news endpoints"
```

### Task 4: Implement admin event intelligence APIs

**Files:**
- Modify: generated `backend/internal/logic/eventnews/*.go`
- Modify: generated `backend/internal/handler/eventnews/*.go`
- Modify: `backend/internal/utils/` if admin auth helpers are needed

**Step 1: Write failing tests for admin create/update/publish behavior**

Cover:

- admin can create draft event
- admin can update fields
- admin can publish and unpublish
- non-admin is rejected

**Step 2: Implement minimal create logic**

Validate required fields:

- `title`
- `game_type`
- `status`
- `start_time` or `sort_time`

**Step 3: Implement update logic**

Allow patching the structured fields without exposing unsupported rich content features.

**Step 4: Implement publish and delete logic**

Prefer logical delete or publish toggle over destructive deletion.

**Step 5: Run backend tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/...
```

Expected: admin and public event news logic passes.

**Step 6: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/internal
git commit -m "feat: add admin event news management endpoints"
```

### Task 5: Add frontend API facade for event intelligence

**Files:**
- Create: `app/api/event-news.js`
- Modify: `app/api/AGENTS.md` if the repository requires module inventory updates

**Step 1: Write the API wrapper**

Add exported functions for:

- `getEventNewsList`
- `getEventNewsDetail`
- `getFeaturedEventNews`
- admin functions if admin frontend shares this package

**Step 2: Verify import style matches project conventions**

Follow the same `get/post` wrappers already used elsewhere.

**Step 3: Add or update small API smoke tests if present**

If the repo has API facade tests, extend them. If not, verify through page integration later.

**Step 4: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/api/event-news.js /Users/wisesearch/Projects/ChasingPoints/app/api/AGENTS.md
git commit -m "feat: add event news frontend api facade"
```

### Task 6: Switch homepage focus card to event intelligence

**Files:**
- Modify: `app/pages/index/index.vue`
- Modify: `app/utils/home-index.js`
- Test: `app/tests/home-index.test.mjs`

**Step 1: Write failing frontend tests for homepage selection**

Add tests for:

- featured endpoint or event list mapping
- excluding unpublished items
- status and time formatting for the focus card

**Step 2: Replace tournament request with event news request**

In homepage loading logic, stop using tournament list for the focus section and use new event-news data instead.

**Step 3: Rename and remap display fields**

The homepage card should display:

- status text
- time text
- title
- summary
- game type

It should no longer mention player count or join state.

**Step 4: Run frontend tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
npm test -- home-index.test.mjs
```

If no package test runner exists, run the existing local test command pattern used by the repo.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/pages/index/index.vue /Users/wisesearch/Projects/ChasingPoints/app/utils/home-index.js /Users/wisesearch/Projects/ChasingPoints/app/tests/home-index.test.mjs
git commit -m "feat: switch homepage focus to event news"
```

### Task 7: Convert tournament list page into event intelligence list page

**Files:**
- Modify: `app/subPages/tournament/index.vue`
- Modify: `app/subPages/tournament/index.scss`
- Modify: `app/pages.json` title text if needed

**Step 1: Rename page copy**

Change visible copy from “赛事大厅” to “赛事情报”.

**Step 2: Change data source**

Use `getEventNewsList` instead of `getTournamentList`.

**Step 3: Replace card fields**

Remove:

- 报名进度
- 人数上限
- 参赛相关文案

Add:

- 状态
- 项目
- 时间
- 地点
- 当前阶段
- 最新赛果摘要
- 来源标签

**Step 4: Keep existing list UX**

Retain pull-to-refresh, infinite scroll, empty state, and top hero structure where it still fits.

**Step 5: Verify page manually**

Run the app and confirm:

- filters work
- list loads
- empty state renders
- detail jump works

**Step 6: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament/index.vue /Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament/index.scss /Users/wisesearch/Projects/ChasingPoints/app/pages.json
git commit -m "feat: convert tournament list page to event intelligence"
```

### Task 8: Convert tournament detail page into event intelligence detail page

**Files:**
- Modify: `app/subPages/tournament/detail.vue`
- Modify: `app/subPages/tournament/detail.scss`

**Step 1: Replace tournament detail request**

Use `getEventNewsDetail`.

**Step 2: Remove action bar**

Delete or disable:

- 报名参加
- 取消报名
- 查看对阵图

**Step 3: Add event content sections**

Show:

- title
- status
- game type
- time
- venue/location
- summary
- stage text
- result text
- source info
- source link button

**Step 4: Keep visual consistency**

Reuse the current hero/info card layout where it saves time, but update copy and visual hierarchy for content reading.

**Step 5: Manual verification**

Check:

- upcoming event with empty result
- live event with stage text
- finished event with result summary

**Step 6: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament/detail.vue /Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament/detail.scss
git commit -m "feat: convert tournament detail page to event detail"
```

### Task 9: Add admin frontend event intelligence management

**Files:**
- Explore and modify the relevant files under `admin/`
- Likely create: list page, form page, API file, route entry

**Step 1: Explore admin structure**

Locate:

- router configuration
- admin API facade location
- list/table page conventions
- form conventions

**Step 2: Write a minimal list page**

Show:

- title
- game type
- status
- featured flag
- published flag
- start time
- updated time

**Step 3: Write create/edit form**

Include only structured fields from the approved design.

**Step 4: Add actions**

Support:

- create
- edit
- publish/unpublish
- delete or offline

**Step 5: Manual verification**

Confirm admin can create one record and it appears in frontend list/detail after publish.

**Step 6: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/admin
git commit -m "feat: add admin event news management"
```

### Task 10: Seed representative content and validate end-to-end

**Files:**
- Optional create: seed script or SQL
- Optional modify: admin fixtures or local seed helpers

**Step 1: Prepare three seed items**

Create at least one item per target game type:

- 斯诺克
- 中式八球
- 中式九球

**Step 2: Verify homepage**

Confirm homepage focus renders meaningful copy from seeded content.

**Step 3: Verify list filtering**

Confirm the three game type filters return the right cards.

**Step 4: Verify detail**

Confirm source link, stage text, and result text all render properly.

**Step 5: Run project verification**

Backend:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./...
```

Frontend:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
npm test
```

If there is no stable frontend test runner, run the project’s actual available verification commands and manually inspect the changed pages.

**Step 6: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints
git commit -m "test: verify event intelligence end to end"
```

### Task 11: Clean up copy and de-emphasize old tournament semantics

**Files:**
- Modify: `app/pages/index/index.vue`
- Modify: `app/pages/user/index.vue`
- Modify: any remaining visible text under `app/subPages/tournament/`
- Reference: `app/GEMINI.md`, `AGENTS.md` files if module descriptions need sync

**Step 1: Search for outdated copy**

Search for:

- 赛事大厅
- 报名参加
- 取消报名
- 赛事中心

Update user-facing text that no longer matches the new experience.

**Step 2: Update lightweight docs if required**

If repository inventories mention tournament pages as joinable pages, update those references to reflect the new user-facing module behavior.

**Step 3: Manual scan**

Make sure users cannot accidentally reach a dead join/leave flow from the main UX.

**Step 4: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app /Users/wisesearch/Projects/ChasingPoints/backend /Users/wisesearch/Projects/ChasingPoints/*AGENTS.md /Users/wisesearch/Projects/ChasingPoints/app/GEMINI.md
git commit -m "chore: align copy with event intelligence module"
```

### Task 12: Final verification and release readiness

**Files:**
- No new files required

**Step 1: Run backend verification**

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./...
```

Expected: all backend tests pass.

**Step 2: Run frontend verification**

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
npm test
```

Expected: relevant frontend tests pass, or documented exceptions if no runner exists.

**Step 3: Manual smoke test**

Check:

- homepage focus card
- event list page
- event detail page
- admin create/edit/publish flow

**Step 4: Prepare release note**

Summarize:

- old join-based tournament entrance replaced with content-focused event intelligence
- supported game types
- admin maintenance workflow

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints
git commit -m "chore: finalize event intelligence rollout"
```
