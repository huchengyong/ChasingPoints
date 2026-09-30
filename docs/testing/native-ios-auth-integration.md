# 原生 iOS 认证链路验收

对应变更：`openspec/changes/archive/2026-09-30-complete-native-ios-auth-integration`。
范围：短信发送 → 验证码登录 → 冷启动恢复 → 普通 401 共享刷新与重放 → 退出/账号切换隔离。
不在范围：业务页面迁移、第三方登录、账号绑定/合并、注销、推送、WebSocket。

## 1. 当前结论

| 层级 | 状态 | 说明 |
| --- | --- | --- |
| 构建 | 通过 | Debug / Release 均可编译，最低部署目标保持 iOS 17.0 |
| 可控原生测试 | 通过 | 45 个 XCTest 用例全绿，含并发、迟到响应及实现 review 新增回归 |
| 真实后端人工联调 | 通过 | 真实收码登录、倒计时、杀进程恢复、退出后重启、隔离短 TTL 真实刷新、恢复失败重试与 UI 反馈均已验收 |

**所有真实必验项已通过（2026-09-29 至 2026-09-30），链路闭环。**

## 2. 接口契约基线（已核对源码）

核对依据：`backend/chasing_points.api`、`backend/internal/logic/auth/*`、
`backend/internal/pkg/httperror/error.go`、`backend/internal/middleware/active_user_session.go`。

| 接口 | 请求 | 成功响应 | 失败形态 |
| --- | --- | --- | --- |
| `POST /api/auth/send-sms` | `phone`、`scene: login` | HTTP 200 顶层 `success`、`message` | 频控/发送失败为 HTTP 200 + `success:false` + `message` |
| `POST /api/auth/login` | `phone`、`sms_code` | HTTP 200 顶层 `success`、`access_token`、`refresh_token`、`expires_in`、`user_info` | 验证码错误/过期/系统错误走 `fmt.Errorf`，go-zero 以 HTTP 400 纯文本写出，如 `验证码错误` |
| `POST /api/auth/refresh-token` | `refresh_token` | HTTP 200 顶层新令牌对 | 明确失效为 HTTP 200 + `success:false` + `reason: SESSION_INVALID` |
| `GET /api/user/info` | `Authorization: Bearer <access_token>` | HTTP 200 顶层 `success`、`user_info` | 普通过期 JWT 为 HTTP 401 空体；用户被删除/停用为 HTTP 401 JSON + `reason: SESSION_INVALID` |

`user_info` 字段为 `id`、`phone`（登录响应已脱敏）、`nickname`、`avatar`、`status`、`created_at`。
客户端错误分类与提示见 `APIClient`：上述四条认证接口仅展示已核对的用户提示（包括验证码、频控与会话失效），过滤纯文本和 JSON `message/msg` 中的未知内部错误；5xx、HTML、堆栈、空体使用通用中文提示且不回显原始响应体；网络失败单独分类。

测试夹具位于 `iOS/iOSAuthTests/AuthFixtures.swift`，全部为虚构数据。

## 3. 构建与运行环境

| 项目 | 值 |
| --- | --- |
| Xcode | 27.0（Build 27A266a） |
| macOS | 27.0（Build 26A428） |
| 自动化验收设备 | iPhone 16 Pro 模拟器，iOS 18.1（22B81） |
| 最低部署目标 | iOS 17.0（Debug/Release 均为该值） |

环境隔离（源码 `AppConfiguration`）：

| 构建 | 后端地址 | Keychain service |
| --- | --- | --- |
| Debug | `https://dev-api.kekemate.cn` | `ChasingPoints.Session.development` |
| Release | `https://api.zhuifen.cn` | `ChasingPoints.Session.production` |

客户端不包含短信供应商密钥或 JWT 签名密钥。

## 4. 验收矩阵

### 4.1 构建验收（通过）

| 场景 | 命令 | 结果 |
| --- | --- | --- |
| Debug 编译 | `xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Debug -destination 'platform=iOS Simulator,name=iPhone 16 Pro,OS=18.1'` | 通过 |
| Release 编译 | `xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Release -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO` | 通过 |
| 最低部署目标 | `xcodebuild -showBuildSettings ... \| grep IPHONEOS_DEPLOYMENT_TARGET` | Debug/Release 均为 17.0 |
| 环境隔离 | `strings` 检查 Debug `iOS.debug.dylib` 与 Release `iOS` 二进制 | Debug 仅含 `dev-api.kekemate.cn` / `ChasingPoints.Session.development`；Release 仅含 `api.zhuifen.cn` / `ChasingPoints.Session.production` |

运行系统版本（iOS 18.1）与最低支持版本（iOS 17.0）分别记录，不互相替代。

### 4.2 可控原生测试（通过）

执行命令：

```bash
cd iOS
xcodebuild test -project iOS.xcodeproj -scheme iOS \
  -destination 'platform=iOS Simulator,name=iPhone 16 Pro,OS=18.1' \
  -parallel-testing-enabled NO
```

2026-09-29 连续执行 3 次均 `TEST SUCCEEDED`，当时共 42 个用例。2026-09-30 实现 review 后新增 3 个错误反馈回归，并修复 XCTest 宿主启动隔离；最新 45 个用例全部通过，详见 4.5。测试通过注入 `URLSession`（`MockURLProtocol`）控制响应、状态码、网络错误与并发屏障；Keychain 测试使用独立 service 并在结束时删除自身数据。

共享 scheme 仅在 TestAction 设置 `CHASING_POINTS_AUTH_TEST_HOST=1`，此时宿主不创建正常应用的 `SessionStore`，也不执行真实会话恢复；正常 Run 使用原有应用入口。运行测试前已核对生成的 `.xctestrun` 含该环境标记。

| 用例组 | 覆盖内容 |
| --- | --- |
| `APIClientTests`（16） | 四个接口顶层解析；HTTP 200 业务失败与频控；HTTP 400 错误/过期验证码；400 纯文本及 200 JSON 的 Redis 内部错误不回显；400 HTML 不回显；5xx 通用提示；普通 401 空体；`SESSION_INVALID`（401 与 200）保留 `reason`；未知响应体；断网/超时分类；Debug 地址与 Keychain service |
| `SessionStoreLoginTests`（7） | 发送/登录请求序列化；登录成功持久化并发布用户；纯文本失败不产生半登录；缺令牌/缺用户资料/停用用户被拒；Keychain 保存失败不发布会话 |
| `SessionStoreRestoreTests`（6） | 无凭据启动零请求；恢复必须先验证 `user_info`；并发恢复只验证一次；断网/503 保留凭据并可重试；`SESSION_INVALID` 清理一次并提示 |
| `SessionStoreRefreshTests`（8） | 普通 401 刷新一次并重放一次；交错返回的旧 401 复用已刷新凭据、不重复刷新；并发 401 共享刷新；共享刷新 503 保留凭据、零清理/零提示、恢复后新请求可重新刷新；刷新明确失效清理一次；重放仍 401 不二次刷新；`SESSION_INVALID` 跳过刷新；并发失效只清理提示一次 |
| `SessionStoreIsolationTests`（4） | 退出时刷新/登录迟到成功被丢弃；用户 A 的失效响应不影响用户 B；主动退出无被动失效提示 |
| `KeychainSessionTests`（4） | 独立 service 实际保存/读取/删除；新 `SessionStore` 恢复；刷新后凭据可跨重启恢复；退出后重启保持未登录 |

### 4.3 真实后端人工联调（通过）

前置条件（全部满足）：

- 专用测试手机号与短信链路：已确认（真实验证码收到并登录成功）。
- 开发后端、数据库、Redis：已确认可用。
- 可正常人工输入的环境：已确认（iPhone 16 Pro 模拟器 iOS 18.1，Xcode 27.0）。
- 隔离短 TTL 实例：由本地实例提供（详见 5.2），未改动共享开发部署配置。

| 编号 | 场景 | 预期 | 实际 / 结论 |
| --- | --- | --- | --- |
| 5.1 | 联调前检查：四个 Tab 可点击切换、登录入口可打开、手机号/验证码可输入；记录 Xcode、SDK、设备/runtime | 人工输入正常 | **通过**（2026-09-29 人工）：iPhone 16 Pro 模拟器 iOS 18.1；四个 Tab、登录入口与输入均正常 |
| 5.2 | 隔离实例接入：临时本地构建配置指向隔离地址并使用独立测试 Keychain service，记录接入/回切步骤，生产配置不变 | 环境隔离可回退 | **通过**（2026-09-30）：本地实例 `APP_PORT=8899` + `Auth.AccessExpire=120`（refresh 保持 30 天），临时配置与独立 Keychain service 接入，验收后已回切并删除临时配置 |
| 5.3 | 真实收码登录：错误验证码有明确反馈；正确验证码登录成功并在“我的”显示资料；发送后 60s 倒计时；杀进程重启恢复同一用户 | 全链路成功 | **通过**（人工）：真实验证码登录成功，“我的”显示返回资料；60s 倒计时与错误验证码提示均已验证；杀进程重启恢复同一用户 |
| 5.4 | 真实刷新：等待隔离实例 access token 过期，从原生端发请求；确认实际 refresh-token 调用、原请求重放一次、重启后仍恢复用户；记录请求顺序与计数（不记录令牌正文） | 刷新与重放成功 | **通过**（2026-09-30）：见下方真实刷新证据 |
| 5.5 | 断网启动恢复失败后可联网重试；主动退出后重启保持未登录；恢复临时后端及原生本地配置并清理测试自身数据 | 全部符合预期 | **通过**：实例停止时启动显示恢复失败与“重试”，实例恢复后点重试成功回到同一用户；退出后重启保持未登录且无恢复请求；配置与 Keychain 已清理 |

真实刷新证据（同一台模拟器、同一连接；只记录时间/方法/状态，未记录令牌正文）：

```
12:14:38  POST /api/auth/send-sms       200   真实短信
12:14:53  POST /api/auth/login          200   AccessExpire=120s
12:17:59  GET  /api/user/info           401   过期 access
12:17:59  POST /api/auth/refresh-token  200   仅 1 次刷新
12:17:59  GET  /api/user/info           200   原请求重放成功
12:18:19  GET  /api/user/info           200   再次杀进程重启直接用新 token，无刷新
13:01:37  GET  /api/user/info           401   实例恢复后点“重试”
13:01:37  POST /api/auth/refresh-token  200
13:01:37  GET  /api/user/info           200   重放成功
```

说明：恢复失败场景通过停止本地实例（连接不可达，与断网同属网络故障类别）验证，未改动 Mac 网络。

接入/回切步骤（已验证）：

1. 复制 `etc/chasing_points-api.yaml` 为临时配置，`Auth.AccessExpire` 改为 120，关闭 Geocode / WSTSync / SeasonLifecycle worker，以 `APP_PORT=8899` 启动本地实例，复用 dev MySQL/Redis。
2. 临时把 Debug `apiBaseURL` 指向 `http://127.0.0.1:8899`，`credentialService` 改为 `ChasingPoints.Session.test.short-ttl`，重建安装。
3. 用真实测试号登录，完成 5.3–5.5 验证。
4. 验收后退出登录清理测试 Keychain，恢复 `AppConfiguration`，重建安装 dev 构建，删除临时配置并停止本地实例。

### 4.4 登录页 UI 验证（通过）

人工验证（iPhone 16 Pro 模拟器 iOS 18.1，2026-09-29 至 2026-09-30）：

- 输入与协议：手机号/验证码格式提示正常；不勾协议提交被阻止并提示“请先阅读并同意用户协议和隐私政策”，无登录请求。
- 发送与倒计时：发送成功提示与 60s 重发倒计时正常，倒计时期间不可重复发送。
- 登录：错误验证码显示服务端原因且可重试；正确验证码登录成功并返回“我的”显示资料。
- 恢复：杀进程重启恢复同一用户；实例不可达时显示恢复失败与“重试”，实例恢复后点重试回到同一用户。
- 防重复提交：按钮忙碌态禁用 + 提交前 guard（源码核对），配合倒计时人工验证。
- 一次性失效引导：`SessionStore` 自动化断言每个认证代次只清理/提示一次（`sessionInvalidationCount == 1`、notice 单次），MyPage 提示与登录导航为源码核对。

### 4.5 实现 review 后回归（2026-09-30，通过）

修复两个问题：XCTest 宿主无条件读取真实会话；认证响应中过短的内部错误被误认为可展示文案。新增回归确认错误/过期验证码和频控仍可展示，而 Redis 内部错误在纯文本与 JSON 失败响应中均被隐藏。

本轮使用 Xcode 27.0、iPhone 16 Pro 模拟器 iOS 18.1，执行结果：

- Debug `build-for-testing`：通过；生成的 `.xctestrun` 包含测试宿主隔离标记。
- `test-without-building`（关闭并行测试）：45 个用例，0 个失败。
- Release 模拟器构建（`CODE_SIGNING_ALLOWED=NO`）：通过。
- 最低部署目标保持 iOS 17.0。本轮未重复发送真实短信，人工链路证据沿用 4.3。

可复现命令（在 `iOS/` 执行，构建产物位于仓库外）：

```bash
AUTH_REVIEW_DIR=$(mktemp -d /tmp/chasingpoints-auth-review.XXXXXX)
xcodebuild build-for-testing -project iOS.xcodeproj -scheme iOS \
  -configuration Debug -destination 'platform=iOS Simulator,name=iPhone 16 Pro,OS=18.1' \
  -derivedDataPath "$AUTH_REVIEW_DIR/DerivedData"
xcodebuild test-without-building -xctestrun "$AUTH_REVIEW_DIR"/DerivedData/Build/Products/*.xctestrun \
  -destination 'platform=iOS Simulator,name=iPhone 16 Pro,OS=18.1' \
  -parallel-testing-enabled NO
xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Release \
  -destination 'generic/platform=iOS Simulator' \
  -derivedDataPath "$AUTH_REVIEW_DIR/ReleaseDerivedData" CODE_SIGNING_ALLOWED=NO
```

## 5. 完成情况

所有验收项均已通过（2026-09-29 至 2026-09-30），无未解除阻塞项。

清理记录：

- 本地短 TTL 实例已停止，临时配置 `etc/chasing_points-api.short-ttl.yaml` 已删除。
- App 已恢复 Debug 配置 `https://dev-api.kekemate.cn` + `ChasingPoints.Session.development`，重建后二进制核对无测试地址与测试 service。
- 测试 Keychain service `ChasingPoints.Session.test.short-ttl` 已通过 App 退出登录清空。
- 本地实例日志只含方法/状态码/时间，无 token、验证码或完整手机号。

**链路状态：真实必验项全部通过。**

## 6. 安全与数据边界

- 源码、夹具与本文档不得包含真实 token、验证码、私钥或完整手机号。
- 自动化测试只使用虚构数据与受控响应，Keychain 测试结束后删除自身 service 数据。
- 真实联调仅使用已确认的开发环境与测试号，验证码由测试人员在 App 内输入。
- 真实刷新验证必须使用隔离实例实际签发并过期的 access token，并通过真实 refresh 接口恢复，不得通过改写客户端 JWT 或生产绕过开关伪造。
