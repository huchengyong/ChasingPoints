## Why

原生“我的”已完成资料与基础设置，但用户仍无法在原生端回看自己打过的比赛。对局记录与战报只依赖已稳定的只读接口，而对局实时链路仍受 `fix-client-p0-release-blockers`、`implement-quick-match-invites` 的协议变更影响；先迁移只读战绩，可以在不追随变动协议的前提下建立原生分页列表、身份隔离读取与对局数据解码的基础。

## What Changes

- 在原生“我的”为已登录用户增加“对局记录”入口，进入参赛/执裁两个分栏的分页列表，支持首次加载、下拉刷新、加载更多、空态、错误态与刷新失败提示。
- 增加只读战报页：参赛记录与已完成的执裁记录进入战报，展示比分与胜负/平局、球种、结束时间、排位分变化及明细、亮点与统计、裁判卡与完成方式、本场荣誉奖励；裁判视角隐藏排位与选手操作。
- 列表与战报的读取结果绑定当前身份与认证代次；退出、切号或会话失效后旧结果不得显示给新身份。
- 失败展示明确错误与重试，不像 UniApp 战报那样用默认 `0 - 0` 数据掩盖加载失败；平局在战报中按平局展示。
- 本期不迁移：继续执裁卡片、再来一局、战绩海报、交锋记录、荣誉墙、`matchDetail` 局记录/观战及任何 WebSocket 能力；战报中这些入口不展示。

## Capabilities

### New Capabilities

- `native-ios-match-history`：原生对局记录（参赛/执裁）分页列表与只读战报的展示、读取、错误与身份隔离要求。

### Modified Capabilities

无。复用 `native-ios-auth-integration` 的认证请求与会话失效规则、`native-ios-profile-basics` 的“我的”导航结构，以及 `auth-safe-client-read-coordination` 的身份隔离原则；不改变其已有要求。

## Impact

- 原生端：`ContentView.swift`（“我的”入口），新增对局记录列表、战报视图及对应读取模型和 XCTest；继续使用 `SessionStore.authorizedRequest` 与 `AppTheme`。
- 后端：复用 `GET /api/match/list`、`GET /api/match/referee/history`、`GET /api/match/detail`、`GET /api/match/reward-summary`；不新增接口、不改 `.api` 契约、无数据迁移。
- UniApp / admin / website：不改动；UniApp `matchHistory.vue`、`matchResult.vue` 仅作信息组织参考。
- 验证：受控模拟响应的自动化测试、模拟器界面验收，以及专用开发账号对真实历史数据的只读联调，结果分开记录。
