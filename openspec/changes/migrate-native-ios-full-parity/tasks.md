每项任务的流程：写分阶段规格 → opencode 中的 DeepSeek 编写 → 复核 diff、复跑 Debug/Release 构建与全部测试、查看快照 → 问题写成补丁规格返工 → 提交。勾选表示已复核并提交。

## 1. 骨架与基础设施

- [x] 1.1 R1a：主题语义色、运行配置与合规开关、`authorizedPost` / `optionalAuthRequest`、`ResourceLoader`、`APIQuery`、CDN 资源与默认头像、会员身份规则及测试
- [x] 1.2 R1b：`AppRoute` / `AppNavigator`、五个主 Tab、共享 UI 组件、Toast、「我的」页搬家、快照测试支架
- [x] 1.3 更新 `iOS/AGENTS.md` 的关键入口（目录结构、路由、读取模型、快照用法）

## 2. 「我的」及其子页面

- [x] 2.1 「我的」主页：资料头部、段位卡、生涯数据、快捷入口、服务列表、我的匹配二维码（`pages/user/index`）
- [x] 2.2 编辑资料（头像上传、昵称等）与设置（通知管理、隐私、黑名单入口、账号注销、数据导出）（`subPages/user/editProfile`、`settings`、`notification`）
- [ ] 2.3 段位说明、信誉明细（`rankExplain`、`reputation`）
- [ ] 2.4 竞技分析、表现分析（`statsDetail`、`performance`）
- [ ] 2.5 对局记录对齐小程序（筛选等差异）、过往对手、交锋记录（`matchHistory`、`opponentRecord`、`h2hRecord`）
- [ ] 2.6 荣誉墙与成就详情（`subPages/achievement/*`）
- [ ] 2.7 赛季、赛季报告、历届赛季（`subPages/season/*`）
- [ ] 2.8 装备：我的装备、添加、详情、TA 的装备（`subPages/gear/*`）
- [ ] 2.9 VIP 中心（合规模式下只读权益，不含购买）（`memberCenter`）

## 3. 首页及其入口

- [ ] 3.1 首页：开局卡、约球卡、正在进行、赛事、排行榜、附近球房（`pages/index/index`）
- [ ] 3.2 排行榜（`pages/ranking/index`）
- [ ] 3.3 观赛大厅（`pages/match/index`）
- [x] 3.4 球馆：列表、详情、上传（定位、地图选点）（`subPages/venue/*`）
- [x] 3.5 规则百科、规则详情、术语词典（`subPages/rules/*`）

## 4. 赛讯

- [ ] 4.1 赛讯 Tab：赛事资讯与赛程（`pages/tournament/index`）
- [ ] 4.2 赛事详情、对阵图（`subPages/tournament/detail`、`bracket`；`create` 受合规开关控制）

## 5. 消息与社交

- [ ] 5.1 消息 Tab 与角标（通知列表、已读、删除；未读数 / 红点规则）（`pages/message/index`）
- [ ] 5.2 好友：列表、申请、添加、好友主页、黑名单（`subPages/social/friendList`、`friendRequests`、`addFriend`、`friendHomepage`、`blacklist`）
- [ ] 5.3 约球：发起、等待、约球列表（`challengeCompose`、`challengeWaiting`、`subPages/social/challenges`）
- [ ] 5.4 PK 报表（`subPages/social/pkReport`）
- [ ] 5.5 动态广场、发布、我的动态（受合规开关控制，优先级最低）（`feed`、`postCreate`、`myPosts`）

## 6. 对局核心

- [ ] 6.1 实时通道：WebSocket 客户端、票据、重连、会话失效处理（`utils/websocket*.js`）
- [ ] 6.2 计分规则与视图模型迁移（中八、九球、斯诺克、赛制、结束倒计时）及对应测试（`utils/match-*.js`、`snooker-*.js`、`pool-match-format.js`、`finish-countdown.js`）
- [ ] 6.3 扫码、选择球友、确认对局（相机权限、匹配码与裁判码）（`quick-scan`、`opponentSelector`、`scanConfirm`）
- [ ] 6.4 实时计分页：选手视角（`subPages/match/playing`）
- [ ] 6.5 实时计分页：裁判视角与观赛视角
- [ ] 6.6 赛果、对局详情对齐、单场复盘、本场装备（`matchResult`、`matchDetail`、`matchReplay`、`matchGearCard`）
- [ ] 6.7 战绩海报与系统分享、邀请链接入口（`shareResult`、`linkEntry`）

## 7. 收尾

- [ ] 7.1 欢迎页与登录流程对齐（协议同意、未登录引导）（`pages/welcome`、`pages/login`）
- [ ] 7.2 帮助与反馈、用户协议、隐私政策应用内页面（`subPages/help/feedback`、`subPages/agreement/*`）
- [ ] 7.3 权限说明、隐私清单与工程配置核对（相机、定位、相册）
- [ ] 7.4 全量回归：Debug/Release 构建、全部测试、三种主题下的快照走查，记录到 `docs/testing/native-ios-full-parity.md`
- [ ] 7.5 真实后端只读联调与真机验收（需用户授权后执行，结果与模拟响应分开记录）
