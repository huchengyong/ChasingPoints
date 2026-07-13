# Venue Submit Basics Only Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Shrink the venue submission flow so both the UniApp form and the `/api/venue/create` contract only accept core venue basics: name, city, district, and address.

**Architecture:** Keep venue creation, duplicate-address detection, and async geocode task creation unchanged, but reduce the input surface to a single “基础资料” section. The frontend will only collect and submit the four basic fields, while the backend request schema and request-to-model mapping will be trimmed to match and continue writing empty defaults for the removed venue detail fields.

**Tech Stack:** UniApp Vue 3 Composition API, JavaScript API wrapper, go-zero API/types generation, Go logic tests.

---

### Task 1: Tighten backend tests around the reduced create payload

**Files:**
- Modify: `backend/internal/logic/create_venue_logic_test.go`
- Reference: `backend/internal/logic/create_venue_logic.go`
- Reference: `backend/internal/types/types.go`

**Step 1: Write the failing test**

Update `backend/internal/logic/create_venue_logic_test.go` so the happy-path test builds `types.CreateVenueReq` with only:

- `Name`
- `City`
- `District`
- `Address`

Add assertions that the resulting `venue` keeps removed fields at defaults:

```go
if venue.Phone != "" {
	t.Fatalf("expected empty phone, got %q", venue.Phone)
}
if venue.BusinessHours != "" || venue.PriceRange != "" {
	t.Fatal("expected removed extended fields to stay empty")
}
if venue.TableCount != 0 {
	t.Fatalf("expected zero table count, got %d", venue.TableCount)
}
if venue.Images != "[]" {
	t.Fatalf("expected empty images json, got %s", venue.Images)
}
```

**Step 2: Run the test to verify it fails**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic -run 'Test(BuildVenueAndTaskFromReqCreatesAsyncGeocodePayload|ValidateCreateVenueReqRejectsMissingBaseFields|BuildCreateVenueRespUsesAsyncMessage)' -count=1
```

Expected: FAIL because the current test still assumes removed request fields exist.

**Step 3: Write the minimal implementation**

Adjust the test fixture so it matches the reduced contract and checks the default-field behavior.

**Step 4: Run the test to verify it passes**

Run the same command again and expect PASS.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/create_venue_logic_test.go
git commit -m "test: cover basics-only venue create payload"
```

### Task 2: Reduce the backend create request contract

**Files:**
- Modify: `backend/chasing_points.api`
- Modify: `backend/internal/types/types.go`
- Modify: `backend/internal/logic/create_venue_logic.go`

**Step 1: Shrink the API schema**

In `backend/chasing_points.api`, update `CreateVenueReq` so it only contains:

- `Name`
- `Address`
- `City`
- `District`

Remove:

- `Phone`
- `Images`
- `BusinessHours`
- `TableCount`
- `PriceRange`
- `Description`

**Step 2: Regenerate or align generated request types**

Update `backend/internal/types/types.go` so `CreateVenueReq` exactly matches the new API schema.

If the project uses generated types manually checked in, make the struct match the `.api` contract even if `goctl` is not run in this step.

**Step 3: Default removed fields inside venue construction**

In `backend/internal/logic/create_venue_logic.go`, update `buildVenueAndTaskFromReq` so it no longer reads removed request fields and instead writes explicit defaults when building `model.Venue`.

Expected mapping:

```go
Phone:         "",
Images:        "[]",
BusinessHours: "",
TableCount:    0,
PriceRange:    "",
Description:   "",
```

**Step 4: Run focused backend verification**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic -run 'Test(BuildVenueAndTaskFromReqCreatesAsyncGeocodePayload|ValidateCreateVenueReqRejectsMissingBaseFields|BuildCreateVenueRespUsesAsyncMessage)' -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/backend/chasing_points.api /Users/wisesearch/Projects/ChasingPoints/backend/internal/types/types.go /Users/wisesearch/Projects/ChasingPoints/backend/internal/logic/create_venue_logic.go
git commit -m "refactor: reduce venue create request to basics"
```

### Task 3: Collapse the UniApp submission page to one basic section

**Files:**
- Modify: `app/subPages/venue/submit.vue`
- Modify: `app/api/venue.js`

**Step 1: Remove non-basic form fields from state and template**

In `app/subPages/venue/submit.vue`:

- delete the “位置与联系” section
- delete the “经营信息” section
- delete the “特色说明” section
- remove `phone`, `business_hours`, `table_count`, `price_range`, `description` from `form`

Keep only:

- `name`
- `city`
- `district`
- `address`

**Step 2: Refresh page copy**

Update the intro, section tip, and submit-bar copy so they all describe a basics-only submission flow.

Use wording consistent with the approved design, for example:

- intro desc: `填写基础资料后即可提交，系统会根据地址自动定位球馆位置。`
- submit complete state: `确认基础资料后上传球馆`
- submit complete tip: `提交后系统会根据地址自动定位并等待审核。`

**Step 3: Reduce the create payload and API docs**

In `app/subPages/venue/submit.vue`, change `handleSubmit()` so it only posts:

```js
const data = {
  name: form.value.name.trim(),
  city: form.value.city.trim(),
  district: form.value.district.trim(),
  address: form.value.address.trim()
}
```

Update the JSDoc in `app/api/venue.js` so `createVenue` documents the same four fields.

**Step 4: Run focused frontend verification**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints
rg -n "phone|business_hours|table_count|price_range|description" app/subPages/venue/submit.vue app/api/venue.js
```

Expected: no remaining venue-submit references to removed fields inside those two files.

**Step 5: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/app/subPages/venue/submit.vue /Users/wisesearch/Projects/ChasingPoints/app/api/venue.js
git commit -m "refactor: keep venue submit form to basic fields"
```

### Task 4: Run regression checks and record rollout notes

**Files:**
- Modify: `docs/plans/2026-03-24-venue-submit-basics-only-design.md`
- Modify: `docs/plans/2026-03-24-venue-submit-basics-only-implementation.md`

**Step 1: Run combined verification**

Run:

```bash
cd /Users/wisesearch/Projects/ChasingPoints/backend
go test ./internal/logic -run 'Test(BuildVenueAndTaskFromReqCreatesAsyncGeocodePayload|ValidateCreateVenueReqRejectsMissingBaseFields|BuildCreateVenueRespUsesAsyncMessage)' -count=1

cd /Users/wisesearch/Projects/ChasingPoints
rg -n "phone|business_hours|table_count|price_range|description" app/subPages/venue/submit.vue app/api/venue.js
git diff --stat
```

Expected:

- backend logic tests PASS
- removed fields no longer appear in the venue submit page or create API wrapper docs
- diff only touches the intended venue submission files plus the plan docs

**Step 2: Record rollout note**

Document any remaining compromise, especially:

- venue detail/list pages still support extended fields and may show empty-state text
- database schema intentionally remains unchanged

**Step 3: Commit**

```bash
git add /Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-03-24-venue-submit-basics-only-design.md /Users/wisesearch/Projects/ChasingPoints/docs/plans/2026-03-24-venue-submit-basics-only-implementation.md
git commit -m "docs: capture basics-only venue submit plan"
```
