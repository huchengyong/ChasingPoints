# ADMIN FRONTEND GUIDE

适用于 `admin/`（Vue3 + Vite + TypeScript + Element Plus），同时遵循[仓库指南](../AGENTS.md)。管理后台共用 `backend/`，不复用 UniApp 页面规范或 API 门面。

## 关键入口
| 任务 | 位置 |
| --- | --- |
| 当前路由、页面与鉴权守卫 | [src/router/index.ts](src/router/index.ts)，不在指南中维护路由副本 |
| 布局与菜单 | [src/layout/](src/layout/)、[src/utils/complianceMode.ts](src/utils/complianceMode.ts) |
| 登录态 | [src/store/user.ts](src/store/user.ts) |
| API 与统一响应处理 | [src/api/](src/api/)、[src/utils/request.ts](src/utils/request.ts)、[src/utils/response.ts](src/utils/response.ts) |
| 球馆审核 | [src/views/venues/review.vue](src/views/venues/review.vue)、[src/api/venue.ts](src/api/venue.ts) |
| 构建与依赖版本 | [package.json](package.json)、[vite.config.ts](vite.config.ts) |

## 请求与页面约束
- 页面通过 `src/api/*.ts` 请求，不直接写 axios、拼 token、处理全局 401 或重复弹错误提示。
- token 由 `store/user.ts` 维护在 `admin-token` cookie 中，页面不自行读写 cookie；非公开路由缺 token 时由路由守卫跳登录。
- API 基地址来自 `.env.development` / `.env.production` 的 `VITE_API_BASE_URL`；不要在页面写死环境地址。
- 业务响应的判定与解包复用请求层及 `unwrapBusinessData`，保留 `0`、`false`、空字符串等合法 payload。
- 新 API 返回类型在对应 `src/api/*.ts` 中声明；接口 shape 改动先核对后端 `.api` 和生成结果。
- 新页面同时核对路由与菜单的合规开关，不因为页面文件存在就恢复被隐藏的入口。
- 管理端保持 Element Plus 风格；[DESIGN.md](../DESIGN.md) 的公开用户端规则不直接适用。

## 验证与开发命令
以下命令在 `admin/` 执行，按需选择；依赖通过 `npm install` 安装。

```bash
# 现有测试直接导入 TypeScript；使用支持类型剥离的 Node.js（22.6+）
node --experimental-strip-types --test tests/*.test.mjs
# 包含 vue-tsc 类型检查与 Vite 构建
npm run build
# 本地开发
npm run dev
```

`lint` / `format` scripts 会改写文件，不作为全仓无差别验证步骤；需要格式化时仅处理本次改动范围。构建警告以本次输出为准，不沿用历史“构建通过”结论。
