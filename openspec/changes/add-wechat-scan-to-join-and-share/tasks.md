## 1. 入口链接纯逻辑（UniApp）

- [x] 1.1 新增 `uniapp/utils/entry-link.js`：`buildEntryLink`、`parseEntryLink`、`decodeLaunchQuery`；只接受 `https://www.zhuifen.cn/go`（带或不带结尾斜杠），短码校验为 12 位 base62，对局 ID 必须为正整数，`j` 与 `m` 同时出现或不合法时视为首页
- [x] 1.2 `uniapp/tests/` 新增测试：三种链接的生成与解析往返；其他域名、`http`、子域名、路径前缀相似（如 `/gox`）被拒；`j`/`m` 同时出现、非法短码、非正整数 ID 回落为首页；`q` 被二次编码或缺失时的处理

## 2. 端内扫码兼容链接码（UniApp，先于后端切换发布）

- [x] 2.1 `uniapp/api/match.js` 新增 `resolveMatchInviteCode`（`POST /api/match/invite/resolve`）与 `previewPublicMatchInvite`（`POST /api/public/match-invite/preview`），参数用严格的对象字面量，不传 `undefined`
- [x] 2.2 `match-scan.js`：先识别入口链接（匹配码 → 兑换；对局 → `open_match`；其他网址 → 无效），再走裁判码与裸 token；兑换成功返回与裸 token 相同的 `start_match` 结构
- [x] 2.3 `quick-scan.js`：处理 `open_match`（进入观战详情）；新增 `handoffResolved({identity, inviteToken, opponent})` 写入现有内存交接；`quick-scan-entry.js` 注入兑换接口
- [x] 2.4 测试：链接码兑换成功进入确认页；兑换失败提示失效；「再来一局」扫到非原对手的链接码被拒；裸 token 与裁判码行为不变；`handoffResolved` 在身份不符时不交接；对局链接进入观战详情

## 3. 后端：短码、匿名预览与兑换

- [x] 3.1 `chasing_points.api`：`/api/public` 组新增 `POST /match-invite/preview`；`/api/match` 组新增 `POST /invite/resolve`；`MatchShareData` 新增 `visibility`；在 `backend/` 按指南运行 goctl，不手改生成文件
- [x] 3.2 `backend/internal/pkg/matchinvite/` 新增短码生成（12 位 base62，`crypto/rand`）与 Redis 存储（`match_invite_code:<短码>`，TTL = token 剩余有效期），参照 `wsticket` 的存储写法，带单测（格式、TTL、过期后查不到、写入失败返回错误）
- [x] 3.3 `GetMatchQRCode`：签发 token 后写入短码，`qrcode_data` 返回 `https://www.zhuifen.cn/go/?j=<短码>`；写入失败返回 `success: false`；`invite_token`、`expires_in_seconds` 不变
- [x] 3.4 抽出「短码 → token → 校验 → 邀请人状态」的共用函数；`PreviewMatchInvite` 的现有行为保持不变
- [x] 3.5 实现匿名预览：先校验格式，再按 IP 限流（Redis 固定窗口，每分钟 30 次，`utils.GetClientIP`），再查短码；失败统一返回「匹配二维码已失效，请让对方刷新二维码」；响应不含 token 与用户 ID；Redis 出错返回「匹配二维码服务暂不可用」
- [x] 3.6 实现登录态兑换：返回 `invite_token` 与现有 `MatchInvitePreviewInfo`；不消耗短码与 token
- [x] 3.7 `GetMatchShareData` 返回规范化后的 `visibility`
- [x] 3.8 后端测试：签发返回链接且可兑换；刷新后新旧短码都可用到各自过期；匿名预览不泄露 token/用户 ID，几种失败原因提示相同；邀请人停用失败；限流超限后不查短码；Redis 出错时不放行；兑换接口未登录 401；分享数据带 `visibility`

## 4. 小程序承接页（UniApp）

- [x] 4.1 `pages.json` 在 `subPages/match` 分包注册 `linkEntry`；新增 `subPages/match/linkEntry.vue`：`onLoad` 用 `decodeLaunchQuery` 解析 `q` 并分发（对局 → `redirectTo` 观战详情；首页或解析失败 → `switchTab` 首页）
- [x] 4.2 匹配码流程：未登录时调用匿名预览，展示邀请人头像、昵称、剩余时间、「登录并开局」；已登录（进入时或 `onShow` 发现刚登录）时调用兑换接口，`handoffResolved` 后 `redirectTo` 确认页；只兑换一次；失效时展示失效状态与「返回首页」
- [x] 4.3 短码与 token 只保存在页面内存，不写存储、不打日志；页面遵循 [DESIGN.md](../../../DESIGN.md) 的品牌色与暗色模式
- [x] 4.4 把承接页的状态流转抽成可测的纯逻辑并补测试：未登录预览 → 登录返回 → 兑换 → 交接；进入时已登录直接兑换；登录期间过期；放弃登录保留预览；重复 `onShow` 不重复兑换；迟到响应在身份变化后被丢弃

## 5. 分享卡片与观战引导（UniApp）

- [x] 5.1 新增 `uniapp/utils/share-card.js`：`buildMatchShareCard(detail)` 统一参与者视角与公开视角为选手 1/选手 2 的顺序，生成已结束/进行中标题，昵称截断到 8 个字符；私密、已取消、数据为空返回通用卡片；`buildHomeShareCard()`
- [x] 5.2 测试：参与者视角下分享者是选手 2 时顺序不被交换；公开已结束/进行中标题；私密卡片不含昵称和比分；长昵称截断；排位赛与练习赛文案
- [x] 5.3 在 `pages/index/index.vue`、`subPages/match/matchResult.vue`、`subPages/match/matchDetail.vue` 注册 `onShareAppMessage`（`// #ifdef MP-WEIXIN`），用页面当前数据同步返回卡片
- [x] 5.4 `matchDetail.vue`：观战模式且未登录时显示底部引导条「用追分记录你的比赛」，点击进入登录页；登录后返回并隐藏引导条；参与者视角不受影响
- [x] 5.5 核对分享与引导相关文案中没有任何以奖励引导分享的内容

## 6. 海报二维码（UniApp）

- [x] 6.1 `posterGenerator.js`：新增在海报画布指定位置绘制 `uqrcodejs` 模块矩阵的函数，带留白与旁注；绘制失败时跳过并记录错误
- [x] 6.2 战报海报：按 `visibility` 选择对局链接或首页链接（缺失按私密处理），旁注分别为「微信扫码查看这场比赛」「微信扫码打开追分」
- [x] 6.3 交手报告海报：绘制首页链接二维码
- [x] 6.4 测试：链接选择逻辑（公开/私密/缺失）；二维码绘制抛错时海报仍生成

## 7. 官网

- [x] 7.1 新增 `website/pages/go/index.vue`：静态渲染通用文案，客户端读取查询参数切换匹配码/对局文案，提供下载入口；不请求后端、不显示短码；设置 `noindex`，不加入 `sitemap.xml`
- [x] 7.2 下载页与首页常见问题改为「微信小程序 + 鸿蒙」：去掉 iOS/Android 入口与承诺；鸿蒙未上架时显示「即将上线」且不可点击；小程序码图片由用户提供
- [x] 7.3 微信校验文件放到 `website/public/`（需要用户从公众平台下载提供）
- [x] 7.4 在 `website/` 运行 `npm run test` 与 `npm run generate`，确认产物中有 `/go/` 页面和校验文件

## 8. 自动化验证

- [x] 8.1 在 `uniapp/` 运行 `node --test tests/*.test.mjs`
- [x] 8.2 在 `backend/` 运行 `go vet ./...`、`go test ./...`，对新增 Go 文件执行 `gofmt -l`
- [x] 8.3 核对原生 iOS 不调用受影响的接口（`/api/match/qrcode`、`/api/share/match`），无需改动
- [x] 8.4 `openspec validate add-wechat-scan-to-join-and-share --strict` 通过

## 9. 发布与真机验收（需要用户参与）

- [x] 9.1 用户确认小程序主体已认证、官网的发布方式，并提供校验文件与小程序码图片
- [x] 9.2 先发布能解析链接的客户端体验版，再发布后端开发环境（`backend/deploy-dev.sh`）和官网
- [x] 9.3 用户在微信公众平台配置「扫普通链接二维码打开小程序」：前缀 `https://www.zhuifen.cn/go/`，承接页 `subPages/match/linkEntry`，添加测试链接，测试范围为体验版
- [x] 9.4 真机验收（用户人工确认，2026-10-27）：未注册的微信号扫码 → 预览 → 快捷登录 → 确认页开局；已登录微信号扫码直接进入确认页；小程序端内扫链接码；鸿蒙端内扫链接码；系统相机扫码打开官网兜底页；另一个微信号点开分享卡片看到比分与登录引导；聊天里长按识别海报二维码
- [x] 9.5 核对生产反向代理是否覆盖 `X-Forwarded-For`；如不覆盖，记录为后续事项
- [x] 9.6 在 `docs/testing/` 新增验收记录：自动化结果、真机验收、人工确认情况与未执行项
- [ ] 9.7 经用户确认后发布生产后端与官网；小程序正式发布后，在公众平台发布扫码规则（生产后端与官网已于 2026-10-27 发布；规则 Release 用户近期不上线，暂不做，等正式发布当天再点）
