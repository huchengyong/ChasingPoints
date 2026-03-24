# CHASING POINTS REPOSITORY GUIDE

## OVERVIEW
这是一个多端台球项目仓库，当前由 3 个主要子系统组成：
- `app/`：UniApp Vue3 移动端，面向普通用户，覆盖登录、对局、动态、我的、赛事、球房、规则、赛季、通知等链路。
- `backend/`：go-zero REST API 服务，负责业务接口、WebSocket、Redis 短信验证码、UniPush、球房地理编码任务等。
- `admin/`：Vue 3 + Vite + TypeScript + Element Plus 管理后台，当前覆盖管理员登录、首页统计、用户管理、对局管理、赛事情报、球馆审核。

## REPOSITORY MAP
```text
.
├── AGENTS.md                  # 仓库级知识库入口
├── README.md                  # 当前仅保留项目名
├── docs/plans/                # 设计/方案文档
├── app/                       # UniApp 前端
│   ├── AGENTS.md
│   ├── GEMINI.md
│   ├── api/
│   ├── components/
│   ├── pages/
│   ├── store/
│   ├── subPages/
│   ├── tests/
│   └── utils/
├── backend/                   # go-zero 后端
│   ├── AGENTS.md
│   ├── chasing_points.api
│   ├── internal/
│   ├── migrations/
│   └── goose.sh
└── admin/                     # Vue3 管理后台
    ├── AGENTS.md
    ├── src/
    └── package.json
```

## ARCHITECTURE SNAPSHOT
- API 契约单一真源是 [backend/chasing_points.api](/Users/wisesearch/Projects/ChasingPoints/backend/chasing_points.api)。
- 用户端页面只能调用 `app/api/*.js`，不要在页面中直接写 `uni.request`。
- 管理后台页面通过 `admin/src/api/*.ts` 调后端，统一走 [admin/src/utils/request.ts](/Users/wisesearch/Projects/ChasingPoints/admin/src/utils/request.ts)。
- 后端主链路是 `handler -> logic -> model`，共享依赖统一从 `internal/svc/ServiceContext` 注入。
- 实时能力走 2 条 WebSocket 路由：`/api/match/ws`、`/api/user/ws`。
- 管理端不是独立后端，仍复用同一个 go-zero 服务中的 admin 路由。

## CROSS-PROJECT RULES
- 用简体中文沟通。
- 做功能改动时先确认自己所在子系统，再读取对应目录下的 `AGENTS.md`。
- 不要把历史文档、旧分支记忆、旧模块列表当作当前事实；先以仓库实际文件为准。
- 修改后端 `.api` 文件后，下一步必须立刻运行 goctl 生成代码，不要手改生成文件。
- 修改数据库表结构时，迁移、Gorm 模型、前后端字段命名和接口响应要一起核对。
- 新增接口时，要同时考虑 `app/api` 或 `admin/src/api` 是否需要补对应门面。

## CURRENT TECH FACTS
- `app/` 当前是 JavaScript 项目，没有统一 npm scripts；现有测试通过 `node --test tests/*.test.mjs` 执行。
- `admin/` 使用 Vite，接口基地址来自 `.env.development` / `.env.production` 的 `VITE_API_BASE_URL`。
- `app/` 当前通过 `utils/runtime-config.js` 按 `NODE_ENV` 解析 HTTP/WS 基地址；开发环境默认走 tunnel，生产环境默认走正式域名。
- `backend/` 使用 go 1.25、go-zero、Gorm、Redis、Aliyun SMS、UniPush。
- `backend/internal/model` 中有些模型在构造函数里 `AutoMigrate`，但 `event_news` 明确由迁移管理，不会自动建表。

## REVIEW HOTSPOTS
- 移动端请求层：[app/utils/request.js](/Users/wisesearch/Projects/ChasingPoints/app/utils/request.js)
- 移动端 WebSocket：[app/utils/websocket.js](/Users/wisesearch/Projects/ChasingPoints/app/utils/websocket.js)
- 移动端全局状态与主题：[app/store/](/Users/wisesearch/Projects/ChasingPoints/app/store) 与 [app/App.vue](/Users/wisesearch/Projects/ChasingPoints/app/App.vue)
- 后端依赖注入：[backend/internal/svc/service_context.go](/Users/wisesearch/Projects/ChasingPoints/backend/internal/svc/service_context.go)
- 后端实时链路：[backend/internal/pkg/ws/](/Users/wisesearch/Projects/ChasingPoints/backend/internal/pkg/ws)
- 后端迁移与模型一致性：[backend/migrations/](/Users/wisesearch/Projects/ChasingPoints/backend/migrations) 与 [backend/internal/model/](/Users/wisesearch/Projects/ChasingPoints/backend/internal/model)
- 管理后台鉴权与 API：[admin/src/router/index.ts](/Users/wisesearch/Projects/ChasingPoints/admin/src/router/index.ts)、[admin/src/utils/request.ts](/Users/wisesearch/Projects/ChasingPoints/admin/src/utils/request.ts)

## COMMANDS
```bash
# 移动端逻辑测试
cd app
node --test tests/*.test.mjs

# 管理后台构建
cd admin
npm run build

# 后端服务启动
cd backend
go run chasing_points.go -f etc/chasing_points-api.yaml

# 后端 API 代码生成
cd backend
goctl api go --api chasing_points.api --dir . --style go_zero --home ~/.goctl/default

# 后端测试
cd backend
go test ./...

# 数据库迁移
cd backend
./goose.sh status
./goose.sh up
```

## KNOWN PITFALLS
- `app` 没有根级 `npm test` script，默认验证命令是 `node --test tests/*.test.mjs`。
- `backend` 的 `event_news` 相关测试依赖显式建表；如果新增或调整该领域测试，不能指望模型构造函数自动建 schema。
- `app/AGENTS.md` 与 `app/GEMINI.md` 需要保持同步；仓库里当前没有 `IFLOW.md`。
