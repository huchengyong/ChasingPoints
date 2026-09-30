# 原生 iOS 用户资料与基础设置验收

对应变更：`openspec/changes/archive/2026-09-30-migrate-native-ios-profile-basics`（已归档）。
范围：原生“我的”资料卡、昵称编辑、基础设置（主题 / 用户协议 / 隐私政策 / 退出确认）。
不在范围：头像上传、手机号绑定、战绩、段位、会员、通知和其他 Tab 业务页面。

## 1. 当前结论

| 层级 | 状态 | 说明 |
| --- | --- | --- |
| 构建 | 通过 | Debug / Release 均可编译，最低部署目标保持 iOS 17.0 |
| 可控原生测试 | 通过 | 55 个 XCTest 用例全绿（原认证 45 + 本期资料 10） |
| iOS 17.x 运行 | 通过（有限） | iPhone 15 Pro 模拟器 iOS 17.5 安装、启动与系统深色外观正常；未覆盖 iOS 17.0–17.4 与真机 |
| 模拟器人工界面验收 | 通过 | 2026-09-30 由用户在 iPhone 16 Pro 模拟器 iOS 18.1 上人工完成 A–H，全部通过 |
| 真实开发账号保存验收 | 通过 | 专用开发账号完成保存、返回同步与重启恢复，验收后已恢复原昵称 |

本期所有验收项已执行并通过（2026-09-30）。

## 2. 接口契约基线（已核对源码）

核对依据：`backend/chasing_points.api`、`backend/internal/logic/user/update_user_profile_logic.go`、
`backend/internal/logic/user/get_user_info_logic.go`、`backend/internal/logic/user/profile_payload.go`。

| 接口 | 请求 | 成功响应 | 失败形态 |
| --- | --- | --- | --- |
| `GET /api/user/info` | `Authorization: Bearer <access_token>` | HTTP 200 顶层 `success`、`user_info` | 普通过期 JWT 为 HTTP 401 空体；用户被删除/停用为 HTTP 401 JSON + `reason: SESSION_INVALID` |
| `POST /api/user/profile` | JWT 认证；本期仅发 `nickname` | HTTP 200 顶层 `success`、`message`、完整 `user_info` | 业务失败为 HTTP 200 + `success:false` + `message`；普通 401 与 `SESSION_INVALID` 复用既有认证处理 |

`user_info` 字段为 `id`、`phone`（服务端脱敏）、`nickname`、`avatar`、`status`、`created_at`。
服务端对昵称执行 `TrimSpace` 后按 Go rune 校验 2–12 个字符；省略头像时保留服务端当前值。
客户端实现只发送 `nickname`，不携带本地头像，也不新增资料读取路径。

测试夹具位于 `iOS/iOSAuthTests/AuthFixtures.swift` 的 `updateProfileResponse` / `updateProfileFailure`，全部为虚构数据。

## 3. 构建与运行环境

| 项目 | 值 |
| --- | --- |
| Xcode | 27.0（Build 27A266a） |
| macOS | 27.0（Build 26A428） |
| 自动化验收设备 | iPhone 16 Pro 模拟器，iOS 18.1（22B81） |
| 额外运行检查设备 | iPhone 15 Pro 模拟器，iOS 17.5 |
| 最低部署目标 | iOS 17.0（Debug/Release 均为该值） |

环境隔离（源码 `AppConfiguration`，二进制核对）：

| 构建 | 后端地址 | Keychain service |
| --- | --- | --- |
| Debug | `https://dev-api.kekemate.cn` | `ChasingPoints.Session.development` |
| Release | `https://api.zhuifen.cn` | `ChasingPoints.Session.production` |

Debug 二进制仅含开发地址与服务，Release 二进制仅含正式地址与服务；客户端不包含短信供应商密钥或 JWT 签名密钥。

## 4. 已执行验收矩阵（2026-09-30）

### 4.1 构建与部署目标（通过）

| 场景 | 命令 | 结果 |
| --- | --- | --- |
| Debug 测试构建 | `xcodebuild build-for-testing -project iOS.xcodeproj -scheme iOS -configuration Debug -destination 'platform=iOS Simulator,name=iPhone 16 Pro,OS=18.1' -derivedDataPath <临时目录>` | 通过 |
| Release 编译 | `xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Release -destination 'generic/platform=iOS Simulator' -derivedDataPath <临时目录> CODE_SIGNING_ALLOWED=NO` | 通过 |
| 最低部署目标 | `xcodebuild -showBuildSettings ... \| grep IPHONEOS_DEPLOYMENT_TARGET` | 17.0 |
| 环境隔离 | `strings` 检查 Debug 应用与 Release 二进制 | Debug 仅含 `dev-api.kekemate.cn` / `ChasingPoints.Session.development`；Release 仅含 `api.zhuifen.cn` / `ChasingPoints.Session.production` |

### 4.2 可控原生测试（通过）

```bash
cd iOS
xcodebuild test-without-building -xctestrun <构建产物>.xctestrun \
  -destination 'platform=iOS Simulator,name=iPhone 16 Pro,OS=18.1' \
  -parallel-testing-enabled NO
```

2026-09-30 执行：**55 个用例，0 失败**。

| 用例组 | 数量 | 覆盖内容 |
| --- | --- | --- |
| `APIClientTests` | 16 | 认证接口解析、错误分类与提示过滤（未改动） |
| `SessionStoreLoginTests` | 7 | 登录/发送验证码与失效处理（未改动） |
| `SessionStoreRestoreTests` | 6 | 启动恢复、重试与去重（未改动） |
| `SessionStoreRefreshTests` | 8 | 401 刷新、一次重放与并发（未改动） |
| `SessionStoreIsolationTests` | 4 | 退出/切号隔离（未改动） |
| `KeychainSessionTests` | 4 | Keychain 保存/读取/删除（未改动） |
| `SessionStoreProfileTests`（新增） | 10 | 见下 |

`SessionStoreProfileTests` 覆盖：

- 昵称边界：trim、2/12 个 Unicode scalar 合法，空值与 1/13 个 scalar 拒绝；中文、emoji、组合序列按 scalar 计数。
- 请求字段：只发送 `nickname`，自动去除两端空白；恢复后再无额外 `/api/user/info` GET。
- 成功同步：只以服务端返回的完整 `user_info` 回写会话。
- 业务失败、网络失败、`user_info` 缺失/身份不匹配/停用响应均不发布成功。
- 恢复未完成前编辑被拒绝且不产生请求。
- 401 刷新后带同一请求体重放一次。
- 保存中退出（迟到成功）与保存中切号（迟到失败）均被丢弃，不污染新身份。

测试宿主隔离：`.xctestrun` 中 `CHASING_POINTS_AUTH_TEST_HOST = 1`，宿主不创建正常应用 `SessionStore`，也不执行真实恢复；Keychain 测试使用独立 service 并自行清理。自动化不访问真实账号。

### 4.3 iOS 17.x 运行检查（通过，范围有限）

在 iPhone 15 Pro 模拟器（iOS 17.5）安装并启动 Debug 构建：应用正常启动、首页渲染正常；切换系统外观为深色后应用整体进入暗色主题且无崩溃。该检查证明二进制可在 iOS 17.x 运行且暗色主题链路可用，但不能替代“我的 / 设置 / 编辑”页面的逐项人工验收，也不覆盖真机与 iOS 17.0–17.4。

## 5. 人工/真实后端验收结果（已执行）

执行时间：2026-09-30；执行方式：用户人工操作；环境：iPhone 16 Pro 模拟器（iOS 18.1），Debug 构建（开发后端 `https://dev-api.kekemate.cn`）。

### 5.1 模拟器界面人工验收（对应任务 4.2）— 通过

| 编号 | 场景 | 结果 |
| --- | --- | --- |
| A | 三种主题 | 通过 |
| B | 头像状态（空值 / 无效 / 加载失败） | 通过 |
| C | 昵称编辑（校验、未修改、保存中防重复、失败重试、成功返回） | 通过 |
| D | 取消编辑不改变资料 | 通过 |
| E | 用户协议 / 隐私政策跳转 | 通过 |
| F | 退出确认、取消保留、确认回访客页且主题保留 | 通过 |
| G | 登录恢复、失败重试与会话失效引导 | 通过 |
| H | 普通 Tab 切换与设置/编辑进出无新增 `/api/user/info` GET | 通过 |

### 5.2 真实开发账号保存验收（对应任务 4.3）— 通过

| 步骤 | 结果 |
| --- | --- |
| 专用开发账号短信登录，记录验收前昵称 | 通过 |
| 编辑昵称保存（请求体只含 `nickname`） | 通过 |
| 返回“我的”立即同步新昵称 | 通过 |
| 杀进程重启后仍为新昵称 | 通过 |
| 验收后恢复原昵称 | 已恢复 |

验收后核对：模拟器 App 容器于验收当日更新；Keychain 无 `ChasingPoints` 遗留凭据。

## 6. 安全与数据边界

- 源码、夹具与本文档不包含真实 token、验证码、私钥或完整手机号。
- 自动化测试只使用虚构数据与受控响应，Keychain 测试结束后删除自身 service 数据。
- 真实联调仅使用已确认的开发环境与专用账号，验证码由测试人员在 App 内输入。
- 不通过改写客户端 JWT 或本地伪造响应替代真实保存验收。
