# FRONTEND API LAYER GUIDE

## OVERVIEW
`api/` 是用户端页面访问后端的唯一门面层。页面、组件和 store 不应直接触达 `utils/request.js`；统一通过这里按业务域组织接口。

## STRUCTURE
```text
api/
├── achievement.js   # 成就列表、用户成就、称号
├── auth.js          # 短信、登录、OAuth、绑定手机号
├── challenge.js     # PK 挑战发送/接受/拒绝
├── event-news.js    # 赛事情报列表、焦点赛事、详情
├── friend.js        # 好友列表、好友申请、搜索用户
├── match.js         # 对局、当前对局、历史、交锋、公开对局、对手
├── notification.js  # 通知列表、已读、删除、未读数
├── rank.js          # 段位、榜单
├── rules.js         # 规则目录、详情、术语
├── season.js        # 当前赛季、赛季榜、赛季报告
├── share.js         # 对局/赛事分享
├── social.js        # 动态流、发帖、评论、点赞
├── stats.js         # 竞技分析、趋势、对手强度
├── tournament.js    # 赛事列表、创建、详情、报名、对阵
├── user.js          # 用户资料、昵称、推送 token
└── venue.js         # 球房列表、详情、附近球房、签到、提交
```

## REQUEST CONTRACT
- 所有模块统一基于 `utils/request.js` 的 `get/post/put/del`。
- `utils/request.js` 会自动附带 `Authorization: Bearer <token>`。
- 全局成功判定是 `data.code === 0 || data.success`。
- 401 会由 request 层统一清 token、默认 toast 并跳转登录页。
- 需要“静默失败”时，给请求透传 `{ silent: true }`，不要在页面里重复实现 401 兜底。

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| 登录/绑定/短信 | `auth.js` | 与 `store/user.js` 联动最紧密 |
| 用户资料和设置 | `user.js` | 昵称、资料、推送 token |
| 对局流转与观战 | `match.js` | 同时覆盖 `/api/match`、`/api/public`、`/api/opponent` |
| 通知中心 | `notification.js` | 与 `store/notification.js`、`subPages/notification/index.vue` 联动 |
| 社交关系链 | `friend.js`, `social.js`, `challenge.js` | 当前仓库没有单独的 `follow.js` 文件 |
| 赛事与赛季 | `tournament.js`, `season.js` | 赛事分享和赛季报告是独立模块 |
| 球房与规则 | `venue.js`, `rules.js` | 对应分包页面较多 |
| 分享和竞技分析 | `share.js`, `stats.js` | 对局海报、赛季报告、用户竞技分析 |

## CONVENTIONS
- 页面层不要直接 import `utils/request.js`；只能 import `api/*.js`。
- 一个模块尽量对应一个后端业务域，方法名保持 `getXxx/createXxx/updateXxx/deleteXxx/markXxx` 风格。
- API 层只做请求和参数组织，不写 toast、弹窗、跳转等 UI 行为。
- 新接口先确认后端 `.api` 契约，再补前端封装，避免页面直接依赖未稳定路径。
- 公共接口也放在已有领域文件中维护，不要为了 `/api/public` 单独再造一层页面请求。
- 如果多个页面复用相同的请求参数整形逻辑，优先提炼到 API 层或 `utils/*.js`，不要复制粘贴。

## ANTI-PATTERNS
- 禁止在页面里直接 `uni.request`。
- 禁止在页面里绕过 `api/*.js` 直接调用 `utils/request.js`。
- 禁止在 API 层做页面状态管理、副作用导航或重复认证处理。
- 禁止保留已经不存在的旧模块说明，如 `follow.js`、`mall.js`、`order.js`、`favorites.js`、`address.js`。
