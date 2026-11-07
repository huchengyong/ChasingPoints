## Context

管理员登录用 `pkg.GenerateToken(-adminID, 同一个 AccessSecret, ...)` 签发令牌，用户令牌用同一个密钥签发，区别只有 ID 的正负。go-zero 的 `jwt: Auth` 只验签名和有效期，不区分管理员和用户。

现状（见 proposal）：全局中间件 `ActiveUserSessionMiddleware` 对负数 ID 直接放行，对正数 ID 只确认用户启用；后台 logic 各自调用 `utils.GetAdminIDFromCtx` 来拒绝非管理员。39 个带 JWT 的后台接口里有 20 个没有这个调用（或调用后忽略错误），所以普通用户令牌能直接调到。

go-zero 的路由器在匹配前对 `r.URL.Path` 做 `path.Clean`（`rest/router/patrouter.go`），所以 `/api//admin/x`、`/api/admin/./x` 都会匹配到 `/api/admin/x`。全局中间件在 JWT 校验之后、handler 之前执行（`engine.bindRoute` 先加认证再加 `Use` 的中间件），此时 `user_id` 已在上下文里。

## Goals / Non-Goals

**Goals:**
- 一个统一的关口，使所有 `/api/admin/` 接口都要求管理员身份，新增接口默认受保护。
- 关闭现有 20 个接口的越权；不改变管理员的任何现有行为。
- 回归测试能在“有人新增后台路由却没被覆盖”时失败。

**Non-Goals:**
- 令牌吊销、改密后旧令牌失效、管理员角色细分、操作审计。
- 清理各 logic 里已有的管理员检查（它们同时用于取得操作人 ID）。
- 改动 `.api` 契约、表结构或管理后台前端。

## Decisions

**1. 用全局中间件按路径兜底，而不是给每个 logic 补检查，也不用 `.api` 的 `middleware:` 声明。**
- 逐个补检查已经证明会漏；再补 20 处也无法阻止下一次遗漏。
- 在 `.api` 的 `@server` 里加 `middleware:` 需要重新生成代码，且是按分组声明的，新增分组忘记声明仍然会漏。
- 仓库已有按路径前缀工作的全局中间件（`OptionalPublicJWTMiddleware`）和做会话校验的全局中间件，这是一致的做法。按路径兜底在运行时默认拒绝，新增接口不需要记得做任何事。

**2. 放行规则：只有三个公开入口走白名单，其余 `/api/admin/` 路径一律要求负数 ID。**
- 白名单是精确匹配规整后的路径：`/api/admin/login`、`/api/admin/init`、`/api/admin/exists`。
- 白名单路径直接放行，不看令牌。这样请求头里带着过期的普通用户令牌访问登录页也不会被拦。
- 非白名单路径：没有身份 → 401；ID ≥ 0 → 403；ID < 0 → 放行。没有身份在 JWT 路由上本来已被 go-zero 拦成 401，这里的 401 是给“忘了加 jwt 的新路由”兜底。
- 备选：只拦截正数 ID、匿名放行。否决——新增一个忘了加 `jwt:` 的后台路由会变成匿名可访问。

**3. 路径规范化与路由器保持一致：用 `path.Clean(r.URL.Path)` 再判断前缀。**
- 前缀判断为 `p == "/api/admin" || strings.HasPrefix(p, "/api/admin/")`，覆盖根路径。
- 备选：直接 `strings.HasPrefix(r.URL.Path, "/api/admin/")`。否决——会被 `/api//admin/...` 绕过。
- 比较区分大小写，与路由器一致（路由器本身区分大小写，`/API/ADMIN` 匹配不到任何路由）。

**4. 拒绝响应：HTTP 403 + `{"code":403,"success":false,"message":"无管理员权限"}`。**
- 管理后台前端对 403 只弹“没有权限访问”，不会登出；对 401 才登出。用户令牌是误用，不应让后台页面登出。
- 响应体字段与后台 logic 的响应信封（`code`、`success`、`message`）保持一致。

**5. 中间件放在 `ActiveUserSessionMiddleware` 之前。**
- 先拒绝普通用户令牌，就不会为这些请求去查一次用户表。
- 被禁用的用户同样会得到 403（身份就是错的），不影响“被禁用用户拒绝访问后台”的要求。
- 靠后的仍然是 `RequestContextMiddleware`、`RequestObservabilityMiddleware`，拒绝会被记录。

**6. 把全局中间件的注册抽成 `registerGlobalMiddlewares(server, c, svcCtx)`，供 `main` 和测试共用。**
- 这样测试走的是真实的注册顺序，而不是测试里自己拼一遍；以后有人在 `main` 里删掉这一行，测试会失败。
- 备选：只测中间件单元。否决——这次的漏洞正是“看起来有检查、实际没接上”这一类，需要真实服务的覆盖。

**7. 回归测试用真实服务：`rest.MustNewServer` + `handler.RegisterHandlers` + `registerGlobalMiddlewares`，监听本机空闲端口。**
- `rest.Server` 没有对外的 `ServeHTTP`，只能监听端口。测试把 `Routes()` 里 `/api/admin` 开头的路由逐一用“普通用户令牌”和“匿名”请求访问：普通用户令牌 → 403，匿名 → 401（公开入口例外）。
- 对管理员令牌只在中间件单元测试里用桩 handler 验证“放行”，不在真实服务里调用，因为真实 handler 会访问数据库。

## Risks / Trade-offs

- [管理员的合法请求被误拦] → 管理员令牌是负数 ID，规则只看 ID 符号；用单元测试覆盖负数放行，并在开发环境用真实管理员账号走一遍后台页面再发生产。
- [前端静态资源或其他系统调用 `/api/admin` 时带普通令牌] → 已核对 UniApp、原生 iOS、官网都不调用 `/api/admin`；只有管理后台调用。
- [中间件路径判断与路由器不一致造成绕过] → 使用同样的 `path.Clean`，并用真实服务测试 `//`、`/./`、`/x/../` 变体。
- [管理员令牌仍然不查库、不可吊销] → 本变更不处理，列入后续事项。
- [白名单维护] → 只有三条，新增必须改代码并过评审；测试会校验白名单路径之外的后台路由都不能匿名访问。

## Migration Plan

1. 后端发布到开发环境；用管理员账号登录后台，走一遍用户、球馆、配置、装备目录等页面，确认没有 403。
2. 用匿名请求验证开发环境上各后台接口返回 401，公开入口仍可访问。
3. 确认后发布生产（`backend/deploy-prod.sh`）。
4. 回滚：回退到上一个后端版本即可，没有数据迁移。
