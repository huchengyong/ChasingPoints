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
