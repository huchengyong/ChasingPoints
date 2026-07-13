# Notification Preferences And Message Center Closure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add server-backed notification preferences with five real business-type switches so notification settings, message center list, unread count, user WebSocket refresh, and push dispatch all follow one consistent backend-controlled decision.

**Architecture:** Add a dedicated user notification preference table plus read/write notification preference APIs, then funnel all business notification creation through one shared backend dispatch service that checks preferences before writing notifications, sending push, or emitting `notification_update`. On app side, replace the local placeholder settings page with server-backed switches and align the message-center type map with the real backend notification types.

**Tech Stack:** go-zero API with goctl-generated handlers/types, Gorm models and migration SQL, shared backend logic helpers/services, UniApp Vue 3 app frontend, Node `node:test` app tests, backend Go tests with explicit schema prep.

---

### Task 1: Add notification preference schema and API contract

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/chasing_points.api`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/migrations/20260415110000_add_user_notification_preferences_table.sql`
- Generated: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/types/types.go`
- Generated: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/handler/routes.go`
- Generated: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/handler/notification/get_notification_preferences_handler.go`
- Generated: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/handler/notification/save_notification_preferences_handler.go`
- Test: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification/notification_preferences_logic_test.go`

**Step 1: Write the failing backend tests**

Create a focused notification-preferences logic test skeleton that references new DTOs and new handlers.

```go
func TestGetNotificationPreferences_DefaultsAllEnabled(t *testing.T) {}

func TestSaveNotificationPreferences_PersistsSwitches(t *testing.T) {}
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/go-build go test ./internal/logic/notification -run NotificationPreferences
```

Expected: FAIL because request/response DTOs and handlers do not exist yet.

**Step 3: Extend API contract and migration**

Add to `backend/chasing_points.api`:

```api
type NotificationPreferences {
	MatchResultEnabled   bool `json:"match_result_enabled"`
	FriendRequestEnabled bool `json:"friend_request_enabled"`
	ChallengeEnabled     bool `json:"challenge_enabled"`
	TournamentEnabled    bool `json:"tournament_enabled"`
	FollowEnabled        bool `json:"follow_enabled"`
}
```

Add:

- `GET /api/notification/preferences`
- `POST /api/notification/preferences`

Add migration for table:

```sql
CREATE TABLE IF NOT EXISTS `user_notification_preferences` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `match_result_enabled` TINYINT NOT NULL DEFAULT 1,
  `friend_request_enabled` TINYINT NOT NULL DEFAULT 1,
  `challenge_enabled` TINYINT NOT NULL DEFAULT 1,
  `tournament_enabled` TINYINT NOT NULL DEFAULT 1,
  `follow_enabled` TINYINT NOT NULL DEFAULT 1,
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`)
);
```

**Step 4: Regenerate goctl outputs**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
goctl api go --api chasing_points.api --dir . --style go_zero --home ~/.goctl/default
```

Expected: generated handlers and DTOs appear for notification preferences.

**Step 5: Run tests again**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/go-build go test ./internal/logic/notification -run NotificationPreferences
```

Expected: FAIL in logic implementation, but compile no longer fails because DTOs are missing.

**Step 6: Commit**

```bash
git add -A /Users/wisesearch/Projects/ChasingPoints/backend/chasing_points.api /Users/wisesearch/Projects/ChasingPoints/backend/migrations/20260415110000_add_user_notification_preferences_table.sql /Users/wisesearch/Projects/ChasingPoints/backend/internal/types/types.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/handler/routes.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/handler/notification/get_notification_preferences_handler.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/handler/notification/save_notification_preferences_handler.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification/notification_preferences_logic_test.go
git commit -m "新增通知偏好接口与数据契约" -m "为通知设置闭环补齐服务端持久化入口，先定义通知偏好表与读写接口，让前端设置页可以脱离本地占位存储。" -m "Constraint: 默认行为必须兼容现有全量通知策略" -m "Confidence: high" -m "Scope-risk: moderate" -m "Directive: 不要在前端继续把本地存储当作通知偏好真源" -m "Tested: focused notification logic compile test and goctl generation" -m "Not-tested: end-to-end API invocation"
```

### Task 2: Implement backend notification preference model and logic

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/model/user_notification_preference.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification/get_notification_preferences_logic.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification/save_notification_preferences_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/svc/service_context.go`
- Test: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/model/user_notification_preference_test.go`
- Test: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/testsupport/notification_schema.go`

**Step 1: Write the failing model tests**

Cover:

- no row -> returns all-enabled defaults
- save first row
- update existing row
- read saved booleans correctly

Example expectation:

```go
prefs, err := model.GetByUserIdOrDefault(1001)
if !prefs.MatchResultEnabled || !prefs.FriendRequestEnabled {
	t.Fatalf("expected defaults enabled")
}
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/go-build go test ./internal/model -run UserNotificationPreference
```

Expected: FAIL because model helper does not exist.

**Step 3: Write minimal implementation**

Create model struct and helpers:

```go
type UserNotificationPreference struct {
	Id                   int64
	UserId               int64
	MatchResultEnabled   bool
	FriendRequestEnabled bool
	ChallengeEnabled     bool
	TournamentEnabled    bool
	FollowEnabled        bool
}
```

Add:

- `FindByUserId`
- `GetByUserIdOrDefault`
- `Upsert`

Wire model into `ServiceContext`.

Implement notification logic:

- get logic returns defaults when no row exists
- save logic upserts the row and returns latest server state

**Step 4: Run tests to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/go-build go test ./internal/model -run UserNotificationPreference
GOCACHE=/tmp/go-build go test ./internal/logic/notification -run NotificationPreferences
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A /Users/wisesearch/Projects/ChasingPoints/backend/internal/model/user_notification_preference.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification/get_notification_preferences_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification/save_notification_preferences_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/svc/service_context.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/model/user_notification_preference_test.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/testsupport/notification_schema.go
git commit -m "实现用户通知偏好读写逻辑" -m "补齐通知偏好模型与获取/保存逻辑，确保没有配置记录时也能稳定返回默认全开配置。" -m "Constraint: migration 管理 schema，测试需显式准备表结构" -m "Confidence: high" -m "Scope-risk: narrow" -m "Directive: 默认值必须在 model 层统一兜底，避免 logic 和前端各写一套" -m "Tested: go test ./internal/model -run UserNotificationPreference; go test ./internal/logic/notification -run NotificationPreferences" -m "Not-tested: concurrent save race under production MySQL load"
```

### Task 3: Add shared notification dispatch service and migrate notification producers

**Files:**
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification_dispatch_service.go`
- Create: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification_dispatch_service_test.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/friend/send_friend_request_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/friend/accept_friend_request_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/challenge/send_challenge_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/challenge/accept_challenge_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/challenge/reject_challenge_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/follow/follow_user_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/tournament/join_tournament_logic.go`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/match/finish_match_logic.go`

**Step 1: Write the failing dispatch tests**

Cover:

- preference disabled -> no notification row created
- preference disabled -> no `notification_update` send
- preference disabled -> no push send attempt
- preference enabled -> row created with preserved title/content/data

Sketch:

```go
svc.Dispatch(NotificationDispatchInput{
	UserId: 88,
	Type: "friend_request",
	Title: "收到好友申请",
})
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/go-build go test ./internal/logic -run NotificationDispatch
```

Expected: FAIL because dispatch service does not exist.

**Step 3: Write minimal implementation**

Create shared input type:

```go
type NotificationDispatchInput struct {
	UserId       int64
	Type         string
	Title        string
	Content      string
	Data         *string
	PushTitle    string
	PushContent  string
	PushData     map[string]interface{}
	WSCategory   string
}
```

Service flow:

1. load preferences by user id
2. map `Type` -> enabled field
3. if disabled, return without side effects
4. create notification row
5. send push when token exists
6. emit `notification_update` when category configured

Then replace direct `NotificationModel.Create(...)` calls in all producer logics with the shared service.

**Step 4: Run focused backend tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/go-build go test ./internal/logic -run NotificationDispatch
GOCACHE=/tmp/go-build go test ./internal/logic/friend -run 'SendFriendRequest|AcceptFriendRequest'
GOCACHE=/tmp/go-build go test ./internal/logic/challenge -run 'SendChallenge|AcceptChallenge|RejectChallenge'
GOCACHE=/tmp/go-build go test ./internal/logic/match -run FinishMatch
```

Expected: PASS, and legacy message title/content expectations still hold.

**Step 5: Commit**

```bash
git add -A /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification_dispatch_service.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification_dispatch_service_test.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/friend/send_friend_request_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/friend/accept_friend_request_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/challenge/send_challenge_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/challenge/accept_challenge_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/challenge/reject_challenge_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/follow/follow_user_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/tournament/join_tournament_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/match/finish_match_logic.go
git commit -m "收口通知分发并接入偏好判断" -m "把分散在好友、挑战、关注、赛事、对局中的通知创建统一收口到共享分发服务，避免前端关掉开关后后端仍继续写库和推送。" -m "Constraint: 需保持现有通知标题、内容、跳转 payload 行为不变" -m "Rejected: 前端收到后再过滤 | 未读数和 push 仍不闭环" -m "Confidence: high" -m "Scope-risk: broad" -m "Directive: 后续新增通知类型必须先补偏好映射，再接入分发服务" -m "Tested: focused notification dispatch and domain logic go tests" -m "Not-tested: real UniPush external delivery"
```

### Task 4: Replace app-side local placeholder settings with server-backed switches

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/api/notification.js`
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/user/notification.vue`
- Test: `/Users/wisesearch/Projects/ChasingPoints/app/tests/notification-settings.test.mjs`

**Step 1: Write the failing app tests**

Assert:

- settings page references five real keys
- settings page imports and uses notification API methods
- local `notification_settings` is not sole source of truth anymore

Example:

```js
assert.match(source, /matchResultEnabled/)
assert.match(source, /friendRequestEnabled/)
assert.match(source, /getNotificationPreferences/)
assert.match(source, /saveNotificationPreferences/)
```

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/notification-settings.test.mjs
```

Expected: FAIL because page still uses local-only settings.

**Step 3: Write minimal implementation**

Add API methods:

```js
export const getNotificationPreferences = () => get('/api/notification/preferences')
export const saveNotificationPreferences = (data) => post('/api/notification/preferences', data)
```

Update page state to:

```js
const preferences = reactive({
  matchResultEnabled: true,
  friendRequestEnabled: true,
  challengeEnabled: true,
  tournamentEnabled: true,
  followEnabled: true
})
```

Page behavior:

- `onMounted` / `onShow` loads server preferences
- switch change sends save request
- save fail rolls UI state back

**Step 4: Run app test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/notification-settings.test.mjs
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A /Users/wisesearch/Projects/ChasingPoints/app/api/notification.js /Users/wisesearch/Projects/ChasingPoints/app/subPages/user/notification.vue /Users/wisesearch/Projects/ChasingPoints/app/tests/notification-settings.test.mjs
git commit -m "把通知管理页改为服务端真实配置" -m "设置页不再只写本地缓存，而是直接读取和保存服务端通知偏好，并把开关粒度改成真实业务通知类型。" -m "Constraint: 用户升级后默认行为必须与现状一致" -m "Confidence: high" -m "Scope-risk: moderate" -m "Directive: 不要再把展示文案和通知类型做粗粒度错配" -m "Tested: node --test tests/notification-settings.test.mjs" -m "Not-tested: real device switch interaction"
```

### Task 5: Align message center type mapping with real notification types

**Files:**
- Modify: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/notification/index.vue`
- Test: `/Users/wisesearch/Projects/ChasingPoints/app/tests/notification-center-types.test.mjs`

**Step 1: Write the failing app tests**

Assert message center maps:

- `challenge`
- `tournament`
- `friend_request`
- `follow`
- `match_result`

and keeps a default fallback.

**Step 2: Run test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/notification-center-types.test.mjs
```

Expected: FAIL because `follow` and `match_result` are not explicitly mapped.

**Step 3: Write minimal implementation**

Add icon / class mapping:

```js
const map = {
  challenge: '🎯',
  tournament: '🏆',
  friend_request: '👥',
  follow: '⭐',
  match_result: '🏁'
}
```

Add matching style classes for:

- `.type-follow`
- `.type-match_result`

Keep unknown type fallback as `🔔`.

**Step 4: Run app test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/notification-center-types.test.mjs
```

Expected: PASS

**Step 5: Commit**

```bash
git add -A /Users/wisesearch/Projects/ChasingPoints/app/subPages/notification/index.vue /Users/wisesearch/Projects/ChasingPoints/app/tests/notification-center-types.test.mjs
git commit -m "对齐消息中心通知类型映射" -m "补齐 follow 和 match_result 的显式展示映射，避免真实通知只能走默认图标和样式兜底。" -m "Constraint: 旧未知类型仍需保留默认兜底展示" -m "Confidence: high" -m "Scope-risk: narrow" -m "Directive: 前端类型映射必须与后端真实发消息类型同步维护" -m "Tested: node --test tests/notification-center-types.test.mjs" -m "Not-tested: visual regression on all device widths"
```

### Task 6: Run full verification and cleanup pass

**Files:**
- Verify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/notification/*.go`
- Verify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/friend/*.go`
- Verify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/challenge/*.go`
- Verify: `/Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/match/finish_match_logic.go`
- Verify: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/user/notification.vue`
- Verify: `/Users/wisesearch/Projects/ChasingPoints/app/subPages/notification/index.vue`

**Step 1: Run focused backend suites**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
GOCACHE=/tmp/go-build go test ./internal/logic/notification ./internal/logic/friend ./internal/logic/challenge ./internal/logic/follow ./internal/logic/tournament ./internal/logic/match
```

Expected: PASS

**Step 2: Run app tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/*.test.mjs
```

Expected: PASS, including new notification settings / center tests.

**Step 3: Run static confidence checks**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "notification_settings" app
rg -n "NotificationModel\\.Create\\(" backend/internal/logic
```

Expected:

- `notification_settings` no longer acts as sole source of truth
- direct `NotificationModel.Create(` calls only remain where intentionally allowed or in the new dispatch service tests

**Step 4: Manual smoke checklist**

Verify manually:

1. Open notification settings page and see five switches
2. Turn off `好友申请`
3. Trigger new friend request from another account
4. Confirm no new row in message center and unread count unchanged
5. Turn it back on
6. Trigger again and confirm row appears

**Step 5: Commit**

```bash
git add -A
git commit -m "完成通知偏好与消息中心闭环验证" -m "集中跑通通知偏好、统一分发、消息中心展示和设置页改造后的验证，确保关闭某类通知后不会再出现伪闭环行为。" -m "Constraint: 验证必须覆盖写库、未读数、WebSocket、Push 四个侧面" -m "Confidence: medium" -m "Scope-risk: moderate" -m "Directive: 后续新增通知类型必须补回归测试，否则易再次出现设置与消息链路脱节" -m "Tested: focused backend go test suites; node --test tests/*.test.mjs; manual smoke checklist" -m "Not-tested: real production push provider delivery"
```
