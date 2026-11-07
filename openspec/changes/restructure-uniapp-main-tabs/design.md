## Context

- 当前 `uniapp/pages.json` 的 tabBar 是原生的「首页 `pages/index` | 观赛 `pages/match` | 赛讯 `pages/tournament` | 我的 `pages/user`」。主题由 `theme.json` 加 `utils/theme-application.js` 驱动：运行时按下标调用 `setTabBarStyle` / `setTabBarItem` 切换亮色、暗色图标；用户可以选择跟随系统、浅色或深色（`store/theme.js`）。
- 「扫码开局」在 `pages/index`、`pages/match`、`pages/user` 中各有一份几乎相同的实现：先用 `gameTypeModal` 选球种，再用 ActionSheet 选对局类型，然后扫码，最后 `startMatch`。「扫码执裁」及裁判确认弹层只在观赛页。赛果页「再来一局」把上下文写入本地存储 `pending_match_rematch`，再 `switchTab` 到观赛页消费。扫码解析统一走 `utils/match-scan.js`，它已能区分 `start_match`、`preview_match_invite`、`join_referee` 等结果。
- 未读和待处理计数：Bootstrap 及 `activityStore` 同步到 `notificationStore.unreadCount` 和 `friendRequestStore.pendingCount`，WebSocket 的 `NOTIFICATION_UPDATE` / `USER_DATA_UPDATED` 实时更新这些计数，并让 `userDataInvalidationStore` 的 `notification` scope 失效。「我的」铃铛显示 `未读 + 好友申请`。后端只有在对应通知偏好开启时才创建 `friend_request` / `challenge` 通知（`NotificationDispatchService.isNotificationEnabled`），所以好友申请和约球邀请在默认设置下会同时产生一条未读通知和一个待处理计数。
- 通知中心目前是分包页 `subPages/notification/index`，每次打开都在 `onLoad` 时全量加载。后端推送和通知载荷的跳转目标都不是它，也不是任何 Tab 页。
- 平台：UniApp 只编译微信小程序与鸿蒙（HBuilderX 4.76，`app-harmony`），iOS / Android 改为原生，不在本次范围。最近一次小程序生产构建（2026-10-23）主包约 978KB，上限 2MB。
- 已核实的平台能力：
  - **微信小程序**：`custom: true` 需要代码根目录下的原生组件 `custom-tab-bar/index.{js,json,wxml,wxss}`。uni-app 的 mp-weixin 编译器会把项目根的 `custom-tab-bar` 原样拷进产物，不经过 Vue 编译。官方文档说明：自定义后「与 tabBar 样式相关的接口，如 wx.setTabBarItem 等将失效」；每个 Tab 页有各自的 TabBar 实例，选中态要在当前页通过 `getTabBar()` 调用 `setData`。uni-app 的小程序 JSON 处理会删掉 tabBar 里的 `midButton`。
  - **鸿蒙**：HBuilderX 4.76 自带运行时（`@dcloudio/uni-app-runtime` 4.73，更新日志「2.3.4 新增 支持 tabBar midButton」）中，`uni-tabview` 原生模块提供 `onMidButtonClick`，原生 TabBar 把中间按钮插在第 `list.length / 2` 项之后，栏高 50vp，并标注「MidButton不支持动态修改」；`setTabBarStyle` 的协议里没有 `midButton`。`setTabBarBadge`、`setTabBarStyle`、`hide/showTabBar` 在非 Tab 页调用都会被拒绝（`not TabBar page`）。鸿蒙运行时不读取 tabBar 的 `custom`。

## Goals / Non-Goals

**Goals:**
- 两端的导航都是「首页 | 赛讯 | (扫码) | 消息 | 我的」，样式、主题与角标一致。
- 所有「扫对手码 / 扫裁判码」入口收敛为一个先扫后选的流程，删掉三份重复的开局代码。
- 观赛并入首页，观赛大厅只负责观赛列表。
- 消息 Tab 做成常驻页后，读取次数仍然有界，账号之间不串数据。
- 不改后端接口、数据库，不改原生 iOS。

**Non-Goals:**
- 原生 iOS / Android 的导航调整（iOS 目前是四个占位 Tab，交给 iOS 迁移工作）。
- 私信、会话等新的消息类型；通知列表本身的视觉改版。
- 首页「正在进行」改成好友维度或实时推送。
- 外凸式中间按钮。

## Decisions

### D1：小程序用原生 `custom-tab-bar`，鸿蒙用原生 `midButton`
- `pages.json` 的 `tabBar.list` 改为四项：首页、赛讯、消息、我的。`custom: true` 放在 `// #ifdef MP-WEIXIN` 里，`midButton` 放在 `// #ifdef APP-HARMONY` 里，两段都写在 `color` 等键之前、每段末尾带逗号，保证去掉任一段后仍是合法 JSON。
- 小程序在 `uniapp/custom-tab-bar/` 新增原生组件，用 `view`（不用 `cover-view`）渲染四个 Tab 和中间按钮，以 `position: fixed` 固定在底部并包含 `env(safe-area-inset-bottom)`。用普通 view，页面弹层才能盖住它。
- 鸿蒙继续用原生 TabBar，并配置 `midButton`；点击由 `App.vue` 的 `onLaunch` 在 `#ifdef APP-HARMONY` 内用 `uni.onTabBarMidButtonTap` 注册一次。
- 现在有多个测试直接 `JSON.parse(pages.json)`。新增 `tests/helpers/pages-json.mjs`，读取时去掉整行 `//` 注释再解析（两个平台的键都会保留），用到它的测试统一改用这个 helper。
- 备选：两端都在页面内用 Vue 组件画 TabBar。否决：鸿蒙需要在启动时隐藏原生栏，会闪一下，还会失去原生安全区处理；小程序为了 `custom: true` 仍然得放一个空的原生组件；而且四个页面都要挂载并维护同一个组件。
- 备选：不写条件编译，两个键都直接写。否决：虽然目前编译器和运行时会各自忽略不认识的键，但这依赖实现细节；条件编译是 uni-app 文档里的写法，读的人一眼就能看出平台意图。

### D2：自定义 TabBar 只负责渲染和切换，状态由页面同步
- 原生组件只维护 `selected`、`dark`、`hidden`、`badge`（`none` / `dot` / `text`）这几项数据。点击 Tab 调用 `wx.switchTab`；点击中间按钮时调用页面同步时注入的 `onScan` 回调，回调缺失时不做任何事。它不 import 项目里的模块，也不发请求。
- `utils/main-tab-bar.js` 放纯逻辑和平台分支：`MAIN_TABS`、`resolveMessageBadge`、`resolveMainTabBarState`、`syncMainTabBar` 与 `setMainTabBarVisible`。它不引用 store，避免与 `theme-application.js` 形成循环依赖，`uniApi` 由调用方显式传入（小程序里 `uni` 是编译期注入的，不能依赖 `globalThis.uni`）。平台用运行时 `uniPlatform` 判断（`mp-weixin` 走自定义组件，其余走原生 TabBar）。
- `utils/use-main-tab-bar.js` 提供组合式函数 `useMainTabBar(tabKey)`，供四个 Tab 页使用，并导出已注入 `uni` 的 `setMainTabBarVisible`。它在 `onShow` 时同步一次，页面显示期间监听计数和主题，变化后重新同步，`onHide` 时停止同步。计数只在对应 store 的身份与当前 `userId + authGeneration` 一致时才采用。
- 小程序分支通过当前页的 `getTabBar()` 调 `setData`，调用前先判断 `getTabBar` 是否存在。鸿蒙分支只在 Tab 页处于显示状态时由 `useMainTabBar` 调用，避免 `not TabBar page` 的拒绝，并缓存已下发的角标与显隐状态，避免重复调用原生接口（重复的 `showTabBar` 会再播一次动画）。
- 备选：在原生组件里通过 `getApp().globalData` 读状态、调扫码。否决：会增加一个全局可变的入口，而且组件里的数据只有在被拉取时才会更新，切号后容易显示旧计数。

### D3：主题与图标
- `TAB_BAR_ROUTES` / `TAB_BAR_ITEMS` 改为首页、赛讯、消息、我的。`applyRuntimeTheme` 在小程序端跳过原生 TabBar 的样式接口（自定义模式下它们无效），由 `syncMainTabBar` 下发 `dark`；鸿蒙端照旧调用。
- 新增消息 Tab 的四张图标（亮色 / 暗色，选中 / 未选中，64×64，风格与现有图标一致），以及一张中间按钮图：品牌金圆形配深色扫码图形，两种主题共用，不随主题切换，因为鸿蒙 `midButton` 不能动态修改。观赛的 Tab 图标和 `theme.json` 里的对应变量随本次一起删除。
- 中间按钮与栏同高、不外凸：鸿蒙的 `midButton` 用 `backgroundImage` 承载这张居中圆形图，宽高不超过 50px；小程序复用同一张图，并加 `aria-label="扫码"`。外凸的按钮在鸿蒙上会超出 50vp 的行高，超出部分的点击命中不可靠，所以不做。

### D4：统一扫码流程加扫码确认页
- 新增 `utils/quick-scan.js`（纯逻辑，只用相对路径引用，会话、预览接口和 `uni` 都由 `createQuickScan` 注入），再由 `utils/quick-scan-entry.js` 绑定真实的用户会话与接口，导出 `startQuickScan({ rematch, replace })` 和 `takeScanHandoff(identity)`。流程依次为：
  1. 未登录时引导登录；
  2. 用进行中标记防止重复触发；
  3. 记录发起时的 `userId + authGeneration`；
  4. 调用 `scanAndResolveMatchCode`；
  5. 身份已变化就丢弃结果；
  6. 按结果类型分流；
  7. 把结果放进按身份绑定、只能取一次的内存交接槽，然后 `navigateTo('/subPages/match/scanConfirm')`。

  路由里不带任何凭证。
- `subPages/match/scanConfirm.vue` 有两种模式：
  - 开局模式：展示对手，球种默认常用球种，类型复用 `START_MATCH_MODE_OPTIONS`，可勾选「设为常用」；点击「开始对局」后调用 `buildStartMatchPayload` 和 `startMatch`，再用 `resolveStartMatchGuardAction` 处理结果。
  - 执裁模式：调用 `previewMatchReferee`，确认后调用 `joinMatchReferee`。

  两种模式成功后都用 `redirectTo` 进入计分页，返回时回到发起页。拿不到交接数据时，显示「重新扫码」。
- 中间按钮（两端）、首页「扫描对手二维码」、排行榜页「发起PK」和赛果页「再来一局」都调用 `startQuickScan`。「再来一局」传入期望的对手和预设的球种、类型与公开范围；对手不一致时就地提示。`pending_match_rematch` 的读写随之删除。首页、观赛、我的、排行榜四处各自的扫码和开局代码、`gameTypeModal` 引用以及观赛页的裁判弹层一并删除；排行榜页原先固定按排位开局，改走统一流程后与其他入口一样在确认页选择对局类型（默认练习赛）。
- 先扫后选会把「选球种」挪到扫码之后。匹配凭证默认 300 秒有效（`MATCH_INVITE_TTL_SECONDS`），足够完成选择；过期由确认页提示后重新扫码。
- 备选：在每个 Tab 页放一个全局弹层宿主。否决：鸿蒙中间按钮的回调在应用级，没有确定的宿主页；弹层还得配合隐藏原生 TabBar；赛果页等非 Tab 页也要再放一份。确认页在所有入口下的行为一致，也便于测试。

### D5：首页「正在进行」走公共短缓存
- `publicReadStore` 新增 `loadLiveMatchPreview`，请求 `GET /api/public/matches?scope=hall&status=1&page=1&page_size=4`，TTL 30 秒（与排行榜摘要一致），并加入首页 `loadData` 的 `Promise.allSettled`。区块的错误单独记录。客户端在渲染时去掉本人当前对局，最多取 3 场；`isMine` 等访问者相关字段用 `normalizeSpectatorMatch` 按当前身份计算，不写进公共缓存。
- 首页首次显示多 1 个请求，TTL 内切回不再请求，下拉刷新强制回源。卡片的点击规则复用 `shouldOpenPlayingForSpectatorMatch` 和 `buildFinishedMatchDetailRoute`。
- 备选：好友维度的预览。否决：需要用户级缓存与身份隔离，未登录时也不可用；观赛大厅的好友标签已经覆盖这个需求。备选：把预览并进 Bootstrap。否决：只为一个首页区块改后端。

### D6：观赛大厅保留 `pages/match/index` 路径，原入口按新位置迁移
- 该页从 tabBar 移除，但保留在主包 `pages` 中，标题改为「观赛大厅」。删除开局卡、约球卡与入口、扫码、出示二维码、裁判弹层和 `pending_match_rematch` 的消费；不再从列表里过滤本人对局。默认标签、筛选、分页和下拉刷新不变。
- `challengeCompose` 发送成功后改为 `switchTab` 到首页（活动 Store 已标脏）；赛果页改为调用 `startQuickScan`。全仓不再有 `switchTab` 指向该路径。
- 「我的」头部只保留「出示我的二维码」和设置。「约球记录」放在服务列表第一行，打开 `subPages/social/challenges?tab=history`。快捷区保持现有四列栅格不变。首页头部去掉消息图标（非小程序）和未读提示（小程序）。
- 备选：迁到分包并保留一个兼容跳转页。否决：多出一个兼容页和大量测试路径改动，主包没有容量压力。

### D7：消息 Tab 承接通知中心，删除旧分包页
- 新增主包 Tab 页 `pages/message/index`：顶部是「好友申请」（`friendRequestStore.pendingCount`）和「约球邀请」（`activityStore.receivedChallengeCount`）两个入口，下面是从 `subPages/notification/index` 迁入的列表逻辑（已读、全部已读、删除、分页、跳转、下拉刷新、`unread_count` 回写），保持现有 `async-page-state` 与身份键写法。删除旧页面及 `subPages/notification` 分包注册，同步更新引用它的测试和 `scripts/check-notification-payloads.mjs` 的路径。
- 刷新规则沿用 `statsDetail` / `friendHomepage` 的 scope-version 写法：记下加载时的 `userId:authGeneration` 和 `versionOf('notification')`，`onShow` 时两者都没变就不回源。页面可见期间监听这个版本，变化后防抖（约 500ms）刷新首屏。`onShow` 时调用 `activityStore.fetch`，由 TTL / dirty 决定是否真正回源，不强制。
- 未登录时显示登录引导，不触发任何用户级读取。

### D8：角标只显示未读数，待处理事项用红点
- 规则见 spec。不把「未读 + 好友申请 + 收到邀请」加总，是因为默认偏好下一次好友申请或约球邀请本身就会产生一条未读通知，加总会让一件事显示成 2（现在「我的」铃铛就有这个问题）。红点覆盖两种情况：已读但尚未处理，以及关闭了该类通知偏好。
- 鸿蒙：下标 2（`midButton` 不占下标）。显示数字用 `setTabBarBadge`，显示红点用 `showTabBarRedDot`，清除用 `removeTabBarBadge`。文本上限为「99+」，因为运行时会把 4 个及以上的字符替换成「...」。
- 小程序：把 `badge` 通过 `setData` 下发给当前页的自定义 TabBar，由组件渲染数字或红点。

### D9：统一显隐和底部预留
- Tab 页里直接调用 `uni.hideTabBar` / `uni.showTabBar` 的地方（首页和「我的」的二维码弹层）改为 `setMainTabBarVisible`：小程序下发 `hidden`，鸿蒙调用原生接口；页面卸载时恢复显示。
- 小程序给四个 Tab 页预留 `50px + env(safe-area-inset-bottom)`，用一个样式变量统一表达。「我的」浮动奖励条等固定在底部的元素改用这个变量（原来用的 `--window-bottom` 在小程序上恒为 0）。鸿蒙原生栏不占 webview，变量为 0。

### D10：验证方式
- Node 测试覆盖：
  - 角标与 Tab 状态的纯函数；
  - 两端的同步分支（注入假的 `uni`、页面和 `getTabBar`）；
  - 原生组件的点击逻辑（为 `Component` / `wx` 打桩加载 `custom-tab-bar/index.js`）；
  - 扫码分流、重复触发、身份变化、交接槽只能取一次、「再来一局」锁对手；
  - 确认页挂载；
  - 首页和消息 Tab 的请求图（首次进入、重复 `onShow`、切号、迟到响应）；
  - 观赛大厅删掉内容后的源码约定。
- 平台验收：微信开发者工具和小程序真机；鸿蒙模拟器或真机。结果写入 `docs/testing/`，只记录实际执行过的平台。

## Risks / Trade-offs

- [鸿蒙 `midButton` 在实际设备上的渲染或点击与运行时源码不一致] → 先做鸿蒙验证再铺开（tasks 第 1 组）；如果不可用，鸿蒙暂时在首页头部放扫码入口，复用同一流程，不阻塞小程序。
- [小程序自定义 TabBar 每页一个实例，首次切到某个 Tab 时可能闪一下] → 组件只渲染本地图片和文字，不做网络或计算；选中态在 `onShow` 时立即同步。
- [自定义模式下小程序 `hideTabBar` 等接口行为不确定] → 不依赖这些接口，统一用 `setData({ hidden })`。
- [自定义 TabBar 与页面弹层的层级] → 用普通 view，z-index 低于页面弹层；打开弹层时同时隐藏 TabBar。
- [鸿蒙角标只能在 Tab 页设置] → 每次显示 Tab 页时都重新同步；在计分页等非 Tab 页期间的计数变化，返回后补齐。
- [先扫后选多了一步确认] → 换来开局前能确认对手、避免误开局；确认页上的默认球种和类型都已预选，通常只需点一次「开始对局」。
- [鸿蒙 `midButton` 没有可访问名称] → 记录为已知限制；如果验收要求，可以评估给它加上 `text: "扫码"`。
- [新版本里不再读取旧的 `pending_match_rematch`] → 这个值只在点击「再来一局」到下次显示观赛页之间存在，升级时恰好残留的概率很低，残留也不影响功能，不做迁移清理。

## Migration Plan

1. 无后端、数据库或接口迁移。
2. 先完成鸿蒙 `midButton` 和小程序 `custom-tab-bar` 的最小骨架，在两端分别跑通切换、中间按钮点击、角标和主题，再迁移页面。
3. 小程序和鸿蒙各自构建发布，互不依赖。小程序按既有流程用 miniprogram-ci 上传开发版本后，由用户自行提审。
4. 回滚：回退客户端版本即可，服务端状态和通知数据都不受影响。

## Open Questions

- 消息 Tab 图标和中间扫码按钮的正式设计稿：先按现有风格生成可替换的图，之后换成设计稿。
- 首页「正在进行」是否要对已登录用户优先展示好友对局（本次只做公开进行中）。
- 鸿蒙中间按钮是否需要显示「扫码」文字（取决于验收时的可访问性检查）。
