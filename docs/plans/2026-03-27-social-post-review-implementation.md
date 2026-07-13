# Social Post Review Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a moderation-first publishing flow for social posts so new posts default to pending review, stay hidden from public timelines until approved, and remain visible to their author in a private “my posts” view with clear review status and reject reason.

**Architecture:** Extend the existing `social_posts` table and `SocialPostModel` with moderation fields and filtered query helpers. Keep public/following feeds strictly published-only, add a new authenticated “mine” endpoint for author-only visibility, and add admin list/review endpoints plus a Vite admin review screen. On the app side, add a dedicated “我的动态” page and route post-publish success into that private moderation-aware surface instead of the public feed.

**Tech Stack:** go-zero API generation, Go + Gorm models/logic/tests, MySQL goose migrations, UniApp Vue3 + JS API facade, Vue 3 + Vite + TypeScript + Element Plus admin, Node `--test`, `go test`, `npm run build`.

---

### Task 1: Add moderation schema and model constants for social posts

**Files:**
- Create: `backend/migrations/20260327153000_add_social_post_review_fields.sql`
- Modify: `backend/internal/model/social_post.go`
- Test: `backend/internal/model/no_auto_migrate_test.go`

**Step 1: Write the failing migration-aware test**

Add or extend a backend test that asserts the social post model expects moderation fields in code but does not auto-migrate them. At minimum, cover the new status constants and default semantics in model-level helper tests.

Example assertions:

```go
if SocialPostStatusPublished != 1 || SocialPostStatusPending != 2 || SocialPostStatusRejected != 3 {
	t.Fatalf("unexpected social post status constants")
}
```

**Step 2: Run the focused test to verify current code does not satisfy the new model contract**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/model -run TestSocialPostStatusConstants -count=1
```

Expected: FAIL because the constants and helpers do not exist yet.

**Step 3: Add the migration and model fields**

Create `backend/migrations/20260327153000_add_social_post_review_fields.sql` to add:

- `status TINYINT NOT NULL DEFAULT 2`
- `reject_reason VARCHAR(255) NOT NULL DEFAULT ''`
- `reviewed_at DATETIME NULL`
- `reviewed_by BIGINT UNSIGNED NULL`

Update `backend/internal/model/social_post.go` to add:

- status constants
- new fields on `SocialPost`
- helper methods for published-only filtering, owner filtering, and review updates

Do not use unsupported MySQL `IF NOT EXISTS` column syntax. Use the repository’s `information_schema.COLUMNS` guard pattern inside the migration if needed.

**Step 4: Run focused verification**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/model -run TestSocialPostStatusConstants -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/migrations/20260327153000_add_social_post_review_fields.sql /Users/wisesearch/Projects/ChasingPoints/backend/internal/model/social_post.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/model/no_auto_migrate_test.go
git commit -m "feat: add social post review schema"
```

### Task 2: Make user-facing social APIs moderation-aware

**Files:**
- Modify: `backend/chasing_points.api`
- Modify: `backend/internal/model/social_post.go`
- Modify: `backend/internal/logic/social/create_post_logic.go`
- Modify: `backend/internal/logic/social/get_public_posts_logic.go`
- Modify: `backend/internal/logic/social/get_post_list_logic.go`
- Modify: `backend/internal/logic/social/like_post_logic.go`
- Modify: `backend/internal/logic/social/unlike_post_logic.go`
- Modify: `backend/internal/logic/social/comment_post_logic.go`
- Modify: `backend/internal/logic/social/get_post_comments_logic.go`
- Create: `backend/internal/logic/social/get_my_post_list_logic.go`
- Test: `backend/internal/logic/social/social_post_review_logic_test.go`

**Step 1: Write failing logic tests**

Create `backend/internal/logic/social/social_post_review_logic_test.go` covering:

- `CreatePost` defaults to pending review
- public feed excludes pending/rejected posts
- following feed excludes pending/rejected posts
- my-posts feed returns the author’s pending/rejected/published posts
- like/comment/comment-list reject non-published posts

Example assertions:

```go
if post.Status != model.SocialPostStatusPending {
	t.Fatalf("expected pending status, got %d", post.Status)
}
```

**Step 2: Run the focused social tests to verify they fail**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic/social -run TestSocialPostReview -count=1
```

Expected: FAIL because the new status-aware behavior and `mine` endpoint do not exist.

**Step 3: Update the API contract and generate code**

Modify `backend/chasing_points.api` to:

- add moderation fields to `SocialPostInfo`
- add `@handler GetMyPostList` with `GET /mine`
- keep `public` and `feed` semantics published-only

Then immediately run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
goctl api go --api chasing_points.api --dir . --style go_zero --home ~/.goctl/default
```

Fill any generated social `todo` logic stubs with real implementations before moving on.

**Step 4: Implement moderation-aware logic**

Update the social logic and model methods so that:

- new posts save with `status = pending`
- `public` and `feed` queries filter by `published`
- `mine` queries filter by `user_id = current user`
- `like`, `unlike`, `comment`, and `get comments` require `published`
- `SocialPostInfo` includes `status`, `status_text`, `reject_reason`, and `is_mine` where appropriate

**Step 5: Run the focused tests to verify they pass**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic/social -run TestSocialPostReview -count=1
```

Expected: PASS.

**Step 6: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/chasing_points.api /Users/wisesearch/Projects/ChasingPoints/backend/internal/model/social_post.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/social/create_post_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/social/get_public_posts_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/social/get_post_list_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/social/like_post_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/social/unlike_post_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/social/comment_post_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/social/get_post_comments_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/social/get_my_post_list_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/social/social_post_review_logic_test.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/types/types.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/handler/routes.go
git commit -m "feat: gate social feeds by review status"
```

### Task 3: Add admin social review APIs and tests

**Files:**
- Modify: `backend/chasing_points.api`
- Modify: `backend/internal/model/social_post.go`
- Create: `backend/internal/logic/admin/admin_get_social_post_list_logic.go`
- Create: `backend/internal/logic/admin/admin_review_social_post_logic.go`
- Create: `backend/internal/logic/admin/admin_social_post_review_logic_test.go`
- Reference: `backend/internal/logic/admin/admin_get_venue_list_logic.go`
- Reference: `backend/internal/logic/admin/admin_review_venue_logic.go`

**Step 1: Write failing admin tests**

Create `backend/internal/logic/admin/admin_social_post_review_logic_test.go` covering:

- admin can list posts by status
- review approve changes pending to published
- review reject changes pending to rejected and stores reject reason
- reject without reason returns validation error

**Step 2: Run the focused admin tests to verify they fail**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic/admin -run TestAdminSocialPostReview -count=1
```

Expected: FAIL because the admin social review endpoints do not exist.

**Step 3: Extend the admin API contract and generate code**

Modify `backend/chasing_points.api` to add:

- `AdminSocialPostListReq`
- `AdminSocialPostInfo`
- `AdminSocialPostListResp`
- `AdminSocialPostReviewReq`
- admin routes under `prefix: /api/admin/social`

Then run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
goctl api go --api chasing_points.api --dir . --style go_zero --home ~/.goctl/default
```

Fill any generated admin `todo` logic stubs immediately.

**Step 4: Implement admin review logic**

Follow the same style as the venue admin logic:

- list posts with status filter and author info
- approve sets `status = published`, clears reject reason, stamps `reviewed_at/reviewed_by`
- reject sets `status = rejected`, writes reject reason, stamps `reviewed_at/reviewed_by`

Keep validation strict: only pending posts should be reviewable in this first version.

**Step 5: Run the focused tests to verify they pass**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic/admin -run TestAdminSocialPostReview -count=1
```

Expected: PASS.

**Step 6: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/chasing_points.api /Users/wisesearch/Projects/ChasingPoints/backend/internal/model/social_post.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/admin/admin_get_social_post_list_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/admin/admin_review_social_post_logic.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/admin/admin_social_post_review_logic_test.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/types/types.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/handler/routes.go
git commit -m "feat: add admin social post review apis"
```

### Task 4: Add app-side API facade and a private “我的动态” page

**Files:**
- Modify: `app/api/social.js`
- Modify: `app/pages.json`
- Modify: `app/pages/user/index.vue`
- Modify: `app/subPages/social/postCreate.vue`
- Create: `app/subPages/social/myPosts.vue`
- Test: `app/tests/social-review.test.mjs`

**Step 1: Write failing pure-app tests**

Create `app/tests/social-review.test.mjs` for small framework-free helpers or adapter logic such as:

- status-to-copy mapping
- reject reason visibility
- “my posts” empty state copy

If a new helper file is introduced, test that helper rather than the Vue file directly.

Example assertions:

```js
assert.equal(resolveMyPostStatusCopy(2).title, '审核中，仅自己可见')
assert.equal(resolveMyPostStatusCopy(3, '包含辱骂内容').reasonVisible, true)
```

**Step 2: Run the app test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/social-review.test.mjs
```

Expected: FAIL because the helper/page contract does not exist yet.

**Step 3: Add the API facade and private route**

Update `app/api/social.js` to add `getMyPosts`.

Update `app/pages.json` to register `subPages/social/myPosts`.

Update `app/pages/user/index.vue` to add a visible “我的动态” entry.

Create `app/subPages/social/myPosts.vue` to:

- call `getMyPosts`
- render status tags
- show reject reason when present
- allow delete on own posts
- keep interactions disabled for non-published posts

Update `app/subPages/social/postCreate.vue` success copy to:

- say the post was submitted for review
- optionally route the user toward the private page instead of a silent back

**Step 4: Run the focused app test to verify it passes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/social-review.test.mjs
```

Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/api/social.js /Users/wisesearch/Projects/ChasingPoints/app/pages.json /Users/wisesearch/Projects/ChasingPoints/app/pages/user/index.vue /Users/wisesearch/Projects/ChasingPoints/app/subPages/social/postCreate.vue /Users/wisesearch/Projects/ChasingPoints/app/subPages/social/myPosts.vue /Users/wisesearch/Projects/ChasingPoints/app/tests/social-review.test.mjs
git commit -m "feat: add private reviewed social post view"
```

### Task 5: Build the admin moderation screen

**Files:**
- Modify: `admin/src/router/index.ts`
- Modify: `admin/src/layout/index.vue`
- Create: `admin/src/api/social.ts`
- Create: `admin/src/views/social-posts/index.vue`
- Reference: `admin/src/api/venue.ts`
- Reference: `admin/src/views/venues/index.vue`

**Step 1: Write the screen contract in TypeScript first**

Create `admin/src/api/social.ts` with explicit types for:

- social post review list item
- list params
- review payload

Match the backend API contract exactly.

**Step 2: Add the route and menu entry**

Update:

- `admin/src/router/index.ts`
- `admin/src/layout/index.vue`

Add a new page like:

- route path: `/social-posts`
- title: `动态审核`

**Step 3: Create the review page**

Create `admin/src/views/social-posts/index.vue` with:

- status filter
- table columns for author, type, content summary, images count, created time, status
- detail dialog with content and image preview
- approve/reject actions
- reject dialog or prompt that requires a reason

Use the venues page as the interaction template rather than inventing a new admin pattern.

**Step 4: Run the admin build**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/admin
npm run build
```

Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/admin/src/router/index.ts /Users/wisesearch/Projects/ChasingPoints/admin/src/layout/index.vue /Users/wisesearch/Projects/ChasingPoints/admin/src/api/social.ts /Users/wisesearch/Projects/ChasingPoints/admin/src/views/social-posts/index.vue
git commit -m "feat: add admin social post review screen"
```

### Task 6: Run end-to-end verification and document any rollout notes

**Files:**
- Modify: `docs/plans/2026-03-27-social-post-review-design.md`
- Modify: `docs/plans/2026-03-27-social-post-review-implementation.md`

**Step 1: Run backend tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./...
```

Expected: PASS.

**Step 2: Run app tests**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/app
node --test tests/*.test.mjs
```

Expected: PASS.

**Step 3: Run admin build**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/admin
npm run build
```

Expected: PASS.

**Step 4: Sanity-check generated API and routes**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "GetMyPostList|AdminGetSocialPostList|AdminReviewSocialPost|/api/admin/social|/api/social/mine" backend app admin
```

Expected: all new public/admin review entry points are wired through the repo.

**Step 5: Capture rollout notes**

If any edge case remains deferred, update the two plan docs with a short note. Keep this limited to real rollout caveats such as “edit-and-resubmit intentionally deferred”.

**Step 6: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-03-27-social-post-review-design.md /Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-03-27-social-post-review-implementation.md
git commit -m "docs: capture social post review rollout notes"
```
