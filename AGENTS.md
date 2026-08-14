# CHASING POINTS REPOSITORY GUIDE

## Simplicity First
**Minimum code that solves the problem. Nothing speculative.**
- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.
Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## Surgical Changes
**Touch only what you must. Clean up only your own mess.**
When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.
When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.
The test: Every changed line should trace directly to the user's request.

## OVERVIEW
这是一个多端台球项目仓库，当前由 4 个主要子系统组成：
- `app/`：UniApp Vue3 移动端，面向普通用户，覆盖登录、对局、动态、我的、赛事、球房、规则、赛季、通知等链路。
- `backend/`：go-zero REST API 服务，负责业务接口、WebSocket、Redis 短信验证码、UniPush、球房地理编码任务等。
- `admin/`：Vue 3 + Vite + TypeScript + Element Plus 管理后台，当前覆盖管理员登录、首页统计、用户管理、对局管理、赛事情报、球馆审核。
- `website/`：Nuxt 3 官网子项目，负责品牌首页、下载页、协议页、联系页和基础 SEO。

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
├── admin/                     # Vue3 管理后台
│   ├── AGENTS.md
│   ├── src/
│   └── package.json
└── website/                   # Nuxt3 官网
    ├── AGENTS.md
    ├── pages/
    ├── data/
    └── package.json
```

## ARCHITECTURE SNAPSHOT
- API 契约单一真源是 [backend/chasing_points.api](/Users/wisesearch/Projects/ChasingPoints/backend/chasing_points.api)。
- 用户端页面只能调用 `app/api/*.js`，不要在页面中直接写 `uni.request`。
- 管理后台页面通过 `admin/src/api/*.ts` 调后端，统一走 [admin/src/utils/request.ts](/Users/wisesearch/Projects/ChasingPoints/admin/src/utils/request.ts)。
- 官网公开页面由 `website/pages/*.vue` 暴露，页面文案与下载配置集中在 `website/data/*.ts`。
- 后端主链路是 `handler -> logic -> model`，共享依赖统一从 `internal/svc/ServiceContext` 注入。
- 实时能力走 2 条 WebSocket 路由：`/api/match/ws`、`/api/user/ws`。
- 管理端不是独立后端，仍复用同一个 go-zero 服务中的 admin 路由。

## CROSS-PROJECT RULES
- 用简体中文沟通。
- 做功能改动时先确认自己所在子系统，再读取对应目录下的 `AGENTS.md`。
- 不要把历史文档、旧分支记忆、旧模块列表当作当前事实；先以仓库实际文件为准。
- 修改后端 `.api` 文件后，下一步必须立刻运行 goctl 生成代码，不要手改生成文件。
- 当前后端接口入口 logic 已按 `backend/internal/logic/<group>/` 分组；根目录 `backend/internal/logic/*.go` 只保留共享 helper / service / protocol / payload 等公共层。重新跑 goctl 后如果出现新的 `todo` 空壳文件，只有在同步补齐真实逻辑与 handler 引用后才允许提交。
- 修改数据库表结构时，迁移、Gorm 模型、前后端字段命名和接口响应要一起核对。
- 新增接口时，要同时考虑 `app/api` 或 `admin/src/api` 是否需要补对应门面。
- 高频读链路必须保持服务端查询有界、GET 纯读和性能可观测；用户级客户端缓存与请求去重必须认证隔离，不得用无界扫描、N+1 或通用 GET 缓存换取短期便利。

## CURRENT TECH FACTS
- `app/` 当前是 JavaScript 项目，没有统一 npm scripts；现有测试通过 `node --test tests/*.test.mjs` 执行。
- `admin/` 使用 Vite，接口基地址来自 `.env.development` / `.env.production` 的 `VITE_API_BASE_URL`。
- `website/` 使用 Nuxt 3，正式域名通过 `NUXT_PUBLIC_SITE_URL` 注入。
- `app/` 当前通过 `utils/runtime-config.js` 按 `NODE_ENV` 解析 HTTP/WS 基地址；开发环境默认走 tunnel，生产环境默认走正式域名。
- `backend/` 使用 go 1.25、go-zero、Gorm、Redis、Aliyun SMS、UniPush。
- `backend/` 数据库结构现在统一由 `backend/migrations/*.sql` 管理；测试如果需要 schema，走 `backend/internal/testsupport` 显式准备。
- `backend/` 的接口入口逻辑目录现以 `backend/internal/logic/<group>/` 为准；根目录 `backend/internal/logic/*.go` 是公共层，不再放 handler 一一对应的接口 logic。

## REVIEW HOTSPOTS
- 移动端请求层：[app/utils/request.js](/Users/wisesearch/Projects/ChasingPoints/app/utils/request.js)
- 移动端 WebSocket：[app/utils/websocket.js](/Users/wisesearch/Projects/ChasingPoints/app/utils/websocket.js)
- 移动端全局状态与主题：[app/store/](/Users/wisesearch/Projects/ChasingPoints/app/store) 与 [app/App.vue](/Users/wisesearch/Projects/ChasingPoints/app/App.vue)
- 后端依赖注入：[backend/internal/svc/service_context.go](/Users/wisesearch/Projects/ChasingPoints/backend/internal/svc/service_context.go)
- 后端实时链路：[backend/internal/pkg/ws/](/Users/wisesearch/Projects/ChasingPoints/backend/internal/pkg/ws)
- 后端迁移与模型一致性：[backend/migrations/](/Users/wisesearch/Projects/ChasingPoints/backend/migrations) 与 [backend/internal/model/](/Users/wisesearch/Projects/ChasingPoints/backend/internal/model)
- 管理后台鉴权与 API：[admin/src/router/index.ts](/Users/wisesearch/Projects/ChasingPoints/admin/src/router/index.ts)、[admin/src/utils/request.ts](/Users/wisesearch/Projects/ChasingPoints/admin/src/utils/request.ts)
- 官网页面与 SEO：[website/pages/](/Users/wisesearch/Projects/ChasingPoints/website/pages) 与 [website/composables/usePageSeo.ts](/Users/wisesearch/Projects/ChasingPoints/website/composables/usePageSeo.ts)

## COMMANDS
```bash
# 移动端逻辑测试
cd app
node --test tests/*.test.mjs

# 管理后台构建
cd admin
npm run build

# 官网开发
cd website
npm run dev

# 官网构建与测试
cd website
npm run test
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
- `backend` 的赛讯领域测试依赖显式建表；当前有效表是 `event_news_events`、`tournaments` 和 `tournament_matches`，不要再依赖已删除的阶段表或废弃的单表 `event_news`。
- `backend` 的 MySQL 迁移要注意版本兼容：`CREATE TABLE IF NOT EXISTS` 可以用，但不要默认写 `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` 或 `DROP COLUMN IF EXISTS`，部分环境会直接报 1064。给已有表补字段时，先查 `information_schema.COLUMNS` 再决定是否执行 `ALTER TABLE`。
- `app/AGENTS.md` 与 `app/GEMINI.md` 需要保持同步；仓库里当前没有 `IFLOW.md`。
- `website` 当前下载链接、联系信息和 sitemap 仍是占位值，上线前必须替换为正式内容。
