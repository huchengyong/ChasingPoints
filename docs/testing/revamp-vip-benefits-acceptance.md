# VIP 权益重构验收记录（`revamp-vip-benefits`）

> 对应 OpenSpec 变更 `revamp-vip-benefits`。本文只记录某一次验收的事实，不是长期开发约束；开发规则见各子系统的 `AGENTS.md`。
>
> 状态（2026-10-22）：后端与 UniApp 客户端实现完成；后端已部署测试环境（见第 4 节）。改动已按主题提交（见 `git log`），**真机验收尚未进行。** 下面分开记录自动化执行、人工确认与未执行项。

## 1. 本次自动化执行

执行环境：本机，隔离的内存 SQLite 与 miniredis；没有连接开发库、测试库或生产库，没有读取 `.env`。

| 范围 | 命令（工作目录） | 结果 |
| --- | --- | --- |
| 后端构建 | `go build ./...`（`backend/`） | 通过 |
| 后端测试 | `go test ./... -count=1`（`backend/`） | 全部通过（2026-10-22 部署前执行）。部署前的一次运行曾偶发 `internal/pkg/matchinvite` 失败，已修正测试，见第 2 节 |
| UniApp 测试 | `node --test tests/*.test.mjs`（`uniapp/`） | 872 项全部通过 |
| 管理后台测试 | `node --test tests/*.test.mjs`（`admin/`） | 9 项全部通过（成长规则默认值） |
| 规格校验 | `openspec validate revamp-vip-benefits --type change --strict --no-interactive` | 通过 |
| 空白检查 | `git diff --check` | 通过 |

覆盖的关键行为：

- **公平边界**：一场会员对免费用户的 V2 排位，四种会员组合下双方排位分变化完全一致。同一场对局按旧规则会给会员多 12 分，证明该测试对会员差异敏感。
- **会员身份只读且不拖垮接口**：排行榜、好友、对局详情、分享数据、约球五处批量解析档位，查询次数固定、不随行数增长；档位查询失败时按 0 返回且接口成功。长期缓存的对局核心摘要不含会员字段。
- **表现分析**：表现分算法与独立的 Python 实现对拍；服务端会员门禁（免费、到期、续开）；免费用户只拿到样本数与本月预览；旧规则、规则切换记录、不计分对局不进入样本；读取上限触发时被截断的月份标为不完整。
- **单场复盘**：门禁顺序（已完成、参与者、有效会员），非参与者与未完成对局统一拒绝；按球种生成并在明细对不上时降级到局级数据；超长对局返回 `detail_truncated`。
- **客户端**：会员身份工具与组件（含真实挂载）、各页面接入、好友主页参数透传、海报模板（标准模板的画布调用序列与改动前逐项一致）、表现分析 Store 的首进、重复 `onShow`、切号、迟到响应、两个新页面的真实挂载（含合规模式下无开通入口）、会员中心改版。
- **变异验证**：对每组新增行为临时破坏实现，确认测试会失败，再恢复并核对文件一致。其中一处变异（海报「对局时间」行不下移）最初没被发现，已加强断言后重做。

页面渲染检查（H5 构建 + 无头 Chrome，数据来自本地 mock，不是真实后端）：

- 已看过截图：排行榜（领奖台、列表、我的排名，含暗色）、首页排行榜摘要、我的、好友列表、好友主页、约球列表、约球等待页、对局中、战报页、对局详情、表现分析（会员、锁定、暗色）、单场复盘（斯诺克、中式八球、暗色）、会员中心（有效、到期）、会员海报（尊享、进阶、标准档、免费用户对手为会员）。
- 暗色用向页面根节点注入 `dark-mode` 类的方式检查（H5 下无法靠 `theme_mode` 切换），图标颜色等由 JS 计算的部分不在此范围。
- 检查中发现并修复的问题：头像框翼饰在列表里被裁切或压住文字（引入三种占位方式）、领奖台昵称被框遮住、对手标识压到文字、会员海报头部间距偏紧。

小程序构建（HBuilderX CLI 的 `mp-weixin`，本机产物）：

- 2026-10-23 生产模式（`NODE_ENV=production`）构建，日志在 scratchpad `build-mp-release.log`，构建完成无错误。
- 主包 0.954 MB（1000202 字节，上限 2 MB），按 `app.json` 的 `subPackages` 根目录切分统计。
- `subPages/user/performance`、`subPages/match/matchReplay` 位于分包（`subPages/user`、`subPages/match`），不在主包页面列表。
- `components/memberAvatar` 因被多个分包共用位于主包，约 4 KB。

App 资源构建（HBuilderX CLI 的 `app-plus`，生产模式）：

- 构建完成无错误，两个新页面的样式与脚本编译进 `app-service.js`。
- 仅生成资源，未做云打包，因此没有 `.apk` / `.ipa`。

## 2. 部署前修复的测试失败

- **已删除** `backend/migrations/career_achievement_migration_test.go`（`TestCareerAchievementMigrationContainsV2Catalog`）。它检查 `uniapp/static/images/achievements/*.png` 是否存在，而提交 `04b0d10` 已把这些大图移到七牛 CDN。删除后 `go test ./migrations/` 通过。这是用户明确要求的改动，不影响迁移 SQL 本身。
- **偶发失败的修复**：`internal/pkg/matchinvite` 的 `TestSignerRejectsTamperedAndExpiredInvite` 原先把签名最后一个字符改为 `x`。签名是 43 个 base64url 字符，末字符只有 4 个有效位，原字符恰为 `w`、`x`、`y`、`z` 之一时解码字节不变，篡改会被接受（约 1/16 概率）。测试改为替换签名段首字符（6 个有效位），单独运行 500 次全部通过。实现未改动。

## 3. 用户人工确认

暂无。以下项目都需要真实环境，见第 4 节。

## 4. 未执行或阻塞

- **任务 1.2a（已完成，只读核对）**：2026-10-22，经 SSH 登录测试服务器（`chasingpoints-dev`，`/www/server/chasing-points/shared/.env` 的 `MYSQL_DSN`）与生产服务器（`chasingpoints-prod`，`/www/wwwroot/api.zhuifen.cn/.env` 的 `MYSQL_DSN`），在服务器上用 `mysql --batch` 执行 `SELECT COUNT(*) FROM member_rights_configs` 与对应的 `SELECT`。结果：测试库 0 行，生产库 0 行。没有已保存的配置覆盖代码默认值，运行时回退到 `DefaultMemberGrowthRulesConfig`（门槛 10/40/120/300、每日上限 8），与新数值一致，因此没有修改任何环境的配置。只读脚本未打印 DSN 或密码；没有写入任何数据。
- **任务 9.2（发行包）**：未完成发行动作。已完成生产模式的 `mp-weixin` 与 `app-plus` 资源构建，并确认主包体积与分包归属（见第 1 节）。未完成：App 云打包（需要 DCloud 账号登录，生成 `.apk` / `.ipa`），以及微信小程序上传到后台（需要上传密钥，并会在后台生成对外可见的版本）。这两项需要用户确认后再做。
- **任务 9.3 的部署部分（已完成）**：2026-10-22 通过 `backend/deploy-dev.sh` 部署到测试环境（`chasingpoints-dev`，`api-zhuifen.kekemate.cn`）。发布号 `20261022172041-04b0d106c394-dirty`（部署时工作区有未提交改动，所以带 `-dirty`；部署后仅修正了 `user_ranking.go` 中一处注释的位置，没有行为改动）。
  - 迁移：本次没有新增 SQL 迁移，`goose` 输出 `no migrations to run`，当前版本 `20261019120000`。
  - 服务：`chasing-points.service` 为 active，启动于 2026-10-22 09:21:24 UTC；本机 `/api/admin/exists` 返回 200；公网 `https://api-zhuifen.kekemate.cn/api/admin/exists` 返回 200。
  - 新接口：`GET /api/member/performance`、`GET /api/member/match-replay` 未带 token 返回 401（路由已注册，不是 404）。
  - 启动后 10 分钟内 journal 无 panic、fatal 或 error。部署脚本健康检查时出现过一次 `curl (7)`，是服务刚启动时的连接失败，之后检查全部正常。
  - 没有部署管理后台测试站，也没有构建 App 或小程序包；真机需要的包仍由 HBuilderX 出。
- **任务 9.3 的真机验收（未进行）**：需要用测试账号在真机上确认：
  - 会员与免费用户对局时，双方看到的头像框与 VIP 标识。
  - 排行榜上的会员标识变化不超过一个缓存周期（20 秒）。
  - 表现分析与单场复盘在会员、免费、到期三种状态下的表现。
  - 合规模式下没有任何支付入口。
  - 会员海报：海报用 `uni.getImageInfo` 读取头像与 CDN 头像框，微信小程序需要这些域名在 downloadFile 合法域名里。读不到时海报只缺头像框并用占位头像，不会生成失败，但真机上要确认实际效果。
- **范围外**：观战与历史对局使用的公开详情接口不带会员字段，这些页面按非会员展示；原生 iOS、管理后台与官网未改动（新字段均为可选，iOS 由迁移工作接入）。
