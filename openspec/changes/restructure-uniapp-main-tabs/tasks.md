## 1. 平台骨架与先行验证

- [x] 1.1 新增 `uniapp/tests/helpers/pages-json.mjs`（去掉整行 `//` 注释后解析），并把直接 `JSON.parse(pages.json)` 的测试改用它：`member-analysis-pages`、`page-theme-contract`、`theme-platform-config`、`tournament-page`。
- [x] 1.2 修改 `pages.json` 的 tabBar：`list` 改为首页、赛讯、消息（`pages/message/index`）、我的；`// #ifdef MP-WEIXIN` 内加 `"custom": true`，`// #ifdef APP-HARMONY` 内加 `midButton`（`backgroundImage` 居中圆形图，高度不超过 50px）；新增 `pages/message/index` 页面注册（系统导航栏，标题「消息」，开启下拉刷新）；`pages/match/index` 改标题为「观赛大厅」。
- [x] 1.3 生成可替换的图标：消息 Tab 四张（`message[-selected]_{light,dark}.png`，64×64，与现有风格一致）和中间按钮图 `scan-button.png`（品牌金圆形配深色扫码图形，两种主题共用）；`theme.json` 增加消息图标变量，删除观赛图标变量和 `static/tabbar/match*.png`。
- [x] 1.4 新增 `uniapp/custom-tab-bar/index.{js,json,wxml,wxss}` 最小可用版本：四个 Tab 加中间按钮，`view` 渲染，`position: fixed`，包含安全区；数据项为 `selected / dark / hidden / badge`；点击 Tab 调 `wx.switchTab`，点击中间按钮调用注入的 `onScan`；中间按钮带 `aria-label="扫码"`。
- [x] 1.5 先建一个最小的 `pages/message/index` 占位页，在 `App.vue` 的 `onLaunch` 中用 `#ifdef APP-HARMONY` 注册 `uni.onTabBarMidButtonTap`（先接一个临时提示）。
- [ ] 1.6 先行验证并记录结果：在微信开发者工具中验证 Tab 切换、中间按钮点击和暗色样式；在鸿蒙模拟器或真机上验证 `midButton` 的位置、点击回调、`setTabBarBadge` / `showTabBarRedDot` 和暗色样式。鸿蒙不可用时按 design 的风险预案调整后，再继续后续任务。（编译层已验证：mp-weixin 产物 `tabBar.custom = true`、`custom-tab-bar/` 原样拷入；app-harmony 产物含 `midButton` 配置。设备验证待人工执行，见 `docs/testing/restructure-uniapp-main-tabs-acceptance.md`。）

## 2. 主导航同步

- [x] 2.1 先写 `tests/main-tab-bar.test.mjs`，覆盖：`resolveMessageBadge`（未读数、红点、99+、未登录、身份不匹配）；`resolveMainTabBarState`；小程序分支（`getTabBar` 存在 / 缺失）；鸿蒙分支（非 Tab 页跳过、下标 2）；`setMainTabBarVisible` 两端行为。
- [x] 2.2 实现 `utils/main-tab-bar.js` 和 `useMainTabBar(tabKey)`：`onShow` 时同步选中态、主题、角标、显隐并注入扫码回调；页面显示期间监听计数和主题，`onHide` 时停止监听。
- [x] 2.3 为原生组件补测试：为 `Component` / `wx` 打桩加载 `custom-tab-bar/index.js`，断言点击各 Tab 的 `switchTab` 路径正确、点击中间按钮调用注入回调、未注入时不报错。
- [x] 2.4 修改 `utils/theme-application.js`：`TAB_BAR_ROUTES` / `TAB_BAR_ITEMS` 改为首页、赛讯、消息、我的；小程序端跳过原生 TabBar 样式接口，改由同步工具下发 `dark`。更新 `tests/theme-application.test.mjs` 与 `theme-platform-config` 中的图标断言。
- [x] 2.5 在首页、赛讯、消息、我的四个 Tab 页接入 `useMainTabBar`，小程序端统一预留 `50px + env(safe-area-inset-bottom)` 的底部空间；「我的」浮动奖励条改用同一变量。

## 3. 统一扫码流程

- [x] 3.1 先写 `tests/quick-scan.test.mjs`，覆盖：未登录引导登录且不调起相机；重复触发被忽略；取消不提示；无效码提示；对手码和裁判码分流；扫码期间切号后丢弃结果；交接槽只能取一次且绑定身份；「再来一局」对手不一致时提示；路由不含凭证。
- [x] 3.2 实现 `utils/quick-scan.js`（`startQuickScan` 与内存交接槽），复用 `scanAndResolveMatchCode`；鸿蒙中间按钮回调与小程序注入回调都接到 `startQuickScan`。
- [x] 3.3 新增 `subPages/match/scanConfirm.vue` 并注册到 `subPages/match` 分包：
  - 开局模式：对手卡、球种（默认常用球种，可设为常用）、类型（`START_MATCH_MODE_OPTIONS`）、开始对局。
  - 执裁模式：预览、确认、放弃。

  两种模式成功后都用 `redirectTo` 进入计分页；失败或交接缺失时提供「重新扫码」；全程丢弃身份变化后的迟到响应。样式遵循 DESIGN.md，包含暗色。
- [x] 3.4 为确认页写挂载测试：开局成功跳转、未选球种时禁用、匹配码失效、本人已有未结束对局、执裁确认与失效、无交接数据、切号后迟到响应，并纳入 `page-state-contract` / `page-theme-contract` 检查。
- [x] 3.5 赛果页「再来一局」改为调用 `startQuickScan` 并传入对手与预设，删除 `pending_match_rematch` 的写入；更新相关测试。
- [x] 3.6 排行榜页「发起PK」改为调用 `startQuickScan`，删除该页自带的球种弹窗、扫码和开局代码；更新相关测试。

## 4. 首页并入观赛

- [x] 4.1 先为 `publicReadStore.loadLiveMatchPreview` 写测试：参数为 `scope=hall&status=1&page_size=4`、30 秒 TTL、`force` 回源、并发 single-flight、失败不污染缓存。
- [x] 4.2 实现 loader，并把它加入首页 `loadData`：渲染时去掉本人当前对局、最多 3 场；区块错误独立显示并可重试；「观赛大厅」入口始终可见；点击规则复用观赛列表的工具函数。
- [x] 4.3 首页清理：删除头部消息图标与未读文字、本地的扫码和开局代码、`gameTypeModal`；「发起对局 → 扫描对手二维码」改为调用 `startQuickScan`；二维码弹层改用 `setMainTabBarVisible`。
- [x] 4.4 更新首页相关测试（`home-index`、`request-graph`、`quick-match-home`、`game-type-preference`、`member-identity-pages`、`tournament-page` 等）：首次进入请求数、TTL 内切回不请求、下拉刷新强制回源、切号与迟到响应。

## 5. 观赛大厅

- [x] 5.1 `pages/match/index.vue` 删除开局卡、约球卡与入口、扫码、出示二维码、裁判弹层、`pending_match_rematch` 消费及无用的 import 和样式；不再过滤本人对局；保留标签、筛选、分页、下拉刷新和进入规则。
- [x] 5.2 `subPages/match/challengeCompose.vue` 发送成功后改为 `switchTab` 到首页；用全仓搜索确认没有 `switchTab` 再指向 `/pages/match/index`。
- [x] 5.3 更新 `spectator-lobby`、`referee-scan-preview`、`match-invite-contract`、`snooker-v2`、`match-core-flow`、`quick-match-home`、`request-graph`、`game-type-preference` 等测试，把原本断言观赛页里开局 / 扫码 / 裁判逻辑的用例，改到确认页或统一扫码模块上。
- [x] 5.4 删除不再被任何页面引用的 `components/gameTypeModal.vue` 与 `tests/game-type-modal.test.mjs`，并同步 `page-theme-contract` 的组件清单。

## 6. 消息 Tab

- [x] 6.1 先写消息 Tab 测试：未登录不请求；首次进入只请求一次通知首屏，计数取自活动快照；没有变化时 `onShow` 不回源；`notification` scope 失效后 `onShow` 刷新；可见期间多次失效防抖为一次；切号清空并丢弃迟到响应；单条已读、全部已读、删除对计数的回写。
- [x] 6.2 实现 `pages/message/index.vue`：好友申请、约球邀请入口及数量；从 `subPages/notification/index.vue` 迁入的通知列表逻辑；登录引导；接入 `useMainTabBar('message')`；样式和暗色遵循 DESIGN.md。
- [x] 6.3 删除 `subPages/notification/index.vue` 及 `subPages/notification` 分包注册；更新 `honor-wall`、`page-state-contract`、`notification-center-types`、`auth-scoped-retained-pages`、`notification-list-page` 测试和 `scripts/check-notification-payloads.mjs` 中的路径。

## 7. 「我的」与赛讯

- [x] 7.1 「我的」头部删除扫码按钮和通知铃铛，保留「出示我的二维码」和设置；服务列表第一行新增「约球记录」（`/subPages/social/challenges?tab=history`）；二维码弹层改用 `setMainTabBarVisible`；删除本地的扫码和开局代码。
- [x] 7.2 更新 `user-page`、`user-page-match-entry`、`rank-info-cache`、`session-storage` 等测试中的头部入口、服务列表和二维码显隐断言。
- [x] 7.3 赛讯页接入 `useMainTabBar('tournament')` 和底部预留，不改动赛讯的读取逻辑。

## 8. 文档

- [x] 8.1 更新 `DESIGN.md`：9.5 增加中间扫码按钮和自定义 TabBar 的视觉约束；9.6 的弹层显隐改为统一显隐方法。
- [x] 8.2 更新 `uniapp/AGENTS.md` 关键入口（`custom-tab-bar/`、`utils/main-tab-bar.js`、`utils/quick-scan.js`），同步到 `GEMINI.md` 并用 `cmp` 确认一致；把 `uniapp/subPages/AGENTS.md` 中通知中心的位置改为 `pages/message/index.vue`。

## 9. 验证与收尾

- [x] 9.1 在 `uniapp/` 运行 `node --test tests/*.test.mjs` 和 `cmp AGENTS.md GEMINI.md`；用 `git diff --check` 检查改动。
- [ ] 9.2 用 HBuilderX 生产构建微信小程序：确认产物包含 `custom-tab-bar/`、`app.json` 中 `tabBar.custom = true` 且没有 `midButton`，主包仍低于 2MB。在开发者工具和真机上验收导航、角标、主题、弹层显隐、扫码开局与执裁、再来一局、消息 Tab 刷新。（构建与产物核对已完成：主包约 976 KB；开发者工具与真机验收待人工执行。）
- [ ] 9.3 构建鸿蒙包，在模拟器或真机上验收同样的场景，重点检查 `midButton` 的点击与样式、非 Tab 页返回后角标是否补齐、暗色主题。（鸿蒙编译已通过；模拟器 / 真机验收待人工执行。）
- [x] 9.4 在 `docs/testing/` 记录本次实际执行的命令、平台、环境来源和结果，未执行的项明确标注；给 iOS 迁移工作留一条备注：原生 iOS 仍是四个占位 Tab。
