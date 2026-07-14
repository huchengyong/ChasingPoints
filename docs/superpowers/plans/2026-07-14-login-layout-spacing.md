# 登录页布局间距调整 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 下移微信登录页的品牌和主文案区域，同时保持底部登录按钮靠底。

**Architecture:** 仅调整登录页现有 SCSS 间距值，不修改模板、业务逻辑或组件结构。继续依赖 `.wechat-actions { margin-top: auto; }` 保持底部操作区的位置。

**Tech Stack:** UniApp、Vue 3、SCSS、Node.js `node:test`

---

## 文件结构

- 修改 `app/pages/login/login.vue`：调整页面顶部内边距和主文案顶部外边距。
- 不新增测试文件：这是两个静态布局值的调整，仓库当前没有页面视觉回归测试框架；使用全量逻辑测试和精确 diff 检查验证无逻辑影响、无范围扩张。

### Task 1: 调整登录页纵向间距

**Files:**
- Modify: `app/pages/login/login.vue:615-688`
- Test: `app/tests/*.test.mjs`

- [ ] **Step 1: 确认当前样式基线**

Run:

```bash
rg -n "padding: calc\(44rpx|margin-top: 116rpx|margin-top: auto" app/pages/login/login.vue
```

Expected: 找到 `.login-container` 的 `44rpx`、`.hero-section` 的 `116rpx`，以及 `.wechat-actions` 的 `margin-top: auto`。

- [ ] **Step 2: 实施最小样式修改**

在 `app/pages/login/login.vue` 中应用：

```diff
- padding: calc(44rpx + env(safe-area-inset-top)) 40rpx calc(32rpx + env(safe-area-inset-bottom));
+ padding: calc(76rpx + env(safe-area-inset-top)) 40rpx calc(32rpx + env(safe-area-inset-bottom));
```

```diff
- margin-top: 116rpx;
+ margin-top: 156rpx;
```

不要修改 `.wechat-actions` 的 `margin-top: auto`。

- [ ] **Step 3: 检查变更范围**

Run:

```bash
git diff --check
git diff -- app/pages/login/login.vue
```

Expected: `git diff --check` 无输出；页面 diff 只包含上述两个数值变化。

- [ ] **Step 4: 运行移动端逻辑测试**

Run:

```bash
cd app
node --test tests/*.test.mjs
```

Expected: 所有测试通过，无失败用例。

- [ ] **Step 5: 提交实现**

```bash
git add app/pages/login/login.vue docs/superpowers/plans/2026-07-14-login-layout-spacing.md
git commit -m "fix(app): rebalance login page spacing"
```
