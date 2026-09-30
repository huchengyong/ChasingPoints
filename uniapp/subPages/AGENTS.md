# UNIAPP SUBPAGES GUIDE

本目录仅补充分包页面规则；请求、主题和测试遵循[UniApp 指南](../AGENTS.md)。

## 注册与导航
- 新增/移动页面前检查 [pages.json](../pages.json) 的 `subPackages`、路径与导航配置。目录存在不代表已注册；分包根也不一定位于 `subPages/`（例如 `pages/ranking`）。
- 改路径时一起核对调用方、分享入口与兼容跳转，不按旧页面清单恢复已经收口的入口。
- [notification/index.vue](notification/index.vue) 是通知中心，[user/notification.vue](user/notification.vue) 是通知偏好设置，不能混淆职责。

## 跨页面状态
- 对局进行中、结果、详情与分享页复用现有对局同步规则；修改前一起检查 [utils/websocket.js](../utils/websocket.js) 与 [utils/match-action.js](../utils/match-action.js)，不要每页复制 snapshot/revision 处理。
- 进出页面不能额外建立重复读取或 WS 订阅；检查生命周期中的订阅、清理与身份变化行为。
- 时间格式复用 `utils/format.js`；相对时间展示使用其 `formatRelativeTime`，不复制页面专属实现。
