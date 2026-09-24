# 微信消息推送 URL 验证设计

## 目标

让微信小程序管理后台在「开发 → 开发管理 → 消息推送配置」中提交配置时，能够验证 `https://<公网域名>/api/wechat/message-push`。

本变更仅覆盖首次 GET 验签。不会接收或处理后续 POST 推送，也不会发送客服消息。

## 已确认范围

- 采用最小验证端点方案。
- 后端只校验微信提交时携带的 `signature`、`timestamp`、`nonce`，验签成功后原样返回 `echostr`。
- 不新增数据库表、业务 API 契约或客户端代码。
- 不实现消息的 AES 解密、事件分发或回包加密；这些属于后续独立变更。

## 接口与数据流

```text
微信后台
  └─ GET /api/wechat/message-push?signature=...&timestamp=...&nonce=...&echostr=...
       └─ 后端按字典序拼接 Token、timestamp、nonce，计算 SHA-1
            ├─ 匹配：HTTP 200，纯文本原样返回 echostr
            └─ 不匹配：HTTP 403
```

该路由是微信协议回调，返回体必须为纯文本，因此在 `backend/chasing_points.go` 中以手工路由注册，不加入 `chasing_points.api`，也不修改 goctl 生成文件。

## 配置

在现有 `WechatMiniProgram` 配置块增加以下环境变量引用：

- `WECHAT_MINI_MESSAGE_PUSH_TOKEN`
- `WECHAT_MINI_MESSAGE_PUSH_ENCODING_AES_KEY`

Token 用于 GET 验签。EncodingAESKey 仅安全地保存在服务端配置中，以便与微信后台的安全模式配置保持一致；本变更不读取、记录或使用它进行消息解密。

真实 Token 与 EncodingAESKey 只写入部署环境的 `.env` 或密钥管理系统，样例文件仅保留占位值。缺少 Token 时，端点返回 HTTP 503，不回显任何配置内容。

## 错误与安全边界

- Token、timestamp、nonce 使用字符串字典序排序、拼接后计算 SHA-1，与 `signature` 做常量时间比较。
- 缺少任一验证参数或签名不匹配时，返回 HTTP 403，且不输出 Token、AES Key 或待比较的签名。
- 成功响应只包含 `echostr` 本身，设置 `text/plain; charset=utf-8`。
- POST 暂不注册；微信验证通过后触发的真实消息推送不属于本次承诺范围。

## 验证

新增 HTTP handler 单元测试，覆盖：

1. 正确签名返回 HTTP 200，响应体与 `echostr` 完全一致。
2. 错误签名返回 HTTP 403，且不回显挑战字符串或机密。
3. 未配置 Token 返回 HTTP 503。
4. 响应不是项目通用 JSON 包装，而是微信要求的纯文本。

部署后，使用 HTTPS 443 的公网域名配置 URL，并在微信后台填入与环境变量相同的 Token 和 EncodingAESKey。选择安全模式与 JSON 后点击“提交”，微信的 GET 验证应通过。

## 非目标

- POST 消息验签、AES-CBC 解密、事件解析和消息回包。
- 客服消息、订阅消息或业务通知发送。
- 修改小程序登录、手机号绑定、UniPush 或数据库结构。
