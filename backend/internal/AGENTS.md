# INTERNAL LAYER GUIDE

## OVERVIEW
`internal/` 采用 go-zero 常见分层，但这个仓库已经扩展出更多手写能力：`handler -> logic -> model` 是主链路，`svc` 做依赖注入，`pkg/ws` 处理实时通信，`utils` 放 JWT 上下文辅助。

## STRUCTURE
```text
internal/
├── config/       # 配置结构体
├── handler/      # HTTP handler；routes.go 为 goctl 生成
├── logic/        # 业务逻辑；一接口一文件，含少量共享 helper
├── model/        # Gorm 模型与查询方法
├── pkg/push/     # UniPush 推送封装
├── pkg/ws/       # Match/User WebSocket Hub 与 handler
├── sms/          # 阿里云短信客户端与验证码管理
├── svc/          # ServiceContext
├── types/        # goctl 生成 DTO
└── utils/        # jwt_parser 等辅助工具
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| handler 注册关系 | `handler/routes.go` | 生成文件，只读 |
| 单接口逻辑 | `logic/*_logic.go` | 命名与 handler 对齐 |
| 共享业务辅助 | `logic/common.go`, `logic/tournament_helper.go` | 避免在多个 logic 重复复制 |
| 数据模型/查询 | `model/*.go` | Gorm 模型和查询入口 |
| 依赖注入 | `svc/service_context.go` | 所有 model、Redis、短信、Push 初始化 |
| 实时通信 | `pkg/ws/*.go` | 对局同步和用户通知 |
| 用户态解析 | `utils/jwt_parser.go` | 统一解析 ctx 中的 `user_id` |

## SERVICE CONTEXT FACTS
- `ServiceContext` 当前注入了 MySQL、Redis、短信客户端/验证码管理器、UniPush 服务。
- 已注册的 model 包括用户、OAuth、对局、段位、成就、好友、关注、动态、通知、挑战、赛事、赛季、球房等领域。
- 新业务如果需要共享依赖，优先扩展 `ServiceContext`，不要在 logic 中零散初始化。

## CONVENTIONS
- handler 负责解析请求、调用 logic、统一输出；复杂业务不要留在 handler。
- logic 负责业务流程编排、权限校验、领域规则，不直接操作底层 DB。
- model 负责 Gorm 查询、分页、聚合、事务辅助；供 logic 复用。
- DTO 来源于 `.api`，修改请求/响应结构要先改 `.api` 再生成，不直接改 `types`。
- 获取当前登录用户 ID 时统一用 `utils.GetUserIDFromCtx(l.ctx)`。
- `pkg/ws` 下的消息类型和连接参数要与前端 `utils/websocket.js` 保持一致。

## ANTI-PATTERNS
- 禁止在 `handler` 中塞入复杂业务流程。
- 禁止在 `logic` 中直接 new DB/Redis/SMS/Push 客户端。
- 禁止直接修改 `types` 或 `routes.go` 这类生成文件。
- 禁止把跨多个 logic 复用的规则散落复制，优先收敛到 helper 或 model。
