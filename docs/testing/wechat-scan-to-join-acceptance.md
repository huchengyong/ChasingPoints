# 微信扫普通链接二维码开局与分享：验收记录

对应变更：`openspec/changes/add-wechat-scan-to-join-and-share`（规格见 `openspec/specs/scan-to-join`、`universal-entry-links`、`match-share-entry`）（任务 8.1–8.4、9.1–9.6）。
范围：普通链接二维码（`https://www.zhuifen.cn/go/?j=<短码>`）→ 小程序承接页 → 登录即开局；分享卡片；战报 / 交手报告海报二维码；官网 `/go/` 兜底页与下载页改版。小程序只产出微信小程序与鸿蒙，原生 iOS 不涉及扫码。

## 1. 本次自动化执行

环境：本机 macOS（darwin/arm64）；Node v22.19.0；Go 1.24.5；工作区未提交（`HEAD` = `5db7301e45c8`，后端部署标记为 `-dirty`）。

| 项目 | 命令（工作目录） | 结果 |
| --- | --- | --- |
| UniApp Node 测试 | `node --test tests/*.test.mjs`（`uniapp/`） | 1128 个用例，1127 通过、1 失败（见第 5 节，与本变更无关） |
| SFC 编译检查 | `@vue/compiler-sfc` 解析 `linkEntry.vue`、`matchDetail.vue`、`matchResult.vue`、`pages/index/index.vue`、`go/index.vue`、`download.vue`（仓库根） | 6 个文件 parse / template / script 均无错误 |
| 样式编译 | `sass.compile('uniapp/subPages/match/matchDetail.scss')`（仓库根，使用 `admin/node_modules/sass`） | 通过 |
| 官网测试 | `npm run test`（`website/`） | 12 个测试文件、33 个用例全部通过 |
| 官网静态生成 | `npm run generate`（`website/`） | 通过，产物含 `/go/index.html`、`/go/mH43X7dRHZ.txt` |
| 后端静态检查 | `go vet ./...`（`backend/`） | 无输出 |
| 后端测试 | `go test ./...`（`backend/`） | 全部通过 |
| 后端格式 | `gofmt -l`（本次新增/修改的 Go 文件） | 无输出 |
| iOS 影响核对 | `grep -rn "match/qrcode\|share/match\|invite/resolve\|match-invite" iOS` | 无匹配，原生 iOS 不调用受影响的接口 |
| OpenSpec 校验 | `openspec validate add-wechat-scan-to-join-and-share --strict` | valid |

本次新增的回归测试：

- `uniapp/tests/entry-link.test.mjs`：入口链接生成/解析、双层 URL 编码的 `q` 解码。
- `uniapp/tests/link-entry-flow.test.mjs`（7 条）：未登录预览 → 登录 → 兑换 → 交接；进入即已登录直接兑换；登录期间过期；放弃登录保留预览；重复 `onShow` 不重复兑换；账号切换后迟到响应被丢弃；网络错误不交接凭证。
- `uniapp/tests/share-card.test.mjs`（7 条）：公开已结束/进行中标题；分享者是选手 2 时顺序不交换；公开视角字段；私密/已取消/未加载回落通用卡片（且不含昵称与比分）；8 字截断。
- `uniapp/tests/poster-generator.test.mjs`（新增 3 条）：公开对局印 `?m=<对局ID>`、私密或缺失印首页；二维码超过容量时不抛错、不画任何格子；公开/私密两种海报都能生成并带对应旁注。
- `backend/internal/pkg/matchinvite/code_test.go`、`backend/internal/logic/match_invite_code_test.go`、`backend/internal/logic/public/preview_public_match_invite_logic_test.go`：短码生成与 Redis 存取、短码→token 兑换的共用逻辑、匿名预览的限流与 Redis 失败不放行。
- `website/tests/download-content.test.ts`、`website/tests/seo-assets.test.ts`：下载页只开放小程序入口、鸿蒙无链接；`/go` 不进 `sitemap.xml`。

## 2. 开发环境部署与接口验证

- 部署命令：`./deploy-dev.sh`（`backend/`）；发布标识 `20261026213100-5db7301e45c8-dirty`；远端 `goose` 输出 `no migrations to run. current version: 20261025115843`，本次改动不含 DDL，未应用任何迁移。
- `POST https://api-zhuifen.kekemate.cn/api/public/match-invite/preview`（匿名、不存在的短码）：HTTP 200，`{"success":false,"message":"匹配二维码已失效，请让对方刷新二维码","preview":null}`，未返回 token 或用户 ID。
- `POST https://api-zhuifen.kekemate.cn/api/match/invite/resolve`（未登录）：HTTP 401。
- 部署前同一路径返回 404，说明新接口只在本次发布后可用。

## 2.1 生产后端发布（2026-10-27）

- 提交：`be755d1 feat(match): 匹配码改为普通链接二维码并支持登录后兑换`，已推送 `origin/devlop`。
- 发布命令：`deploy-prod.sh`（`backend/`）；发布标识 `20261027180027-be755d10d24b`；待执行迁移 0 个（发布前用 goose status 核对为 0）。
- 重启结果：`local health: http=200`、`public health: http=200`，启动后流水 `error=0 severe=0`；旧二进制备份在 `deploy-stage-20261027180027-be755d10d24b/backup/chasing_points.live`。
- 线上接口实测：`POST https://api.zhuifen.cn/api/public/match-invite/preview`（匿名、不存在的短码）→ 200，返回 `success:false` 与统一失效文案；`POST https://api.zhuifen.cn/api/match/invite/resolve`（未登录）→ 401。发布前两者均为 404。
- 说明：体验版是生产构建（`api.zhuifen.cn`），发布生产后端之前 `qrcode_data` 仍是裸 token，微信扫出来只有一串字符；这是首次真机测试失败的原因。

## 2.2 二维码识别率修复（2026-10-27，待重新上传体验版验证）

- 现象：码的内容已是链接（模块变稀疏），但微信与系统相机长时间识别不到。
- 原因：`utils/local-qrcode.js` 用 uqrcodejs 默认参数，`margin=0` 没有静默区，码顶到画布边缘；弹层面板是深色，码四周等于零留白。
- 修复：`renderLocalQRCode` 增加 `margin = size * 0.08` 的留白并固定纯黑白。
- 修复（对齐）：`.qr-image` 由 `440rpx` 改为 `240px`，与画布绘制尺寸一致；原先 440rpx 在 3 倍屏换算成约 252px，码被非整数放大且左右白边差约 12px。
- 反面验证：曾把画布与 `size` 提到 660，微信小程序端出的是裁切图（缺左下定位角、四周留白不对称）。小程序 canvas 的绘制坐标就是 CSS 尺寸，`size` 不能超过显示尺寸；已回退到 240，只保留静默区修复。
- 回归测试：`node --test tests/local-qrcode.test.mjs` 断言留白与颜色。

## 3. 官网发布与线上核对

- 发布方式：`rsync -az --no-perms --omit-dir-times --exclude '.DS_Store' .output/public/ chasingpoints-prod:/www/wwwroot/www.zhuifen.cn/public/`。
- `https://www.zhuifen.cn/go/`：HTTP 200，含 `<meta name="robots" content="noindex, nofollow">`；未登录参数时渲染通用文案。
- `https://www.zhuifen.cn/go/mH43X7dRHZ.txt`：HTTP 200，内容 `fc1d4797f2417ba7db862ec86ae602e7`；根目录 `https://www.zhuifen.cn/mH43X7dRHZ.txt` 同样 200。
- `https://www.zhuifen.cn/download`（301 → `/download/`）：HTTP 200，文案为「微信小程序 / 鸿蒙」，鸿蒙按钮为不可点的「即将上线」。
- `https://www.zhuifen.cn/qr-miniprogram.jpg`：HTTP 200。
- 首页常见问题已改为「在哪里可以开始使用追分？」。

## 4. 微信公众平台配置（用户完成）

- 「扫普通链接二维码打开小程序」已保存：`QR Code Address = https://www.zhuifen.cn/go/`、`Mini Program Path = subPages/match/linkEntry`、`Test Scope = 体验版`。
- 规则状态仍为 **Pending**，未点 Release。按计划在小程序正式发布当天再发布规则。
- 测试链接来源：体验版首页「出示我的匹配二维码」会调用 `GET /api/match/qrcode`，返回 `https://www.zhuifen.cn/go/?j=<短码>`，并把短码写入 Redis `match_invite_code:<短码>`（TTL 跟随 token）。2026-10-27 检查开发环境 Redis，该前缀下 0 个键，说明尚未在体验版里生成过二维码。
- 注意：微信「测试链接」只对小程序开发者与体验成员生效，未注册的新微信号要先加为体验成员，否则扫测试链接打不开体验版。

## 5. 未执行 / 待人工确认

| 项目 | 状态 |
| --- | --- |
| 上传体验版（用户手动） | 已完成（2026-10-27 用户上传） |
| 真机验收（9.4 全项） | 用户人工确认为通过（2026-10-27）；逐项明细未逐条记录 |
| 扫码规则 Release（需小程序正式发布后） | 不在本次范围：用户近期不上线，等正式发布当天再点 Release |
| 规则 Release 前的用户可见行为 | 微信扫 `https://www.zhuifen.cn/go/?j=…` 只按普通链接处理，落到 `/go/` 兜底页，不会拉起小程序；`?j=` 短码因此丢失，需在小程序内重新扫码 |

已知但与本变更无关：`uniapp` 全量测试固定有 1 个失败（`cdn-asset.test.mjs`，由未跟踪的 `uniapp/static/images/.DS_Store` 触发）；`gofmt -l internal/` 另外列出 9 个历史文件，均未在本次改动中修改。

## 6. 已记录的后续事项

1. **`X-Forwarded-For` 可伪造，匿名预览的 IP 限流不生效。** 生产 nginx（`go_chasing_points.conf`）用 `$proxy_add_x_forwarded_for`，而 `backend/internal/utils/http.go` 的 `GetClientIP` 取第一个 IP，客户端自填的头会被当成来源。短码约 71 位熵，枚举不可行；限流只用于防止刷接口，因此风险有限。整改方向：反向代理侧覆盖该头，或后端只信任代理追加的最后一跳。**未在本期处理。**
2. **线上站点出现间歇性 TCP 连接失败。** 从本机连续请求 `https://www.zhuifen.cn/go/`，部分请求在 `connect` 阶段就停在 20 秒超时（`time_connect = 0`），其余请求 TCP 建连 5–7ms、TLS 握手 3.3–4.1 秒。站点 `access_log off`，无法从访问日志侧确认。需与主机侧核对是否有机房/防火墙的新连接限速；微信校验文件此前已通过校验，不影响现有配置。

## 7. 代码复查修复（2026-10-27，未发布）

复查后修正以下问题，均已补回归测试；改动尚未提交、未部署，上面第 2、2.1 节的发布记录不包含这些修复。

| 问题 | 修复 | 回归测试 |
| --- | --- | --- |
| 赛果页分享回调调用 `buildMatchShareCard` 但未导入，微信里点转发会抛 `ReferenceError` | `matchResult.vue` 补导入 | `share-card.test.mjs`：注册分享回调的页面必须从 `share-card.js` 导入所调用的构造函数（去掉修复时该用例失败） |
| 匿名预览限流在 `EXPIRE` 失败时计数键永不过期，整个出口 IP 被永久拦截 | `INCR` 与 `TTL` 放进同一事务管道，键没有过期时间就补设并检查错误 | `TestPreviewRateLimitWindowAlwaysExpires`（旧实现下失败） |
| 官网 `/go/` 对匹配码访客展示首页小程序码，并称「不用再扫码」，照做会丢失邀请 | 文案移到 `website/data/go.ts`；匹配码访客改为提示用微信重扫对方手机上的码，不展示首页小程序码；对局访客说明首页码只能打开首页 | `website/tests/go-content.test.ts` |
| 战报海报二维码第二行旁注与居中的「追分」品牌字重叠 | 二维码区上移，两行旁注都在品牌行之上；海报高度不变 | `poster-generator.test.mjs`：按画布调用检查二维码区与其他文字不重叠（旧布局下失败） |
| 承接页网络失败显示「邀请已失效」，且未登录时预览失败不可重试 | 网络/服务异常返回可重试的 `error` 态（标题「暂时无法打开邀请」+「重试」）；真正失效仍为 `expired` 且不可重试 | `link-entry-flow.test.mjs` 新增 3 条 |
| 签名器构造与 token 校验在 `logic` 与 `logic/match` 各写一份；短码错误映射里有永远走不到的 `ErrExpiredToken` 判断 | 收进 `logic.NewMatchInviteSigner` / `logic.VerifyMatchInvite`，`match` 包委托调用，提示文案不变；删除死判断 | 既有 `match_invite_logic_test.go` 全部通过 |
| 新增代码三处各自拼观战详情路由 | `entry-link.js` 新增 `buildSpectateRoute`，承接页、端内扫码、分享卡片共用（早于本变更的两处未改） | 既有用例覆盖路由值 |
| 任务 3.8 缺少的后端测试 | 补：短码写入失败不返回二维码；分享数据返回归一化 `visibility`；匿名预览只返回展示字段；超限后不查短码 | 对应新增用例 |

本次自动化执行：`node --test tests/*.test.mjs`（`uniapp/`）1133 个用例 1132 通过，唯一失败仍是上文 `.DS_Store` 引起的 `cdn-asset.test.mjs`；`go vet ./...`、`go test ./...`（`backend/`）通过；`npm run test`（`website/`）36 个用例通过，`npm run generate` 成功且产物含 `/go/index.html`；`openspec validate add-wechat-scan-to-join-and-share --strict` 通过。

待人工确认：重新上传体验版后，在赛果页点「转发」确认卡片标题带比分；生成一张公开对局战报海报，确认二维码与「追分」不重叠且可被微信识别。

## 8. 第 7 节修复的真机确认与海报版式修正（2026-10-27）

### 8.1 真机确认（用户提供截图，体验版）

用户重新上传体验版后提供了三张截图：分享给好友的卡片、保存到相册的海报、海报预览页。对局为九球追分练习赛（公开），比分 99:58。

| 项目 | 结果 |
| --- | --- |
| 赛果页转发卡片 | 通过：标题为「PerfectI… 99:58 TimesNew… · 九球追分练习赛」，含比分，昵称按 8 字截断，球种与对局类型正确；卡片封面为赛果页顶部默认截图；卡片标注「未发布的小程序 · 体验版」 |
| 海报二维码与文字 | 通过：两行旁注都在「追分」品牌行之上，无重叠 |
| 海报二维码内容 | 通过：用 macOS CoreImage（`CIDetector` QR 检测）解码保存的海报与预览页截图，均为 `https://www.zhuifen.cn/go/?m=2` |
| 微信扫海报二维码进入小程序 | 未验证：扫码规则仍为 Pending，需在正式发布后确认 |

### 8.2 截图暴露的两个历史问题（早于本变更），已修正

| 问题 | 修复 | 回归测试 |
| --- | --- | --- |
| 练习赛海报右上角显示「排位 ±0」：标签固定写排位，`MatchShareData` 也没有对局类型 | `MatchShareData` 新增 `match_mode`（`NormalizeMatchMode` 归一，经 goctl 生成）；练习赛标签显示「练习赛」，缺少 `match_mode` 的旧数据仍显示排位变化 | 后端 `TestMatchShareDataReturnsNormalizedVisibilityAndMode`；`poster-generator.test.mjs` 练习赛/排位赛/旧数据三种情况（旧实现下失败） |
| 数据卡片左右边距不对称（左 60、右 100），长说明（如「不被对手打断的连续得分峰值」）越出卡片右缘，右上「总结口径与战报页一致」超出卡片区 | 两列卡片改为 60–690 内对称（宽 305、间距 20）；说明文字按卡片宽度计算每行字数、18 号字单行，超出省略；右上说明右对齐到 670；海报高度不变 | `poster-generator.test.mjs`：卡片左缘为 60/385、所有文字在 [60, 690] 内（旧实现下失败） |

改前后对比：在 headless Chrome 里回放单测同款画布调用渲染海报，改前版式与用户真机海报一致，改后卡片对称、说明不越界、标签为「练习赛」。

本次自动化执行：`node --test tests/*.test.mjs`（`uniapp/`）1135 个用例 1134 通过（唯一失败仍为 `.DS_Store` 引起的 `cdn-asset.test.mjs`）；`go vet ./...`、`go test ./...`（`backend/`）通过；原生 iOS 不调用 `/api/share/match`，无需改动。

待人工确认：后端发布 `match_mode` 后重新生成一张练习赛海报与一张排位赛海报，确认标签与卡片版式。练习赛标签依赖新后端返回 `match_mode`；后端未发布前，新客户端仍显示「排位 ±0」。

## 9. 复查修复的发布（2026-10-27，用户授权）

本次发布包含 `04988e9`（第 7 节）与 `4479570`（第 8 节），已推送 `origin/devlop`（`f3d08fd..4479570`）。

| 环境 | 命令与结果 | 线上核对 |
| --- | --- | --- |
| 开发后端 | `./deploy-dev.sh`（`backend/`）；发布标识 `20261027214942-44795708e3ec`；goose：no migrations to run（版本 `20261025115843`） | `systemctl is-active` = active，`current` 指向新发布；匿名预览（不存在的短码）返回统一失效文案；未登录调用 `/api/match/invite/resolve`、`/api/share/match` 均为 401；Redis 中 `rl:invite_preview:<ip>` 带 TTL（49 秒），第 7 节限流修复生效 |
| 生产后端 | 先预演 `./deploy-prod.sh < /dev/null`：线上 `be755d1` → 待发布 `4479570`，包含上述两个提交，待执行迁移 0 个；再 `echo yes \| ./deploy-prod.sh`，发布标识 `20261027215114-44795708e3ec` | 重启后 `local health: http=200`、`public health: http=200`，启动后流水 `error=0 severe=0`；在服务器本机用 `--resolve` 核对：匿名预览返回统一失效文案，兑换与分享接口未登录 401；`restart-prod.sh --status` 正常。旧二进制备份在 `deploy-stage-20261027215114-44795708e3ec/backup/chasing_points.live` |
| 官网 | `npm run test`（36 通过）、`npm run generate`；先在服务器打包备份 `/www/wwwroot/.website-backups/www.zhuifen.cn-public-20261027215207.tar.gz`，再按第 3 节的 `rsync` 同步 | `/go/`、`/go/?j=…`、`/download/`、`/qr-miniprogram.jpg`、两处校验文件均 200，校验文件内容不变；线上 `/go/` 脚本含「重新扫描对方手机上的匹配码」，不再含「不用再扫码」 |

未在线上核对：生产 Redis 用 `redis-cli` 未列出限流键（可能需要认证或不同库），限流修复以开发环境核对与回归测试为准；分享数据的 `match_mode` 需登录态，未用真实账号调用，以回归测试为准。

仍待人工：重新上传体验版后，各生成一张练习赛与排位赛海报，确认「练习赛」标签与对称卡片；扫码规则 Release 仍等小程序正式发布当天。
