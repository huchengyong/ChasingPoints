## Why

UniApp 当前底部是「首页 | 观赛 | 赛讯 | 我的」。观赛页顶部的「我的对局 / 来一局」卡与首页几乎重复；「扫码开局」的逻辑在首页、观赛、我的三处各写了一份，「扫码执裁」只能从观赛页进入；消息只是「我的」头部的一个铃铛，未读提醒同时出现在首页头部、我的铃铛和观赛「收到的邀请」三个地方。UniApp 现在只编译微信小程序和鸿蒙，iOS / Android 之后会改为原生开发，所以这次导航调整只需要在这两个端上成立。

## What Changes

- **BREAKING（导航）** 底部导航改为「首页 | 赛讯 | (扫码) | 消息 | 我的」。观赛不再是 Tab；中间的扫码按钮不对应任何页面，点击就开始扫码。小程序端用自定义 TabBar 实现，鸿蒙端用原生 `midButton` 实现，两端的样式、主题和角标保持一致。
- 观赛并入首页：首页新增「正在进行」区块（最多 3 场公开进行中对局）和常驻的「观赛大厅」入口。原 `pages/match/index` 保留路径，改为普通页面「观赛大厅」，只负责好友 / 进行中 / 已结束列表、球种筛选和分页；它与首页重复的开局卡、约球入口、扫码和弹窗全部移除。
- 扫码统一为「先扫后选」：中间扫码按钮、首页「发起对局 → 扫描对手二维码」、排行榜页「发起PK」、赛果页「再来一局」四个入口共用同一流程。扫到的码按类型分流（对手匹配码 / 裁判码）。之后进入扫码确认页：对手码确认对手并选择球种和对局类型，裁判码预览对局后确认执裁。确认后才创建对局或加入裁判。
- 新增「消息」Tab（主包页面）：顶部是「好友申请」「约球邀请」两个入口，下面是通知列表。原 `subPages/notification/index` 的功能迁入这里，旧页面删除。Tab 角标规则：有未读通知时显示未读数；没有未读、但有待处理好友申请或收到的约球邀请时显示红点。
- 入口迁移：约球记录移到「我的」，收到的邀请移到「消息」，扫码开局和扫码执裁交给中间扫码按钮。首页头部的消息图标 / 未读提示、「我的」头部的扫码和铃铛全部移除；「出示我的二维码」继续保留在「我的」头部和首页「发起对局」里。
- 跳转到原观赛 Tab 的 `switchTab` 一律改掉：约球发送后回首页；赛果页「再来一局」直接走统一扫码流程，不再写 `pending_match_rematch` 交接。
- 不涉及后端接口、数据库、管理后台、官网和原生 iOS。

## Capabilities

### New Capabilities

- `client-main-navigation`：UniApp（微信小程序、鸿蒙）主导航结构，包括五个位置的顺序与平台实现、观赛并入首页与观赛大厅、入口迁移、消息 Tab 的聚合内容与刷新、消息角标规则。
- `universal-scan-entry`：各扫码入口共用的「先扫后选」流程，包括按码分流、扫码确认页、登录与身份隔离、「再来一局」锁定对手。

### Modified Capabilities

- `quick-match-invites`：「双入口、记录与通知读取同一权威状态」原先要求对局页始终保留扫码开局、约球记录和收到邀请入口；改为由首页、消息 Tab、「我的」和中间扫码按钮分别承接。
- `read-api-composition`：「用户 Bootstrap 必须提供共享首屏状态」中复用 Bootstrap 的页面，由「两个主 Tab」改为首页、消息 Tab 与「我的」。
- `auth-safe-client-read-coordination`：「App 前台恢复必须通过一次 Bootstrap 校准共享资源」中的多 Tab 复用场景，由「首页、观赛、我的」改为「首页、消息、我的」。

## Impact

- UniApp 配置：`pages.json`（Tab 列表与顺序、小程序 `custom`、鸿蒙 `midButton`、新增 `pages/message/index`、`pages/match/index` 改为普通页、删除 `subPages/notification` 分包、新增扫码确认页）、`theme.json`（消息图标变量，移除观赛图标变量）、`static/tabbar/`（新增消息图标和扫码按钮图，删除观赛图标）。
- UniApp 新增：`custom-tab-bar/`（微信原生自定义组件，只拷贝进小程序产物）、主导航同步工具（选中态、角标、主题、显隐）、统一扫码流程与 `subPages/match/scanConfirm`、`pages/message/index`。
- UniApp 修改：`pages/index`、`pages/match`、`pages/user`、`pages/tournament`（底部预留与导航同步）、`pages/ranking`（「发起PK」改走统一扫码）、`App.vue`（鸿蒙中间按钮监听）、`utils/theme-application.js`、`store/publicRead.js`（首页公开进行中对局的短 TTL 缓存）、`subPages/match/matchResult.vue`、`subPages/match/challengeCompose.vue`，以及对应的 Node 测试。
- 文档：`DESIGN.md` 的 TabBar / 弹层条目，`uniapp/AGENTS.md` 与 `GEMINI.md`（保持镜像），`uniapp/subPages/AGENTS.md` 中通知中心的位置说明；验收记录放 `docs/testing/`。
- 平台：只验收微信小程序和鸿蒙。鸿蒙 `midButton` 已从 HBuilderX 4.76 自带运行时源码确认支持，但仍需先做真机 / 模拟器验证。原生 iOS 目前是「首页 / 观赛 / 赛讯 / 我的」占位 Tab，本次不改，记录给 iOS 迁移工作。
- 后端：无改动。首页预览复用公开接口 `GET /api/public/matches`（`scope=hall&status=1`）；消息 Tab 复用现有通知列表、Bootstrap 计数和 WebSocket 计数更新。
