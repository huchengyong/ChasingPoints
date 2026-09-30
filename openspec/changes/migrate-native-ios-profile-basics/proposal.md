## Why

原生认证链路已完成，当前“我的”仍是用于验证登录态的简化页面。下一步迁移用户资料与基础设置，形成“查看资料 → 修改昵称 → 返回后立即看到结果”的小型业务闭环，为逐步迁移 UniApp 提供可复用的原生页面模式。

## What Changes

- 将“我的”组织为用户资料卡和设置入口，显示真实头像、昵称、脱敏手机号；保留访客登录、恢复重试和会话失效引导。
- 增加昵称编辑：本地校验、保存状态、错误重试及服务端资料回写，复用已完成的认证请求与账号隔离。
- 将已有主题选择和退出登录迁入基础设置页，提供用户协议与隐私政策入口，退出前支持确认与取消。
- 验证资料刷新次数、保存后即时同步、退出/切号时的迟到响应隔离，以及亮暗主题与导航。

第一期手机号只读、头像只展示；头像上传、绑定手机号、战绩、段位、会员、通知和其他 Tab 的业务页面后续迁移。最低支持 iOS 17.0。

## Capabilities

### New Capabilities

- `native-ios-profile-basics`：原生“我的”资料展示、昵称编辑与基础设置的行为和验收要求。

### Modified Capabilities

无。复用 `native-ios-auth-integration`、`stale-auth-session-recovery` 的原生认证约束及 `auth-safe-client-read-coordination` 的身份隔离与避免重复读取原则；不改变已有要求。

## Impact

- 原生端：`ContentView.swift`、`SessionStore.swift`，新增资料编辑/设置视图以及相应原生测试；继续使用 `AppTheme`。
- 后端：复用 `GET /api/user/info` 和 JWT 保护的 `POST /api/user/profile`，昵称保存仅发送 `nickname`；无需新增接口、数据迁移或客户端密钥。
- 验证：本地受控测试、模拟器界面验收和专用开发账号昵称保存联调；保留真实后端与模拟测试结果的区别。
