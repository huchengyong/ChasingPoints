# 微信小程序一键登录与按需手机号设计

## 目标

将小程序的新用户登录从“欢迎页 → 手机号 → 短信验证码”改为“价值说明 → 微信一键进入”。用户成功完成微信身份登录后立即进入首页；手机号只在用户主动绑定或后续业务明确需要时获取。

首要成功指标是首次登录完成率。实现不移除现有 App/Harmony 的短信及华为登录能力。

## 现状与约束

- `app/pages/welcome/index.vue` 现在是首次启动入口，主操作会进入 `app/pages/login/login.vue` 的手机号短信表单。
- 后端已有 `/api/auth/login`、`/api/auth/bind-phone` 和通用 `/api/auth/login-by-oauth`；当前通用 OAuth 接口直接接受客户端提交的 `open_id`，不能作为小程序身份校验入口。
- `user_oauth` 已对 `(provider, open_id)` 建立唯一约束，可安全保存小程序 OpenID 关联。
- 小程序 AppID 已在前端 manifest 中配置；服务端必须通过环境变量单独持有 AppSecret，不能从前端、接口响应或日志暴露它。
- 现有手机号绑定在号码归属其他用户时会迁移 OAuth 关联并删除临时 OAuth 用户。小程序绑定必须保持该账号合并语义，并更新到保留账号的新令牌。

## 非目标

- 不获取或展示微信头像、昵称；新用户使用固定的“微信用户”默认昵称，避免增加无关授权。
- 不废弃短信登录、短信绑定或 App/Harmony 登录页面。
- 不新增埋点平台、营销弹窗、强制手机号授权或头像昵称资料完善。
- 不把微信支付 AppID、App 私有凭证或小程序会话密钥复用为登录凭证。

## 已确认的页面体验

### 首次入口

小程序首次进入保留 `welcome` 页面作为唯一的登录决策页，页面使用真实的 `app/static/logo.png`，维持现有暖白/追分金视觉基调：

1. 显示“每一杆，都值得被记录”及一句产品价值说明。
2. 唯一主操作是绿色“微信一键进入”；微信绿色只用于此授权动作。
3. “手机号登录”降为文字级兜底入口，进入现有短信表单。
4. 协议文案紧贴主操作；未确认协议时点击主操作显示既有协议确认弹层，确认后继续原先的点击动作。
5. 登录完成后设置现有 `WELCOME_PAGE_VIEWED_KEY` 并进入原本的回跳或首页路径。

`pages/login/login.vue` 也提供等价的微信一键入口，确保从受限业务页、个人页等直接跳转登录时不退回到短信主流程。非 `MP-WEIXIN` 平台保留当前手机号/华为 UI 与逻辑。

### 按需获取手机号

微信登录成功不会主动弹出手机号授权。`bindPhone` 组件在 `MP-WEIXIN` 上改用原生 `button open-type="getPhoneNumber"`：

1. 用户在设置或既有业务引导中主动打开绑定手机号面板。
2. 面板简要说明用途后发起一次微信手机号授权。
3. 用户拒绝授权时留在当前登录态，不显示错误，也不降级为强制短信。
4. 授权成功后更新当前用户信息；若号码已属其他账号，后端合并关联并返回保留账号的新令牌，前端刷新 store。

其他平台继续使用现有手机号 + 短信验证码面板。

## 服务端身份与账号流程

### 小程序一键登录

前端在主按钮的用户手势中调用 `uni.login()`，只把短期 `code` 提交给后端。新增 `POST /api/auth/wechat-mini-login`：

```json
{ "code": "wx-login-code" }
```

后端用配置中的小程序 AppID/AppSecret 调用微信 `jscode2session`，只在服务端读取 `openid` 和可选 `unionid`。后端不得向客户端返回 `session_key`，也不得信任客户端传来的 OpenID。

用 `provider = "weixin_mini_program"` 和 OpenID 查找 `user_oauth`：

- 已有关联：加载已有用户、签发当前用户令牌。
- 没有关联：创建一个无手机号用户和 OAuth 关联，默认昵称为“微信用户”，沿用现有新用户会员奖励规则，再签发令牌。

响应沿用当前 OAuth 登录的令牌和 `user_info` 结构，并返回 `need_bind_phone`，但登录页不据此自动弹出绑定面板。

### 小程序手机号绑定

新增受 JWT 保护的 `POST /api/auth/wechat-mini-bind-phone`：

```json
{ "code": "get-phone-number-code" }
```

后端用小程序 access token 调用微信 `getuserphonenumber`，验证返回的中国大陆手机号格式，再复用现有绑定/合并规则：

- 当前用户没有该手机号：更新手机号。
- 手机号已属于另一用户：把当前小程序 OAuth 关联迁移给手机号用户，清理临时用户，向前端返回保留用户的全新令牌和资料。

响应包含 `success`、`message`、`merged_account`，以及合并时必需的新 `access_token`、`refresh_token`、`expires_in` 和 `user_info`。普通绑定无需换令牌，但允许前端以响应资料刷新用户 store。

### 安全与失败处理

- 服务端小程序客户端封装为可注入依赖，使用超时 HTTP 客户端；测试通过假客户端覆盖成功、微信错误和网络失败。
- AppID/AppSecret 使用 `WECHAT_MINI_PROGRAM_APP_ID` 和 `WECHAT_MINI_PROGRAM_APP_SECRET` 环境变量注入 `config.Config`；样例/生产 YAML 仅引用变量，不写真实值。
- access token 仅在服务端按过期时间缓存；微信响应中的错误码转换为用户可读、不可泄露凭证的错误信息。
- `uni.login`、后端登录、手机号授权失败均恢复按钮状态并保留“手机号登录”兜底；同一点击期间禁止重复提交。
- 通用 `/login-by-oauth` 继续服务现有华为流程，但必须拒绝 `weixin_mini_program`，保证小程序 OpenID 只能来自服务端交换。

## 接口与文件边界

| 范围 | 变更 |
| --- | --- |
| `backend/chasing_points.api` | 新增小程序登录、手机号绑定请求/响应与路由契约。 |
| `backend/internal/config`、`svc`、`pkg/wechatmini` | 小程序机密配置、可注入微信客户端、access token 缓存。 |
| `backend/internal/logic/auth` | 基于服务端换取身份的登录、手机号绑定/合并、通用 OAuth provider 限制。 |
| `backend/internal/model` | 仅在需要事务性复用时补最小 model 方法；不改变现有表结构。 |
| `app/api/auth.js` | 增加小程序登录与手机号绑定门面。 |
| `app/pages/welcome/index.vue` | 小程序首屏使用已确认的真实 Logo/UI 和微信主操作；保留其他平台现有流程。 |
| `app/pages/login/login.vue` | 小程序直接登录入口改为微信优先、短信兜底；非小程序不变。 |
| `app/components/bindPhone.vue` | 小程序下以 `getPhoneNumber` 绑定，其他平台继续短信。 |
| `app/utils`、`app/tests` | 将按钮状态、登录响应和手机号绑定结果中可测的规则抽为纯函数并覆盖。 |

## 验收标准

1. 小程序新用户只需同意协议并点击一次微信主操作，即可创建账号、拿到令牌并进入首页。
2. 客户端永不发送 OpenID、AppSecret 或 session_key；后端拒绝用通用 OAuth 入口伪造小程序身份。
3. 已绑定小程序账号再次登录复用同一用户；新建账号只创建一个 OAuth 关联。
4. 用户可在后续主动执行微信手机号授权；拒绝后仍可继续使用不需要手机号的能力。
5. 手机号与既有账号冲突时，OAuth 关联迁移到既有账号，前端会话切换到该账号。
6. 小程序页面使用真实项目 Logo，不使用原型中的 CSS/文字占位 Logo；App/Harmony 现有登录行为不回归。
7. 后端认证逻辑与 App 纯逻辑测试覆盖成功、拒绝、网络失败、重复提交、账号合并和 provider 限制；全量既有测试保持通过。

## 上线前所需配置

实现可在没有机密的本地假客户端/测试环境完成。联调与生产启用前需要用户提供：

- 微信小程序 AppSecret（对应 manifest 中的小程序 AppID）；
- 微信公众平台将开发环境与生产 API 域名配置为合法 request 域名；
- 如小程序未开通 `getPhoneNumber` 能力，需先在微信公众平台完成开通与权限配置。
