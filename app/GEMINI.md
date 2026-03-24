# APP FRONTEND GUIDE

## OVERVIEW
`app/` 是 UniApp Vue3 用户端，当前代码重点覆盖 4 个 tab 页面、11 个分包页面簇、统一请求层、主题系统、登录与绑定手机号链路、对局实时同步、战报分享和一批纯逻辑测试。

## GLOBAL RULES
- 用简体中文沟通。
- `AGENTS.md` 与 `GEMINI.md` 需要保持同步；仓库里当前没有 `IFLOW.md`。
- 任何目录说明都以当前仓库实际文件为准，不要沿用旧模块名或旧页面结构。

## STRUCTURE
```text
app/
├── App.vue                    # 全局生命周期、主题应用、Push 初始化、前台对局提醒
├── main.js                    # Vue3 SSR App 入口，挂载 Pinia
├── pages.json                 # 主包页面、tabBar、分包注册
├── theme.json                 # UniApp 主题变量
├── pages/                     # 主包页面
│   ├── welcome/
│   ├── login/
│   ├── index/
│   ├── match/
│   ├── ranking/
│   ├── social/
│   └── user/
├── subPages/                  # 分包页面
├── api/                       # 页面唯一请求门面
├── components/                # bindPhone、agreementConsentSheet、gameTypeModal
├── store/                     # Pinia：user/theme/notification/friendRequest
├── utils/                     # request、format、websocket、业务纯函数与导航 helper
├── tests/                     # node:test 纯逻辑测试
├── static/                    # 图片、图标、字体
└── harmony-configs/           # HarmonyOS 配置
```

## CURRENT ARCHITECTURE
- 入口是 [app/main.js](/Users/wisesearch/Projects/ChasingPoints/app/main.js)，使用 Pinia；状态持久化由 [app/store/index.js](/Users/wisesearch/Projects/ChasingPoints/app/store/index.js) 注册的 `pinia-plugin-persistedstate` 完成。
- [app/App.vue](/Users/wisesearch/Projects/ChasingPoints/app/App.vue) 仍使用 Options API，因为需要承接 UniApp app 级生命周期；页面组件默认继续优先用 `script setup`。
- [app/pages.json](/Users/wisesearch/Projects/ChasingPoints/app/pages.json) 当前注册 6 个主包页面和 11 个分包根目录。
- [app/utils/runtime-config.js](/Users/wisesearch/Projects/ChasingPoints/app/utils/runtime-config.js) 负责按环境解析网络基地址；[app/utils/request.js](/Users/wisesearch/Projects/ChasingPoints/app/utils/request.js) 统一处理 token、401、业务成功判定；[app/utils/websocket.js](/Users/wisesearch/Projects/ChasingPoints/app/utils/websocket.js) 负责 match/user 两条 WS 链路。
- 页面层只能依赖 `api/*.js`；业务纯函数尽量沉到 `utils/*.js` 并在 `tests/*.test.mjs` 里覆盖。

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| 页面注册、导航栏、分包 | `pages.json` | 新页面先确认主包还是分包 |
| 全局主题、Push、前台弹窗 | `App.vue`, `theme.json`, `store/theme.js` | 主题和导航栏颜色要一起看 |
| 登录态与用户信息 | `store/user.js`, `api/auth.js`, `components/bindPhone.vue` | `needBindPhone` 和 token 在这里汇总 |
| 通知状态 | `store/notification.js`, `utils/notification.js`, `api/notification.js` | 有页面和 store 双向联动 |
| 请求层 | `utils/request.js`, `utils/request-response.js` | 401、业务成功判定、静默请求都在这里 |
| 对局实时同步 | `utils/websocket.js`, `utils/match-action.js`, `subPages/match/*.vue` | 要同时理解 revision/snapshot 和页面跳转 |
| 首页/登录漏斗纯逻辑 | `utils/home-index.js`, `utils/entry-funnel.js`, `tests/*.test.mjs` | 很多 UI 规则已抽纯函数 |
| 球房提交流程 | `subPages/venue/submit.vue`, `utils/venue-submit.js` | 当前只提交基础字段 |
| 战报/分享 | `utils/posterGenerator.js`, `api/share.js`, `subPages/match/shareResult.vue`, `subPages/social/pkReport.vue` | 涉及画布和分享数据整形 |

## FRONTEND CONSTRAINTS
- 这是跨平台 UniApp 项目，页面层不要直接用 `uni.request`，统一走 `api/*.js`。
- 页面默认优先使用 `script setup` + SCSS；`App.vue` 作为 app 生命周期例外。
- 样式优先拆到同名 `.scss` 文件；全局共享样式才放进 `App.vue` 或 `uni.scss`。
- 列表时间展示优先复用 [app/utils/format.js](/Users/wisesearch/Projects/ChasingPoints/app/utils/format.js)。
- 对局写操作、登录漏斗、球房提交等规则，优先提炼为 `utils/*.js` 纯函数并补测试，而不是把规则散在页面里。
- Push、主题、前台对局提醒属于 app 级行为，优先改 `App.vue`，不要把相同逻辑复制到页面。

## THEME AND UI CONSTRAINTS
- 主题变量必须同时兼容 `theme.json`、`App.vue` 中的 CSS 变量和 `store/theme.js` 的运行时切换。
- 主题色背景按钮文字统一使用白色 `#ffffff`。
- 自定义按钮必须隐藏 `button::after`。
- 页面最外层容器要注意 `box-sizing: border-box` 和首屏 margin collapse，避免顶部漏白。
- 没有明确设计要求时，优先使用系统导航栏；自定义导航栏要和系统高度、返回行为保持一致。

## TEST AND COMMANDS
```bash
# 安装依赖
npm install

# 运行当前纯逻辑测试
node --test tests/*.test.mjs
```

## KNOWN FACTS
- `package.json` 当前只有 `dependencies`，没有 `scripts`；默认不要假设可以直接 `npm test`。
- 当前 HTTP 与 WebSocket 基地址由 `utils/runtime-config.js` 统一管理：开发环境默认走 tunnel，生产环境默认走正式域名。
- `App.vue` 里会直接调用 `post('/api/user/push-token')`，这是 app 级基础设施调用，不是页面层越界。

## ANTI-PATTERNS
- 不要在 `.vue` 页面里直接 `uni.request` 或直接 import `utils/request.js`。
- 不要把业务规则直接埋进页面生命周期，能抽纯函数就抽，并补 `tests/*.test.mjs`。
- 不要继续引用不存在的目录或设计稿目录，例如当前仓库里没有 `design_code/`。
- 不要把旧模块名如 `mall`、`order`、`favorites` 当成当前项目结构。
