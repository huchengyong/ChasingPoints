# 登录页品牌区位置修订 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将登录页品牌行固定下移 `112rpx`，使 Logo 大致位于微信胶囊菜单下方，同时保持底部按钮靠底。

**Architecture:** 仅在现有 `.brand-row` 样式中新增顶部外边距，不引入胶囊位置计算或运行时逻辑。现有 Flex 布局继续负责压缩中间剩余空间并保持操作区位置。

**Tech Stack:** UniApp、Vue 3、SCSS、Node.js `node:test`

---

## 文件结构

- 修改 `app/pages/login/login.vue`：为品牌行增加固定顶部外边距。
- 不新增测试文件：仓库没有页面视觉回归测试框架，使用文本断言、精确 diff 和全量逻辑测试验证。

### Task 1: 下移登录页品牌行

**Files:**
- Modify: `app/pages/login/login.vue:637-642`
- Test: `app/tests/*.test.mjs`

- [ ] **Step 1: 运行目标样式断言并确认失败**

Run:

```bash
rg -n "margin-top: 112rpx" app/pages/login/login.vue
```

Expected: 命令退出码为 1，没有匹配项。

- [ ] **Step 2: 实施最小样式修改**

在 `.brand-row` 中新增：

```scss
margin-top: 112rpx;
```

不要修改页面 padding、`.login-content` 最小高度、`.hero-section` 间距或 `.wechat-actions` 的 `margin-top: auto`。

- [ ] **Step 3: 验证目标样式断言通过**

Run:

```bash
rg -n "margin-top: 112rpx" app/pages/login/login.vue
```

Expected: 在 `.brand-row` 中找到唯一匹配项，命令退出码为 0。

- [ ] **Step 4: 检查变更范围**

Run:

```bash
git diff --check
git diff -- app/pages/login/login.vue
```

Expected: `git diff --check` 无输出；页面 diff 只新增一行 `margin-top: 112rpx;`。

- [ ] **Step 5: 运行移动端全量逻辑测试**

Run:

```bash
cd app
node --test --test-reporter=dot tests/*.test.mjs
```

Expected: 命令退出码为 0，所有测试通过。

- [ ] **Step 6: 提交实现**

```bash
git add app/pages/login/login.vue docs/superpowers/plans/2026-07-14-login-brand-position.md
git commit -m "fix(app): lower login brand section"
```
