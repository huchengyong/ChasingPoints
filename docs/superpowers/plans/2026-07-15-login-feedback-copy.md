# Login Feedback Copy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the two-line login status-card copy with the two approved single-line messages for normal and slow authentication states.

**Architecture:** Keep the existing authentication timer, state card, animations, and request locking. Simplify the shared feedback view model from `title` plus `detail` to one `message`, then render that same message in the welcome and login pages.

**Tech Stack:** UniApp Vue 3, JavaScript, SCSS, Node.js `node:test`

---

### Task 1: Use the approved single-line login feedback copy

**Files:**
- Modify: `app/tests/entry-funnel.test.mjs`
- Modify: `app/tests/bind-phone-component.test.mjs`
- Modify: `app/utils/entry-funnel.js`
- Modify: `app/pages/welcome/index.vue`
- Modify: `app/pages/welcome/index.scss`
- Modify: `app/pages/login/login.vue`

- [ ] **Step 1: Update tests to require a single message**

Change the feedback expectations to:

```js
assert.deepEqual(resolveAuthenticationFeedback(), {
  visible: false,
  phase: 'idle',
  message: ''
})

assert.deepEqual(resolveAuthenticationFeedback({
  isAuthenticating: true,
  isSlow: false
}), {
  visible: true,
  phase: 'pending',
  message: '追分正在为您完成登录'
})

assert.deepEqual(resolveAuthenticationFeedback({
  isAuthenticating: true,
  isSlow: true
}), {
  visible: true,
  phase: 'slow',
  message: '网络有些慢，追分竭尽全力为您继续尝试中'
})
```

Update the page-source test to require `authFeedback.message` and reject both `authFeedback.title` and `authFeedback.detail`.

- [ ] **Step 2: Run targeted tests and verify RED**

Run:

```bash
cd app
node --test tests/entry-funnel.test.mjs tests/bind-phone-component.test.mjs
```

Expected: FAIL because the feedback model and templates still expose `title` and `detail`.

- [ ] **Step 3: Implement the single-message feedback model**

Change `resolveAuthenticationFeedback` to return only `visible`, `phase`, and `message`:

```js
if (!isAuthenticating) {
  return { visible: false, phase: 'idle', message: '' }
}

if (isSlow) {
  return {
    visible: true,
    phase: 'slow',
    message: '网络有些慢，追分竭尽全力为您继续尝试中'
  }
}

return {
  visible: true,
  phase: 'pending',
  message: '追分正在为您完成登录'
}
```

In the welcome-page status card and all three login-page status-card instances (WeChat, phone, and Huawei), replace the title/detail block with:

```vue
<text class="auth-feedback-message">{{ authFeedback.message }}</text>
```

Remove the unused copy wrapper and detail styles. Rename `.auth-feedback-title` and its dark-mode override to `.auth-feedback-message` so the existing font size, weight, color, line height, and dark-mode readability remain unchanged. Keep the progress line, pulse dot, slow-state styling, timer threshold, and request locking unchanged.

- [ ] **Step 4: Run targeted and full tests**

Run:

```bash
cd app
node --test tests/entry-funnel.test.mjs tests/bind-phone-component.test.mjs
node --test tests/*.test.mjs
cd ..
git diff --check
```

Expected: targeted tests pass, all mobile tests pass, and `git diff --check` prints no errors.

- [ ] **Step 5: Commit the implementation**

Stage only the login-feedback implementation files and tests, preserving unrelated website changes:

```bash
git add app/tests/entry-funnel.test.mjs \
  app/tests/bind-phone-component.test.mjs \
  app/utils/entry-funnel.js \
  app/pages/welcome/index.vue \
  app/pages/welcome/index.scss \
  app/pages/login/login.vue
git commit -m "fix(app): improve login feedback copy"
```
