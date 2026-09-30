# UNIAPP FRONTEND GUIDE

适用于 `uniapp/` 跨平台用户端（Vue3 + JavaScript），同时遵循[仓库指南](../AGENTS.md)；不适用于原生 `iOS/`。规则以 [AGENTS.md](AGENTS.md) 为维护源，[GEMINI.md](GEMINI.md) 保持同内容镜像。

## 关键入口
| 任务 | 位置 |
| --- | --- |
| 主包、Tab、分包与导航注册 | [pages.json](pages.json)，以实际注册为准，不按文件目录推断页面是否开放 |
| 应用生命周期、Push、前台提醒 | [App.vue](App.vue)、[main.js](main.js) |
| 用户状态与持久化 | [store/user.js](store/user.js)、[store/index.js](store/index.js) |
| HTTP 门面与认证处理 | [api/](api/)、[api/AGENTS.md](api/AGENTS.md)、[utils/request.js](utils/request.js) |
| HTTP/WS 环境地址 | [utils/runtime-config.js](utils/runtime-config.js) |
| 实时对局 | [utils/websocket.js](utils/websocket.js)、[utils/match-action.js](utils/match-action.js) |
| 主题 | [store/theme.js](store/theme.js)、[theme.json](theme.json)、[utils/theme-application.js](utils/theme-application.js) |
| 页面局部约束 | [subPages/AGENTS.md](subPages/AGENTS.md) |

## 页面与请求边界
- 页面、组件和 store 的业务 HTTP 请求统一经过 `api/*.js`，不直接调用 `uni.request` 或 `utils/request.js`。涉及请求/认证或新增 API 门面时，先读 [API 层指南](api/AGENTS.md)。
- `App.vue` 现有 Push 上报、会话失效转接属于应用基础设施例外，不把这一例外扩大到业务页面。
- 页面优先使用 `script setup` + SCSS，样式优先沿用同名 `.scss`；`App.vue` 保留承接 UniApp 生命周期的 Options API。共享样式放在既有主题/全局样式入口。
- 业务规则需要复用或独立测试时优先放 `utils/*.js` 纯函数，不为简单页面操作额外造通用表单/页面框架。
- 时间格式复用 [utils/format.js](utils/format.js)，图标优先用 `uni-icons`。Push、主题、前台提醒不要在各页面重复实现。

## 读取与状态隔离
- 首屏优先复用聚合接口，避免 `onLoad`、`onMounted`、`onShow` 与子组件重复读取同一资源。
- 禁止在通用请求层增加 GET 缓存；single-flight、TTL/SWR、loaded/dirty 和失效逻辑属于具体领域 Store/helper。
- 用户级缓存与 in-flight 绑定 `userId + authGeneration`；退出、切号或认证代次变化后，旧响应不得回写当前资料、提示或导航。
- 公共缓存只保存与访问者无关的数据，viewer 个性化字段按当前身份组装。
- 写入成功后精确失效相关资源 scope，不用所有页面无条件强制刷新代替一致性管理。
- 新增读取逻辑测试首次进入、重复 `onShow`、并发请求、退出/切号与迟到响应；优先参考 [tests/request-graph.test.mjs](tests/request-graph.test.mjs) 与现有领域缓存测试。

## UI 与平台约束
- 品牌、主题、按钮、WXSS 选择器和安全区规则统一维护在 [DESIGN.md](../DESIGN.md)；修改 UI 时按该文档的 UniApp 细则与验收清单核对，不复制另一套规范。
- 主题接入 `theme.json`、`App.vue` CSS 变量与 `store/theme.js` 的现有切换链路，不绕过用户主题偏好。
- 优先系统导航栏；自定义导航栏需有明确需求，并保持正确高度与返回行为。
- Node 测试（含 Vue SFC/jsdom 挂载）不替代 App/微信小程序编译和真机 UI 验收；跨平台改动需注明实际覆盖的平台。

## 验证命令
在 `uniapp/` 执行；依赖按需通过 `npm install` 安装。[package.json](package.json) 没有 npm scripts，不使用 `npm test`。

```bash
node --test tests/*.test.mjs
# 修改本指南时同步 GEMINI.md，并检查镜像一致
cmp AGENTS.md GEMINI.md
```
