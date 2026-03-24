# ADMIN FRONTEND GUIDE

## OVERVIEW
`admin/` 是当前项目的管理后台，技术栈为 Vue 3 + Vite + TypeScript + Element Plus + Pinia + Vue Router。它与 `app/` 共享同一个 `backend/` 服务，但面向管理员场景，当前模块包括登录、首页统计、用户管理、对局管理、赛事情报、球馆审核。

## STRUCTURE
```text
admin/
├── public/
├── src/
│   ├── api/            # auth / dashboard / user / match / venue / event-news
│   ├── layout/         # 主布局
│   ├── router/         # 路由表和鉴权守卫
│   ├── store/          # 当前主要是 user store
│   ├── styles/         # 全局样式
│   ├── utils/          # axios request 封装
│   └── views/          # login / dashboard / users / matches / event-news / venues / error
├── types/
├── .env.development
├── .env.production
├── vite.config.ts
└── package.json
```

## CURRENT ROUTES
- `/login`：管理员登录页，同时承接首次初始化管理员入口。
- `/dashboard`：首页统计。
- `/users`：用户管理。
- `/matches`：对局管理。
- `/event-news`：赛事情报管理。
- `/venues`：球馆审核。

## REQUEST AND AUTH FACTS
- 统一请求入口是 [admin/src/utils/request.ts](/Users/wisesearch/Projects/ChasingPoints/admin/src/utils/request.ts)。
- API 基地址来自 `.env.development` / `.env.production` 的 `VITE_API_BASE_URL`。
- token 保存在 `admin-token` cookie 中，由 [admin/src/store/user.ts](/Users/wisesearch/Projects/ChasingPoints/admin/src/store/user.ts) 维护。
- 路由守卫在 [admin/src/router/index.ts](/Users/wisesearch/Projects/ChasingPoints/admin/src/router/index.ts)，规则是非公开路由缺 token 就跳 `/login`。
- 页面层通过 `src/api/*.ts` 调接口，不要在页面里直接写 axios。

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| 登录与初始化管理员 | `src/views/login/index.vue`, `src/api/auth.ts` | 依赖后端 `ADMIN_SETUP_TOKEN` |
| 首页统计 | `src/views/dashboard/index.vue`, `src/api/dashboard.ts` | 对应 admin dashboard 接口 |
| 用户管理 | `src/views/users/index.vue`, `src/api/user.ts` | 关注分页与筛选参数 |
| 对局管理 | `src/views/matches/index.vue`, `src/api/match.ts` | 管理员只读/审查视角 |
| 赛事情报 | `src/views/event-news/index.vue`, `src/api/event-news.ts` | 与 backend `logic/admin` 强绑定 |
| 球馆审核 | `src/views/venues/index.vue`, `src/api/venue.ts` | 与用户端球房提交链路联动 |

## CONVENTIONS
- 页面不要绕过 `src/api/*.ts` 直接请求后端。
- 后端业务成功判定沿用 `code === 0 || success === true`。
- 登录态变更统一通过 `store/user.ts`，不要在页面里手动读写 `admin-token` cookie。
- 管理后台使用 TypeScript，新增 API 类型时优先在对应 `src/api/*.ts` 中声明返回结构。
- 如果接口返回 shape 改动，先确认 `backend/chasing_points.api` 与 goctl 生成结果，再改前端类型。

## COMMANDS
```bash
# 安装依赖
npm install

# 本地开发
npm run dev

# 构建
npm run build

# 格式化
npm run format
```

## KNOWN FACTS
- 构建目前可通过，但会有 Sass legacy JS API 的弃用警告。
- 生产构建存在大体积 chunk 警告，后续如果继续扩展模块，需要关注拆包策略。
- 当前仓库里没有单独的 `admin/README` 以外知识库，所以这份 `AGENTS.md` 是后台开发的主要入口。

## ANTI-PATTERNS
- 不要把用户端 `app/` 的页面规范直接照搬到 `admin/`；这是 Vite Web 项目，不是 UniApp。
- 不要在页面里直接拼接 token、处理 401 或重复弹错误提示，统一交给 `src/utils/request.ts`。
- 不要把管理后台的接口误写到 `app/api/`；`admin` 与 `app` 有独立 API 门面层。
