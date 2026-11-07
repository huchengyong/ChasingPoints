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
| 主导航（四个 Tab、中间扫码按钮、消息角标） | [pages.json](pages.json) 的 `tabBar`、小程序原生组件 [custom-tab-bar/](custom-tab-bar/)、[utils/main-tab-bar.js](utils/main-tab-bar.js)（纯逻辑与平台分支）、[utils/use-main-tab-bar.js](utils/use-main-tab-bar.js)（Tab 页接入） |
| 扫码（开局、执裁） | [utils/quick-scan.js](utils/quick-scan.js)（流程）、[utils/quick-scan-entry.js](utils/quick-scan-entry.js)（运行时接线）、[subPages/match/scanConfirm.vue](subPages/match/scanConfirm.vue)（确认页） |
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
- 主导航为「首页 / 赛讯 / 扫码 / 消息 / 我的」：小程序用自定义 TabBar（原生组件，不经 uni-app 编译，`uni.setTabBar*` / `hideTabBar` 对它不生效），鸿蒙用原生 `midButton`；新增或调整 Tab 时同步 `pages.json`、`custom-tab-bar/`、`utils/main-tab-bar.js` 的 `MAIN_TABS` 与 `theme.json` 图标。Tab 页用 `useMainTabBar` 接入，弹层隐藏 / 恢复 TabBar 只用 `setMainTabBarVisible`，不直接调用 `uni.hideTabBar` / `uni.showTabBar`。
- 所有「扫对手码 / 裁判码」入口只调用 `startQuickScan`（先扫后选）；页面里不要再自带球种弹窗、扫码或开局代码，扫码凭证只经内存交接，不进路由、本地存储或日志。
- `uni` 在小程序里是编译期注入的，`utils/` 中的纯逻辑不要依赖 `globalThis.uni`，由调用方显式传入 `uniApi`。
- UniApp 目前只编译微信小程序和鸿蒙（iOS / Android 改为原生开发）。Node 测试（含 Vue SFC/jsdom 挂载）不替代微信小程序 / 鸿蒙的编译和真机 UI 验收；改动需注明实际覆盖的平台。
- 公开界面展示他人会员身份（头像框、VIP 标识）统一使用 [components/memberAvatar.vue](components/memberAvatar.vue) 与 [utils/member-identity.js](utils/member-identity.js)，不在页面里手写头像框或标识；字段缺失按非会员处理。头像框素材很宽，按周围空间选择占位方式（默认溢出、`reserve-frame`、`fit-frame`），不要靠加大父容器边距硬撑。会员分析页（表现分析、单场复盘）的开通入口只由 `canShowPaidMembershipEntry()` 控制。
- 微信小程序主包上限 2MB，`static/` 只放必须随包的小资源。段位、成就、头像框、默认封面和装备图（`static/images/gear/`）等大图放在 [cdn-assets/](cdn-assets/)，目录结构与七牛桶 key 一致，需上传到 CDN 后才生效；页面直接绑定后端返回的 `icon` 时用 [utils/cdn-asset.js](utils/cdn-asset.js) 的 `resolveCdnAssetUrl` 转换。

## 验证命令
在 `uniapp/` 执行；依赖按需通过 `npm install` 安装。[package.json](package.json) 没有 npm scripts，不使用 `npm test`。

```bash
node --test tests/*.test.mjs
# 修改本指南时同步 GEMINI.md，并检查镜像一致
cmp AGENTS.md GEMINI.md
```
