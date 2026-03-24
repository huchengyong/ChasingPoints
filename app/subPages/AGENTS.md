# SUBPAGES GUIDE

## OVERVIEW
`subPages/` 承载非 Tab 主流程页面，必须通过 `pages.json` 的 `subPackages` 注册。这里主要承接用户扩展页、对局详情链路、社交、赛事、球房、规则、赛季和通知中心。

## CURRENT SUBPACKAGES
```text
subPages/
├── agreement/     # userAgreement、privacyPolicy
├── achievement/   # index、detail、titles
├── help/          # feedback
├── match/         # matchDetail、playing、matchResult、shareResult
├── notification/  # index（通知中心）
├── rules/         # index、detail、glossary
├── season/        # index、report
├── social/        # feed、postCreate、friendList、addFriend、friendRequests、challenges、pkReport
├── tournament/    # index、detail、create、bracket
├── user/          # matchHistory、rankExplain、notification、opponentRecord、h2hRecord、settings、statsDetail
└── venue/         # index、detail、submit
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| 用户历史与统计 | `subPages/user/*.vue` | 与 `api/user.js`、`api/stats.js`、`api/match.js` 联动 |
| 对局进行中/结果/分享 | `subPages/match/*.vue` | 强依赖 `utils/websocket.js` 与分享能力 |
| 社交链路 | `subPages/social/*.vue` | 对应 `api/social.js`、`api/friend.js`、`api/challenge.js` |
| 赛事模块 | `subPages/tournament/*.vue` | 创建、报名、对阵图和详情都在这里 |
| 球房与规则 | `subPages/venue/*.vue`, `subPages/rules/*.vue` | 与 LBS/规则内容接口联动 |
| 赛季与通知 | `subPages/season/*.vue`, `subPages/notification/index.vue`, `subPages/user/notification.vue` | 通知中心和用户页通知设置是两条不同链路 |
| 协议与反馈 | `subPages/help/*.vue`, `subPages/agreement/*.vue` | 静态说明和用户反馈入口 |

## CONVENTIONS
- 新增或修改页面时，先确认 `pages.json` 对应分包是否已注册。
- 页面默认使用 `script setup` + SCSS；样式优先拆到同名 `.scss` 文件。
- 列表时间展示统一用 `utils/format.js` 的 `formatRelativeTime`。
- 图标统一优先用 `uni-icons`。
- 主题变量要兼容 `theme.json`、`App.vue` 全局 CSS 变量和 `store/theme.js`。
- 自定义按钮要隐藏 `button::after`；涉及 `width: 100% + padding` 时记得加 `box-sizing: border-box`。
- 自定义导航栏只在确有设计要求时使用，并且高度、返回行为要与系统导航栏一致。
- 对局结果页、进行中页、海报页经常共享同一份对局状态，不要在多个页面各自复制 match 同步逻辑。

## ANTI-PATTERNS
- 禁止继续沿用已不存在的旧目录认知，例如 `order/`、`product/`、`category/`。
- 禁止在子页面直接封装请求或直接调用 `utils/request.js`。
- 禁止写死浅色/深色颜色而不接入现有主题体系。
- 禁止混淆 `subPages/notification/index.vue` 与 `subPages/user/notification.vue` 的职责。
