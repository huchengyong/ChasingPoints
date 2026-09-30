# UNIAPP API LAYER GUIDE

本目录只维护业务 HTTP 门面，遵循[UniApp 指南](../AGENTS.md)。接口路径、字段与业务域以 [backend/chasing_points.api](../../backend/chasing_points.api) 为准，不在此维护模块全集。

## 封装边界
- 复用 [utils/request.js](../utils/request.js) 的 `get/post/put/del`，不重复拼接 token 或调用 `uni.request`；有当前 token 时请求层统一附加 Bearer Header。
- API 方法只组织路径、参数和请求选项，不写 toast、弹窗、跳转或页面状态管理；需要 `silent` 等选项时透传给请求层。
- 优先扩展对应领域文件；公共接口也放在已有领域文件中，不为 `/api/public` 另造一层。
- 方法名沿用 `getXxx/createXxx/updateXxx/deleteXxx/markXxx`。多处复用的参数整形才提炼为 helper。

## 响应与认证语义
- 响应放行与解包复用 [request-response.js](../utils/request-response.js) 的 `shouldResolveBusinessResponse` / `unwrapBusinessResponse`，不要在各门面再维护成功判定公式。
- `accepted` 为布尔值的协议响应会交给业务层处理，包括 `accepted: false`；Promise resolve 不等于写入成功，不能丢掉冲突、revision 或 snapshot 信息。
- 普通 401 由请求层在当前认证代次内协调刷新或复用已刷新 token，最多重放一次；不是遇到 401 就直接清登录态。
- HTTP 401 + `SESSION_INVALID` 跳过刷新，走统一失效处理。旧身份响应不得清除新会话或触发新身份的重新登录导航。
- 重放后的网络/服务/业务错误保持其原始类别，不再解释为 refresh 失败。具体分支以 [request.js](../utils/request.js) 和 [request-errors.js](../utils/request-errors.js) 为准，不另写认证兜底。
- `silent: true` 仅抑制请求层的网络失败反馈，不能抑制已确认会话失效的全局提示与导航；调用方使用现有 handled/superseded helper 避免重复提示旧请求错误。

## 验证
改动请求层或认证语义时，在 `uniapp/` 至少运行以下相关回归，再按影响范围补领域测试：

```bash
node --test tests/request.test.mjs tests/auth-refresh.test.mjs tests/auth-scoped-read-coordination.test.mjs
```
