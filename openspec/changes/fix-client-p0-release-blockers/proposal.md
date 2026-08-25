## Why

当前 App / 小程序主链路虽已可运行，但审查发现第三方登录身份可伪造、实时通道泄露凭证、发布材料与权限边界不安全、账号数据生命周期缺失，以及部分跨端链路和失败反馈不可信等 P0 问题。上述问题会直接影响账号安全、隐私合规、长连接可用性和客户端上线验收，必须在正式发布前作为统一门禁完成修复。

## What Changes

- 将通用第三方 OAuth 登录改为服务端验证授权凭据并解析唯一身份，停止信任客户端直接提交的 `provider/open_id` 组合。
- 收紧生产 HTTP 与 WebSocket 边界：使用明确 Origin 白名单、禁止 WebSocket URL 查询参数携带 access token，并区分鉴权失败与可恢复网络断线。
- 将 WebSocket 固定 5 次重连改为生命周期感知的退避恢复；网络恢复、前台恢复或用户主动重试后可继续恢复实时能力，失效会话不得无限重试。
- 清理仓库中的发布签名材料并完成轮换，建立敏感材料外置门禁；按实际能力缩减 Manifest 高敏权限，并对发布包做权限检查。
- 将 Android/iOS 图标与 Harmony 图标、启动页资源纳入版本控制，确保任意干净工作区均可复现本地 HBuilderX 构建；签名材料继续保持本地化。
- 移除二维码生成对 `api.qrserver.com` 的运行时依赖，改为客户端离线生成或受控的一方能力，确保弱网与隐私场景下可用。
- 新增用户可操作的个人数据导出与账号注销链路，明确确认、会话失效、数据删除或匿名化以及结果反馈规则。
- 补齐小程序扫码与关键 PK 邀约/进入/结果链路，使 App 与小程序均能完成受支持的核心流程；不支持的能力必须明确降级，不能展示无效入口。
- 统一加载失败、空数据和权限/会话错误的展示语义，禁止把请求失败伪装成“暂无数据”。
- 为上述能力增加后端、客户端和发布配置的自动化检查，并将安全回归、真机跨端验证和密钥轮换证明纳入上线门禁。
- **BREAKING**：旧版通用 OAuth 请求若只提交未经验证的 `open_id` 将被拒绝；客户端必须改为提交平台授权凭据。
- **BREAKING**：旧版通过 WebSocket URL 查询参数传 token 的连接将不再被生产服务接受，App 与后端需要协同发布。

## Capabilities

### New Capabilities

- `verified-third-party-login`: 定义第三方登录凭据的服务端验证、身份绑定、防重放与失败行为。
- `secure-realtime-channel`: 定义 WebSocket Origin、凭证传输、鉴权失败、退避重连和生命周期恢复边界。
- `production-security-baseline`: 定义生产 CORS、发布签名材料、最小权限、受控二维码生成和发布门禁。
- `account-data-lifecycle`: 定义个人数据导出、账号注销、数据删除或匿名化以及会话收口。
- `cross-platform-critical-flow`: 定义 App / 小程序扫码与 PK 核心链路的能力探测、闭环和显式降级。
- `truthful-client-state-feedback`: 定义加载、空数据、失败、权限和会话状态的可区分反馈语义。

### Modified Capabilities

- 无。

## Impact

- 客户端：`app/pages/`、`app/subPages/`、`app/api/`、`app/store/`、`app/utils/`、`app/manifest.json`、平台发布配置与 `app/tests/`。
- 后端：`backend/chasing_points.api`、认证与用户 logic、WebSocket handler / 路由、CORS 中间件、配置、model / migration 及对应测试；若修改 `.api`，必须立即通过 goctl 重新生成代码。
- 发布与运维：生产允许域名、第三方 OAuth 凭据、WebSocket 鉴权协商、签名材料托管与轮换、权限清单和上线检查流程。
- 兼容性：OAuth 与 WebSocket 客户端需要与服务端协同升级；灰度期间必须避免旧客户端静默降级到不安全协议。
- 范围明确排除 `website/` 与 `admin/` 的页面和产品功能；仅允许为用户端安全边界所必需的共享后端配置变化。
