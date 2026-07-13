# Event News Schedule Mode Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rebuild the current event news module into a schedule-oriented model where the list shows event parents and the detail page shows ordered stage schedules under each event.

**Architecture:** Replace the existing single-table `event_news` domain with two tables: one for event parents and one for stage schedules. Update the go-zero contract first, then regenerate code, implement model and logic aggregation, and finally adapt admin and app UIs to operate on event-level records with nested stage management.

**Tech Stack:** Go 1.25, go-zero API, Gorm, goose migrations, Vue 3 + Vite + TypeScript admin, UniApp Vue 3 app, existing request wrappers and logic tests.

---

### Task 1: Replace the API contract with event + stage types

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/chasing_points.api`
- Reference: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/eventnews/`

**Step 1: Write the failing contract shape in `.api`**

Add new types for:

- `EventNewsStageInfo`
- `EventNewsEventInfo`
- `EventNewsListItem`
- `GetEventNewsDetailResp` with `event` and `stages`
- admin stage create/update/delete payloads

Remove old single-record assumptions like `stage_text` and `result_text` living directly on every list/detail item.

**Step 2: Update public routes**

Keep:

```api
get /list
get /detail
get /featured
```

But change request/response contracts so `/detail` accepts `event_id` and returns grouped data.

**Step 3: Update admin routes**

Keep event-level routes:

```api
get /list
post /create
post /update
post /publish
post /delete
```

Add stage-level routes:

```api
post /stage/create
post /stage/update
post /stage/delete
```

**Step 4: Run goctl generation**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
goctl api go --api chasing_points.api --dir . --style go_zero --home ~/.goctl/default
```

Expected: generated handlers, logic stubs, and `types.go` reflect the new event + stage contracts.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/chasing_points.api /Users/wisesearch/Projects/ChasingPoints/backend/internal
git commit -m "feat: replace event news api with schedule contract"
```

### Task 2: Replace persistence with event and stage tables

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/migrations/20260325xxxxxx_replace_event_news_with_schedule_tables.sql`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/model/event_news.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/model/event_news_stage.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/testsupport/`

**Step 1: Write the failing migration**

Drop the old `event_news` table and create:

- `event_news_events`
- `event_news_stages`

Ensure `event_news_stages.event_id` is indexed and foreign-keyed for cascade delete where supported.

**Step 2: Write model tests or schema preparation updates first**

Update event-news test schema helpers so tests create the new two-table structure instead of the old single-table structure.

**Step 3: Implement the event model**

Expose methods for:

- create event
- update event
- find event by id
- find published event by id
- list events with filters
- update publish state
- soft delete event

**Step 4: Implement the stage model**

Expose methods for:

- create stage
- update stage
- delete stage
- list stages by `event_id`
- delete stages by `event_id`
- fetch latest/current stage summary for list composition

**Step 5: Wire both models into service context**

Update `/Users/wisesearch/Projects/ChasingPoints/backend/internal/svc/service_context.go` so logic can use both models.

**Step 6: Run backend tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/...
```

Expected: failing tests now point at missing aggregation/business logic rather than schema setup.

**Step 7: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/migrations /Users/wisesearch/Projects/ChasingPoints/backend/internal/model /Users/wisesearch/Projects/ChasingPoints/backend/internal/testsupport /Users/wisesearch/Projects/ChasingPoints/backend/internal/svc/service_context.go
git commit -m "feat: add event schedule persistence layer"
```

### Task 3: Implement public list, detail, and featured aggregation

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/eventnews/helpers.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/eventnews/get_event_news_list_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/eventnews/get_event_news_detail_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/eventnews/get_featured_event_news_logic.go`
- Test: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/eventnews/event_news_logic_test.go`

**Step 1: Write failing tests for grouped responses**

Cover at least:

- list only returns published events
- list item exposes `stage_count`
- list item exposes latest/current stage summary
- detail returns one event with ordered stages
- featured returns one event with stage preview

**Step 2: Run the focused test file**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic/eventnews -run Test -v
```

Expected: FAIL because logic still maps single-record payloads.

**Step 3: Implement list aggregation**

Compose each list item from the event row plus its stage rows:

- pick current stage from active stage if present
- otherwise pick the nearest or latest stage by `stage_order` and `sort_time`
- calculate `stage_count`

**Step 4: Implement detail aggregation**

Load the published event, then load all stages under `event_id`, sort them by `stage_order ASC, id ASC`, and return grouped payload.

**Step 5: Implement featured aggregation**

Choose the best featured event, then attach stage summary fields required by the homepage card.

**Step 6: Run tests again**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic/eventnews -run Test -v
```

Expected: PASS for public schedule-mode logic tests.

**Step 7: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/eventnews
git commit -m "feat: implement public event schedule aggregation"
```

### Task 4: Implement admin event and stage management APIs

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/admin/admin_event_news_helpers.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/admin/admin_create_event_news_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/admin/admin_update_event_news_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/admin/admin_get_event_news_list_logic.go`
- Create: generated stage admin logic files under `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/admin/`
- Test: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/admin/admin_event_news_logic_test.go`

**Step 1: Write failing admin tests**

Cover:

- create event as draft
- update event fields
- publish/unpublish event
- delete event cascades stage deletion
- create stage under an event
- update stage fields
- delete stage

**Step 2: Run the focused admin tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic/admin -run EventNews -v
```

Expected: FAIL because admin flows only support the old single-record model.

**Step 3: Implement event-level validation**

Validate:

- `title`
- `game_type`
- `start_time` or `sort_time`

**Step 4: Implement stage-level validation**

Validate:

- `event_id > 0`
- `stage_name` not empty
- `stage_order` provided
- if both stage times exist, end time is not before start time

**Step 5: Implement stage CRUD logic**

Use the new stage model to create, update, and soft delete stage rows.

**Step 6: Update admin list response mapping**

Return event rows with summary fields like:

- `current_stage_text`
- `latest_result_text`
- `stage_count`

**Step 7: Run admin tests again**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic/admin -run EventNews -v
```

Expected: PASS for event and stage admin flows.

**Step 8: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/admin
git commit -m "feat: add admin event schedule management"
```

### Task 5: Adapt admin API types and page to event-level editing

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/admin/src/api/event-news.ts`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/admin/src/views/event-news/index.vue`

**Step 1: Update TypeScript API contracts**

Split types into:

- event list item
- event detail item
- stage item
- stage create/update payloads

**Step 2: Build failing UI integration mentally against new API**

Confirm the existing page cannot compile cleanly because it expects `stage_text` and `result_text` directly on the editable record.

**Step 3: Refactor the event list table**

Show:

- title
- game type
- event status
- current stage
- stage count
- latest result
- published state
- updated time

**Step 4: Refactor the dialog form**

Keep event base fields in the top portion and add a stage management block below with:

- stage table
- add stage form row or nested dialog
- edit/delete actions for each stage

**Step 5: Wire save flows**

Event save stays event-level.
Stage save calls stage create/update APIs separately during edit mode.

**Step 6: Run admin build**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/admin
npm run build
```

Expected: PASS with no TypeScript errors.

**Step 7: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/admin/src/api/event-news.ts /Users/wisesearch/Projects/ChasingPoints/admin/src/views/event-news/index.vue
git commit -m "feat: rebuild admin event news page for schedule mode"
```

### Task 6: Adapt app list, detail, and homepage featured mapping

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/api/event-news.js`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament/index.vue`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament/detail.vue`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/utils/home-index.js`

**Step 1: Update app API semantics**

Change detail requests to send `event_id` and expect:

- `event`
- `stages`

**Step 2: Refactor list normalization**

Normalize each event list item using event summary fields rather than direct stage fields on every record.

**Step 3: Refactor detail normalization**

Map event base info and a stage array for rendering in the detail page.

**Step 4: Update detail UI**

Replace the old single “当前阶段 / 最新赛果” block with an ordered stage schedule list.

**Step 5: Update homepage featured mapping**

Make `normalizeFeaturedEventNews` consume the new featured payload without assuming the old single-record shape.

**Step 6: Run app logic tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/*.test.mjs
```

Expected: PASS for existing logic tests; manually verify event pages for render/runtime issues.

**Step 7: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/api/event-news.js /Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament/index.vue /Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament/detail.vue /Users/wisesearch/Projects/ChasingPoints/app/utils/home-index.js
git commit -m "feat: switch app event news pages to schedule mode"
```

### Task 7: Run end-to-end verification and clean up

**Files:**
- Verify only

**Step 1: Run backend tests**

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./...
```

Expected: PASS.

**Step 2: Run admin build**

```bash
cd /Users/wisesearch/Projects/ChasingPoints/admin
npm run build
```

Expected: PASS.

**Step 3: Run app tests**

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/*.test.mjs
```

Expected: PASS.

**Step 4: Spot-check generated and manual files**

Confirm:

- no manual edits to goctl-generated files
- app detail uses `event_id`
- admin list no longer depends on per-record `stage_text`

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints
git commit -m "feat: migrate event news to schedule mode"
```
