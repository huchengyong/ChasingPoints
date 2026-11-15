# 原生 iOS 追平小程序：全量回归记录

对应变更：[openspec/changes/migrate-native-ios-full-parity/](../../openspec/changes/migrate-native-ios-full-parity/)（进度见其中的 `tasks.md`）。
范围：原生 SwiftUI 用户端（`iOS/`）与微信小程序（`uniapp/`）页面几乎一一对应，允许平台差异。
本记录只覆盖**模拟器 + 模拟响应**的自动化验证；真机与真实后端验收（任务 7.5）尚未执行，需另行授权。

## 1. 当前结论

| 层级 | 状态 | 说明 |
| --- | --- | --- |
| 构建 | 通过 | Debug / Release 均可编译（`generic/platform=iOS Simulator`），最低部署目标 iOS 17.0 |
| 自动化测试（iOS 17.5） | 通过 | iPhone 15 模拟器，1512 个 XCTest 用例全绿（提交 `39fe92e`） |
| 自动化测试（iOS 18.1） | 通过 | iPhone 16 Pro 模拟器，1512 个 XCTest 用例全绿（提交 `0090141`） |
| 快照走查 | 通过（有限） | 170 个页面状态各导出亮 / 暗两张，共 340 张，已逐模块人工查看；快照只断言能渲染，不做像素比对 |
| 真机 / 真实后端 | 未执行 | 需用户授权；结果须与本记录的模拟响应结果分开记录 |
| Universal Links | 未配置 | 代码已处理 `onOpenURL` / 网页续接，关联域名需工程与服务器配置（见第 6 节） |

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

`0090141` 相对 `39fe92e` 只删除了已无引用的迁移占位页并更新注释与指南，Debug / Release 构建已复核。

### 4.3 快照走查

每个页面至少覆盖有数据、空、失败三种状态中适用的部分，亮 / 暗两种外观；「跟随系统」外观在运行时解析为二者之一，不单独出图。走查时重点核对：文案与小程序逐字一致、数字不带千分位、深色模式下的对比度、海报（PK 报表、战绩海报标准 / 会员模板）的版式与配色。

## 5. 平台差异（有意为之）

- 只做手机号验证码登录；不接微信登录、华为登录、APNs 推送与内购。
- 扫码用相机（`AVCaptureDevice`），地图选点用 MapKit，定位做 WGS-84 → GCJ-02 换算后再请求后端。
- 分享走系统分享面板与「保存到相册」（仅添加权限）；依赖微信的分享入口不显示。海报里的二维码是入口链接，不是小程序码。
- 协议、隐私政策为应用内页面；合规开关（`AppFlags`）与小程序同值。
- 小程序的 `redirectTo` 在原生端是推入新页面，返回栈会多一层承接页（邀请链接承接）。

## 6. 未执行 / 待确认

| 项目 | 状态 | 需要 |
| --- | --- | --- |
| 真机与真实后端只读联调（7.5） | 未执行 | 用户授权、专用开发账号；结果单独记录 |
| Universal Links | 未配置 | Target 增加 Associated Domains `applinks:www.zhuifen.cn`；服务器发布 `/.well-known/apple-app-site-association`（`appID` 为 `<TeamID>.<bundle id>`，paths 覆盖 `/go/*`）；真机验证冷 / 热启动与从微信、Safari 打开 |
| 隐私清单的收集数据类型 | 已确认 | 手机号、昵称、用户 ID、头像照片、精确位置（附近球馆与上传球馆的坐标）、对局数据、用户内容、反馈工单；均关联用户、不用于跟踪 |
| 相册写入、系统分享面板、相机扫码 | 未在真机验证 | 真机人工验收 |

## 7. 迁移中发现的小程序 / 后端问题（只报告，未修改）

- 规则搜索接口不返回球种与类型，结果行点击会带 `category=undefined`。
- 球馆详情的 `recent_checkins` / `has_checked_in` 后端未返回；签到失败时小程序读 `res.msg`，后端给的是 `message`。
- 九球追分计分页提示「左列记到我方，右列记到对手」与实际布局相反（`playing.vue` 约 936 行）。
- 观赛大厅公开列表未过滤隐藏战绩的用户；`player2_name` 取对局快照而 `player1_name` 取实时昵称。
- 实时比分推送的 `my_score` 是选手 1 视角；约球业务失败以 HTTP 200 + `success:false` 返回。
- 好友搜索结果的 `is_friend` 恒为 false；访客打开交锋记录为空白页；信誉明细刷新失败会清空列表。
- VIP 中心跳转表现分析时不带 `game_type`；趋势点 `rank_score` 为 `omitempty`；排位权益说明依赖中文标签子串匹配。
- 战绩海报保存失败时小程序只打日志、没有提示（原生已补「保存失败」提示）。
