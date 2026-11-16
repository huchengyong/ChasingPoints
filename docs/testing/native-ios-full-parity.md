# 原生 iOS 追平小程序：全量回归记录

对应变更：[openspec/changes/migrate-native-ios-full-parity/](../../openspec/changes/migrate-native-ios-full-parity/)（进度见其中的 `tasks.md`）。
范围：原生 SwiftUI 用户端（`iOS/`）与微信小程序（`uniapp/`）页面几乎一一对应，允许平台差异。
第 4 节是**模拟器 + 模拟响应**的自动化验证；第 5 节是经用户授权、用两个专用开发账号在**测试后端**上做的只读验收，两者分开记录。真机验收尚未执行。

## 1. 当前结论

| 层级 | 状态 | 说明 |
| --- | --- | --- |
| 构建 | 通过 | Debug / Release 均可编译（`generic/platform=iOS Simulator`），最低部署目标 iOS 17.0 |
| 自动化测试（iOS 17.5） | 通过 | iPhone 15 模拟器，1512 个 XCTest 用例全绿（提交 `39fe92e`） |
| 自动化测试（iOS 18.1） | 通过 | iPhone 16 Pro 模拟器，1512 个 XCTest 用例全绿（提交 `0090141`） |
| 快照走查 | 通过（有限） | 170 个页面状态各导出亮 / 暗两张，共 340 张，已逐模块人工查看；快照只断言能渲染，不做像素比对 |
| 真实后端只读验收 | 通过 | 测试环境、两个专用开发账号，11 个用例 10 过 1 跳过（测试库无球馆），见第 5 节 |
| 真机 | 未执行 | 需要真机人工验收 |
| Universal Links | 已配置，待真机验证 | 签名团队改为 `BBFNTFR58V`，entitlements 声明 `applinks:www.zhuifen.cn`；官网 `/.well-known/apple-app-site-association` 已上线（200、`application/json`），苹果 CDN 已抓取 |

## 2. 页面对应

`AppRoute` 的每个 case 都在 `AppRouteDestination` 登记了真实页面（`switch` 不留 `default`，迁移占位页已删除）。小程序 `pages.json` 中以下页面**不迁移**：

| 页面 | 原因 |
| --- | --- |
| `subPages/social/feed`、`postCreate`、`myPosts` | 小程序内没有任何入口（合规模式下隐藏，`buildFeedUrl` 只在测试中使用） |
| `subPages/tournament/create`、`bracket` | 小程序内没有入口，只有接口封装 |
| `pages/social/index` | 不在 tabBar，也没有任何跳转 |

合规开关恢复入口时再迁移，见 `tasks.md` 5.5。

## 3. 构建与运行环境

| 项目 | 值 |
| --- | --- |
| Xcode | 27.0（Build 27A266a） |
| macOS | 27.0（Build 26A428） |
| 测试设备 | iPhone 15 模拟器 iOS 17.5；iPhone 16 Pro 模拟器 iOS 18.1 |
| 最低部署目标 | iOS 17.0 |
| 网络 | 全部测试经 `MockHTTPServer` / `MockURLProtocol`，夹具为虚构数据，不访问真实后端 |

## 4. 已执行验证（2026-11-15）

### 4.1 构建

在 `iOS/` 执行：

```bash
xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO
xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Release \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO
```

结果：两者均 `BUILD SUCCEEDED`；Release 产物包含 `PrivacyInfo.xcprivacy`。

### 4.2 全量测试

在 `iOS/` 执行（`TEST_RUNNER_SNAPSHOT_DIR` 可选，用于导出快照）：

```bash
TEST_RUNNER_SNAPSHOT_DIR=<输出目录> xcodebuild test -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination "platform=iOS Simulator,id=<模拟器 UDID>" -parallel-testing-enabled NO
```

| 设备 | 提交 | 结果 |
| --- | --- | --- |
| iPhone 15，iOS 17.5 | `39fe92e` | `Executed 1512 tests, with 0 failures`，`TEST SUCCEEDED` |
| iPhone 16 Pro，iOS 18.1 | `0090141` | `Executed 1512 tests, with 0 failures`，`TEST SUCCEEDED` |
| iPhone 15，iOS 17.5 | `1c25255` | `Executed 1525 tests, with 11 tests skipped and 0 failures`（跳过的是未提供 token 的真实后端验收类），`TEST SUCCEEDED` |

`0090141` 相对 `39fe92e` 只删除了已无引用的迁移占位页并更新注释与指南，Debug / Release 构建已复核。

### 4.3 快照走查

每个页面至少覆盖有数据、空、失败三种状态中适用的部分，亮 / 暗两种外观；「跟随系统」外观在运行时解析为二者之一，不单独出图。走查时重点核对：文案与小程序逐字一致、数字不带千分位、深色模式下的对比度、海报（PK 报表、战绩海报标准 / 会员模板）的版式与配色。

## 5. 真实后端只读验收（2026-11-15，用户授权）

| 项目 | 值 |
| --- | --- |
| 环境 | 测试后端 `https://api-zhuifen.kekemate.cn`（服务器 `APP_ENV=dev`，库 `chasing_points_dev`） |
| 账号 | 用户指定的两个专用开发账号（用户 2、用户 4）；本文档不记录手机号与 token |
| 登录方式 | 经用户同意绕过短信：在测试服务器上用服务端签名密钥为两个账号签发 3 小时有效的 access / refresh token；密钥未离开服务器，token 只保存在本机临时文件并在验收后删除 |
| 设备 | iPhone 15 模拟器 iOS 17.5，Debug 构建 |
| 用例 | [iOSAuthTests/LiveBackendAcceptanceTests.swift](../../iOS/iOSAuthTests/LiveBackendAcceptanceTests.swift)，只发 GET；未提供环境变量时整类跳过 |

运行方式（在 `iOS/` 执行，token 由验收人自行提供，不写入任何文件）：

```bash
TEST_RUNNER_LIVE_API_BASE_URL=https://api-zhuifen.kekemate.cn \
TEST_RUNNER_LIVE_ACCESS_TOKEN_A=<…> TEST_RUNNER_LIVE_REFRESH_TOKEN_A=<…> \
TEST_RUNNER_LIVE_ACCESS_TOKEN_B=<…> TEST_RUNNER_LIVE_REFRESH_TOKEN_B=<…> \
xcodebuild test -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination "platform=iOS Simulator,id=<模拟器 UDID>" -parallel-testing-enabled NO \
  -only-testing:iOSAuthTests/LiveBackendAcceptanceTests
```

结果：`Executed 11 tests, with 1 test skipped and 0 failures`。覆盖我的 / VIP 中心 / 信誉 / 过往对手 / 竞技分析 / 表现分析 / 段位说明 / 通知与隐私设置、对局记录与赛果 / 详情 / 复盘 / 战绩海报 / 本场用杆、首页 / 赛讯 / 规则 / 术语、赛季 / 荣誉墙 / 成就、好友 / 申请 / 黑名单 / 好友主页 / PK 报表 / 交锋记录 / TA 的装备、约球三分栏、消息中心、我的装备，账号 B 的核心页面。每个页面按真实数据导出亮 / 暗快照并人工抽查。

测试库数据现状导致的取舍：球馆 0 个（附近球馆只验证空状态，球馆详情跳过）；两个账号不是好友（好友衍生页面以非好友身份验证，服务端返回「仅可查看好友的对方战绩」或 403，页面正确展示）；装备目录没有上架品牌（只验证空状态）。

验收中发现并已修复：约球页把被取消的请求显示为「请求已取消」；对局记录每行出现系统箭头与卡片箭头两个箭头。

## 6. 平台差异（有意为之）

- 只做手机号验证码登录；不接微信登录、华为登录、APNs 推送与内购。
- 扫码用相机（`AVCaptureDevice`），地图选点用 MapKit，定位做 WGS-84 → GCJ-02 换算后再请求后端。
- 分享走系统分享面板与「保存到相册」（仅添加权限）；依赖微信的分享入口不显示。海报里的二维码是入口链接，不是小程序码。
- 协议、隐私政策为应用内页面；合规开关（`AppFlags`）与小程序同值。
- 小程序的 `redirectTo` 在原生端是推入新页面，返回栈会多一层承接页（邀请链接承接）。

## 7. 未执行 / 待确认

| 项目 | 状态 | 需要 |
| --- | --- | --- |
| 真机验收 | 未执行 | 真机安装 Debug 包，人工走查扫码、相机、相册、分享、定位与实时对局 |
| Universal Links | 待真机验证 | 开发者网站上该 App ID 需开启 Associated Domains（或由 Xcode 自动签名开启）；真机验证冷 / 热启动，从备忘录、短信、Safari 打开 `https://www.zhuifen.cn/go/...`（微信内置浏览器通常会拦截） |
| 隐私清单的收集数据类型 | 已确认 | 手机号、昵称、用户 ID、头像照片、精确位置（附近球馆与上传球馆的坐标）、对局数据、用户内容、反馈工单；均关联用户、不用于跟踪 |
| 相册写入、系统分享面板、相机扫码 | 未在真机验证 | 真机人工验收 |

## 8. 迁移中发现的小程序 / 后端问题（只报告，未修改）

- 规则搜索接口不返回球种与类型，结果行点击会带 `category=undefined`。
- 球馆详情的 `recent_checkins` / `has_checked_in` 后端未返回；签到失败时小程序读 `res.msg`，后端给的是 `message`。
- 九球追分计分页提示「左列记到我方，右列记到对手」与实际布局相反（`playing.vue` 约 936 行）。
- 观赛大厅公开列表未过滤隐藏战绩的用户；`player2_name` 取对局快照而 `player1_name` 取实时昵称。
- 实时比分推送的 `my_score` 是选手 1 视角；约球业务失败以 HTTP 200 + `success:false` 返回。
- 好友搜索结果的 `is_friend` 恒为 false；访客打开交锋记录为空白页；信誉明细刷新失败会清空列表。
- VIP 中心跳转表现分析时不带 `game_type`；趋势点 `rank_score` 为 `omitempty`；排位权益说明依赖中文标签子串匹配。
- 战绩海报保存失败时小程序只打日志、没有提示（原生已补「保存失败」提示）。
