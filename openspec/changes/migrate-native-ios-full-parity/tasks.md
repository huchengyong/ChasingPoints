每项任务的流程：写分阶段规格 → opencode 中的 DeepSeek 编写 → 复核 diff、复跑 Debug/Release 构建与全部测试、查看快照 → 问题写成补丁规格返工 → 提交。勾选表示已复核并提交。

## 1. 骨架与基础设施

- [x] 1.1 R1a：主题语义色、运行配置与合规开关、`authorizedPost` / `optionalAuthRequest`、`ResourceLoader`、`APIQuery`、CDN 资源与默认头像、会员身份规则及测试
- [x] 1.2 R1b：`AppRoute` / `AppNavigator`、五个主 Tab、共享 UI 组件、Toast、「我的」页搬家、快照测试支架
- [x] 1.3 更新 `iOS/AGENTS.md` 的关键入口（目录结构、路由、读取模型、快照用法）

## 2. 「我的」及其子页面

- [x] 2.1 「我的」主页：资料头部、段位卡、生涯数据、快捷入口、服务列表、我的匹配二维码（`pages/user/index`）
- [x] 2.2 编辑资料（头像上传、昵称等）与设置（通知管理、隐私、黑名单入口、账号注销、数据导出）（`subPages/user/editProfile`、`settings`、`notification`）
- [x] 2.3 段位说明、信誉明细（`rankExplain`、`reputation`）
- [x] 2.4 竞技分析、表现分析（`statsDetail`、`performance`）
- [x] 2.5 对局记录对齐小程序（筛选等差异）、过往对手、交锋记录（`matchHistory`、`opponentRecord`、`h2hRecord`）
  - 过往对手、交锋记录已完成；对局记录对齐并入 2.9 那一轮。「查看对方战绩」（带目标用户的过往对手）还没有入口，随好友主页一起做。
- [x] 2.6 荣誉墙与成就详情（`subPages/achievement/*`）
- [x] 2.7 赛季、赛季报告、历届赛季（`subPages/season/*`）
- [x] 2.8 装备：我的装备、添加、详情、TA 的装备（`subPages/gear/*`）
- [x] 2.9 VIP 中心（合规模式下只读权益，不含购买）（`memberCenter`）

## 3. 首页及其入口

- [x] 3.1 首页：开局卡、约球卡、正在进行、赛事、排行榜、附近球房（`pages/index/index`）
- [x] 3.2 排行榜（`pages/ranking/index`）
- [x] 3.3 观赛大厅（`pages/match/index`）
- [x] 3.4 球馆：列表、详情、上传（定位、地图选点）（`subPages/venue/*`）
- [x] 3.5 规则百科、规则详情、术语词典（`subPages/rules/*`）

## 4. 赛讯

- [x] 4.1 赛讯 Tab：赛事资讯与赛程（`pages/tournament/index`）
- [x] 4.2 赛事详情、对阵图（`subPages/tournament/detail`、`bracket`；`create` 受合规开关控制）
  - 赛事详情已完成；小程序当前没有页面链接到对阵图与创建赛事（后者受合规开关控制），原生端暂不迁移。

## 5. 消息与社交

- [x] 5.1 消息 Tab 与角标（通知列表、已读、删除；未读数 / 红点规则）（`pages/message/index`）
- [x] 5.2 好友：列表、申请、添加、好友主页、黑名单（`subPages/social/friendList`、`friendRequests`、`addFriend`、`friendHomepage`、`blacklist`）
- [x] 5.3 约球：发起、等待、约球列表（`challengeCompose`、`challengeWaiting`、`subPages/social/challenges`）
- [x] 5.4 PK 报表（`subPages/social/pkReport`）
- [ ] 5.5 动态广场、发布、我的动态（受合规开关控制，优先级最低）（`feed`、`postCreate`、`myPosts`）
  - 2026-11-14 核对：小程序当前没有任何页面链接到这三页（合规模式下入口全部隐藏，`buildFeedUrl` 只在页面内部使用），原生端暂不迁移；合规开关放开、小程序恢复入口时再做。
  - 2026-11-15 复核：`subPages/tournament/create`、`bracket` 与 `pages/social/index` 同样没有任何入口（只有接口封装），一并不迁移。

## 6. 对局核心

- [x] 6.1 实时通道：WebSocket 客户端、票据、重连、会话失效处理（`utils/websocket*.js`）
  - 客户端已迁移并有测试；接入应用生命周期与页面放在 5.1（用户通道）和 6.4（对局通道）。
- [x] 6.2 计分规则与视图模型迁移（中八、九球、斯诺克、赛制、结束倒计时）及对应测试（`utils/match-*.js`、`snooker-*.js`、`pool-match-format.js`、`finish-countdown.js`）
- [x] 6.3 扫码、选择球友、确认对局（相机权限、匹配码与裁判码）（`quick-scan`、`opponentSelector`、`scanConfirm`）
- [x] 6.4 实时计分页：选手视角（`subPages/match/playing`）
- [x] 6.5 实时计分页：裁判视角与观赛视角
- [x] 6.6 赛果、对局详情对齐、单场复盘、本场装备（`matchResult`、`matchDetail`、`matchReplay`、`matchGearCard`）
  - 对局详情按小程序区分观战（`.publicMatchDetail`）、交锋记录（`source=history` + 视角用户）与参与者三种进入方式；进行中连对局通道。
- [x] 6.7 战绩海报与系统分享、邀请链接入口（`shareResult`、`linkEntry`）
  - 标准 / 会员两套海报模板；`onOpenURL` 与网页续接已处理，Universal Links 的关联域名待工程与服务器配置。

## 7. 收尾

- [x] 7.1 欢迎页与登录流程对齐（协议同意、未登录引导）（`pages/welcome`、`pages/login`）
- [x] 7.2 帮助与反馈、用户协议、隐私政策应用内页面（`subPages/help/feedback`、`subPages/agreement/*`）
  - 欢迎页、协议确认弹层、登录页、设置页的协议链接都改为打开应用内协议页。
- [x] 7.3 权限说明、隐私清单与工程配置核对（相机、定位、相册）
  - 相机、定位、相册（仅添加）用途说明已在工程配置；`PrivacyInfo.xcprivacy` 声明不跟踪、UserDefaults 原因 CA92.1。收集数据类型：手机号、昵称、用户 ID、头像照片、精确位置、对局数据、用户内容、反馈工单（2026-11-15 用户确认补上位置）。
- [x] 7.4 全量回归：Debug/Release 构建、全部测试、三种主题下的快照走查，记录到 `docs/testing/native-ios-full-parity.md`
- [ ] 7.5 真实后端只读联调与真机验收（需用户授权后执行，结果与模拟响应分开记录）
  - 2026-11-15 真实后端只读验收已完成（用户授权两个专用开发账号，测试环境，模拟器）；真机验收未执行。
