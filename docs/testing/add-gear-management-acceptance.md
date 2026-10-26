# 装备管理：验收记录

对应变更：`openspec/changes/add-gear-management`（任务 6.3、10.4、12.1–12.4）。
范围：装备目录（品牌、型号、默认图）、用户装备（球杆、皮头、巧克、冲杆 / 跳杆、球房公杆与主力）、对局用杆记录、后台目录维护。UniApp 只编译微信小程序与鸿蒙；iOS / Android 不在本次范围。

## 1. 本次自动化执行

环境：本机 macOS；Node v22.19.0；Go 工具链见 `backend/go.mod`；HBuilderX CLI 构建；MySQL 8.0.42 为临时实例（`--no-defaults`，端口 33099，相对 socket，验收后已停止并删除数据目录，没有使用开发机自己的 mysqld 或 `.env`）。

| 项目 | 命令（工作目录） | 结果 |
| --- | --- | --- |
| 后端静态检查 | `go vet ./...`（`backend/`） | 无输出 |
| 后端测试（SQLite） | `go test ./... -count=1`（`backend/`） | 56 个包全部通过，无失败 |
| 装备 MySQL 端到端 | `RUN_MYSQL_GEAR_TESTS=1 go test ./migrations -run TestMySQLGearEndToEnd -count=1 -v`（`backend/`，临时 MySQL） | 通过 |
| 热点读 EXPLAIN | `RUN_MYSQL_EXPLAIN_TESTS=1 go test ./migrations -run TestMySQLHotReadPlans -count=1 -v`（同上） | 通过，装备 10 条计划见第 2 节 |
| 迁移文件检查 | `TestAddGearTablesMigrationCreatesTablesWithoutCharsetOverride`（含在全量后端测试内） | 通过：建表语句不含 CHARSET / COLLATE |
| UniApp Node 测试 | `node --test tests/*.test.mjs`（`uniapp/`） | 1099 个用例全部通过 |
| 后台 Node 测试 | `node --experimental-strip-types --test tests/*.test.mjs`（`admin/`） | 18 个用例全部通过 |
| 后台类型检查与构建 | `npm run build`（`admin/`） | 通过（`dist/` 已被忽略，未入库） |
| 指南镜像 | `cmp uniapp/AGENTS.md uniapp/GEMINI.md` | 一致 |
| 空白检查 | `git diff --check` | 无问题（见第 6 节执行时点） |
| OpenSpec 校验 | `openspec validate add-gear-management --strict` | 通过 |
| 微信小程序生产构建 | HBuilderX CLI，`UNI_PLATFORM=mp-weixin`、`NODE_ENV=production`，输出到临时目录 | 构建成功，见第 3 节 |
| 鸿蒙编译 | 同上，`UNI_PLATFORM=app-harmony` | 构建成功；`app-config-service.js` 与 `app-service.js` 均含 `subPages/gear`，产物中 `kekemate` 出现 0 次 |

`gofmt -l` 列出的 9 个文件都不是本次改动（`admin_handler_test.go`、`eventnews/helpers.go`、`payment/member_subscription_support.go`、`user/privacy_logic_test.go`、`middleware/active_user_session_route_test.go`、`testsupport/` 下 4 个 schema 文件），按「不顺手清理历史代码」未处理。

主要回归测试覆盖：

- 目录：品牌与型号查询只返回启用项、分页稳定且页大小封顶、SQL 条数不随行数增长；他人视角看不到自填的品牌与型号名；管理员与匿名令牌不能读用户端装备接口。
- 用户装备：每人 30 件上限、同一未收录型号的不同写法合并到同一 `custom_key`、退役或删除后主力清空、没有可用装备的选手不写快照、好友与黑名单可见性。并发写入靠用户行加锁串行化，SQLite 测试不能证明这一点，由第 2 节的 MySQL 端到端用例覆盖事务路径。
- 对局记录：九球追分直接完成、排位确认、超时、斯诺克（不记冲杆）、中八抢局（`EndRound`，选手记录对手的胜局）都会写快照；写入失败不影响结算；7 天内可补录与修改，之后拒绝，只有双方选手能看。
- 后台：非管理员令牌（普通用户、匿名）调用 6 个后台装备接口全部被拒（已用变异验证：去掉校验后这些用例会失败）；归并关联一次最多处理一批，返回 `remaining`。
- 个人数据：导出包含 `gear_items`、`gear_defaults`、`match_gear_snapshots`，注销时三张表一并清理。
- UniApp：装备列表 / 添加 / 详情 / 好友页与对局用杆卡片的挂载测试，身份切换后缓存隔离，退出登录清空装备缓存，页面主题契约。
- 后台：表单校验与请求体构造（含「新建型号默认不选品牌，避免误建球房公用条目」，已用变异验证）。

## 2. MySQL 验证

端到端用例在真实 MySQL 上验证了迁移、预置的球房公杆与球房巧克、`ON DUPLICATE KEY` 的主力更新、`LIKE ... ESCAPE`（关键词 `100%`、`%` 按字面匹配）、行锁事务下的 30 件上限、好友视图、对局完成后的快照记录与修改、后台归并关联（双方历史对局随之显示为正式型号）、个人数据导出与注销。

EXPLAIN 计划（数据量由测试种子构造，型号约 1k、待归并登记约 1 万）：

| 查询 | 索引 | 备注 |
| --- | --- | --- |
| 有启用型号的品牌 | `idx_gear_products_list` | `FirstMatch`，覆盖索引 |
| 单品牌型号分页 | `idx_gear_products_list` | 覆盖索引，无 filesort |
| 整类别型号分页（不选品牌） | `idx_gear_products_list` | 估算 1155 行，`Using filesort`；目录规模小，接受 |
| 好友装备 / 本人装备 | `idx_user_gear_items_user_status` | 本人视图含 filesort，单用户最多 30 行 |
| 用户 / 双方主力 | `uniq_user_gear_defaults` | 覆盖索引 |
| 一场对局的用杆记录 | `uniq_match_gear_snapshots` | 覆盖索引 |
| 待收录分组 | `idx_user_gear_items_pending` | 估算 9780 行，`Using temporary; Using filesort`；仅后台使用，已分页 |
| 归并关联批次 | `idx_user_gear_items_pending` | 覆盖索引，每批 500 |

## 3. 构建产物核对（微信小程序）

- `subPages/gear` 已注册为分包，页面为 `index`、`add`、`item`、`friend`；主包页面中没有装备页。
- 分包体积约 68 KB；按产物文件字节数估算，主包（不含各分包目录）约 0.98 MB，全部文件约 1.91 MB。这是文件字节数估算，不是 `miniprogram-ci` 上传结果，主包上限 2 MB。
- 产物中没有任何装备 PNG / SVG；默认图路径经 `resolveCdnAssetUrl` 解析到 `https://cdn.zhuifen.cn/static/images/gear/...`。
- 产物中 `kekemate` 出现 0 次。

## 4. 视觉自检（仅布局参考，不是验收）

UniApp：H5 构建 + 无头 Chrome 配合**本地 mock 接口**，亮 / 暗两套主题截图检查了装备列表（含主力选择抽屉）、添加（目录型号、未收录型号、选皮头 / 巧克）、装备详情（球杆、冲杆、退役）、好友装备页、对局详情 / 结果页的用杆卡片与修改弹层。检查中修正过：卡片高度不齐、字符中间断行、对局卡片顶部缺边距、「老王 的装备」多余空格。装备卡图片区域实测约 175×177 CSS px（目标不小于 160px）。

管理后台：Vite 开发服务 + 无头 Chrome 配合**本地 mock 接口**（`Authorization: Bearer` 头已带上，mock 不校验），走了一遍：

- `/gear` 重定向到 `/gear/pending`，侧栏「装备目录」分组展开。
- 待收录：打开关联弹窗，未选型号时「关联」按钮禁用；选型号后弹出确认框；点取消不发请求；确认后 `POST /api/admin/gear/pending/link` 的请求体为 `{custom_key, product_id}`，列表刷新。
- 型号管理：新增弹窗品牌默认未选；空名称、未选品牌分别被拦截且不发请求；补全后请求体为 `{category, brand_id, name, cue_style, ...}`。
- 品牌管理：新增品牌请求体正确。
- 过程中控制台没有错误。

局限：H5 不是发布目标，mock 数据不代表真实后端；没有在微信开发者工具或鸿蒙上看过。截图没有留在仓库里。

## 5. 未执行 / 需要人工确认

任务 12.3 已勾选，依据是用户的确认：真机首轮验收时后端还没有部署装备接口（见第 6 节），装备页返回 404；部署后又遇到「型号加载失败」，修复并换到 1.0.2 后用户确认全部场景验收通过。

CDN（任务 6.3）：

- [x] `uniapp/cdn-assets/static/images/gear/` 下 7 张默认图已由用户上传到七牛桶（key 与目录结构一致）。这是用户的确认；我在本机用 curl 访问 `https://cdn.zhuifen.cn/...` 得到 403，已有的段位图同样是 403，说明本机的访问方式受限，不能独立核对线上地址。

微信小程序真机、鸿蒙（以下场景由用户确认已验收通过；用户没有说明具体平台，访问日志里能看到的客户端只有 iPhone 上的微信 8.0.78，鸿蒙是否验过没有记录）：

- [x] 添加目录型号与未收录型号；设置皮头、巧克与主力；球房公杆的添加。
- [x] 完成一场中八对局后，双方的本场用杆；7 天内修改本场用杆；超过 7 天不可改（超过 7 天不可改只在自动化测试里验证，真机无法等 7 天）。
- [x] 好友能查看装备，非好友 / 黑名单不可见；他人看不到自填的品牌与型号名。
- [x] 深色模式下默认图的轮廓（描边外圈在乌木、黑色后把上是否托得住）。
- [x] 图片区域在小程序里的实际尺寸、CDN 图片加载失败时的占位。
- [ ] 鸿蒙上装备页的布局与 `uni.$once / $emit` 选择皮头 / 巧克的交接。用户没有说明是否验过鸿蒙，保持未勾选。

后台：

- [ ] 用真实后端与真实管理员令牌走一遍录入品牌与型号、归并关联。本次只在 MySQL 端到端用例（逻辑层）和 mock 界面上验证过。
- [ ] 用真实 HTTP 请求带普通用户 JWT 调用 `/api/admin/gear/*`，确认被拒。本次只在逻辑层用「用户令牌 / 匿名」上下文验证过拒绝，没有经过路由、中间件和真实 JWT。

## 6. 备注

- 部署（2026-10-26）：
  - 真机验收时装备页显示「资源不存在或已下线」，原因是测试和生产后端都还没有装备接口：`/api/gear/*` 返回 404，已有接口（如 `/api/friend/search`）返回 401。
  - 测试环境：`backend/deploy-dev.sh`，发布 `20261026101435-8bf206a15639-dirty`（先于提交，工作区未提交状态），执行迁移 `20261025115843_add_gear_tables.sql`；服务 active，装备用户端与后台共 6 类路由探测均为 401（路由存在，需要登录）。
  - 生产环境：先提交并推送 `devlop`（`bbbb502` 后端、`186cdac` UniApp、`763926d` 后台、`e8035a1` 文档），预演列出待执行迁移 1 个、线上配置模板无变化，再正式发布 `20261026102155-e8035a1343c9`；脚本先备份数据库，再执行迁移、替换二进制并重启，本机与公网健康检查均为 200，日志无错误行；装备路由在服务器本机探测同为 401。
  - 没有直接查询线上库里预置的球房公杆 / 球房巧克行；这两行由迁移写入，已在临时 MySQL 的端到端用例里核对过，线上迁移执行结果为 OK。
  - 部署后没有用真实登录态走过装备接口，仍需真机重测。
  - 真机重测发现添加装备页「型号加载失败」：生产访问日志里该请求是 `GET /api/gear/catalog/products?category=cue&brand_id=undefined&keyword=undefined&page=1&page_size=20`，返回 400。原因是小程序的 `uni.request` 会把值为 `undefined` 的 GET 参数拼成字符串 `"undefined"`，后端按 `brand_id` 解析整数失败。H5 构建、mock 接口和 Node 测试都没有暴露它，因为它们不经过小程序的参数序列化。修复：`subPages/gear/add.vue` 只带有值的参数；`tests/gear-add-page-mount.test.mjs` 把列表请求参数改成严格相等，并新增选品牌、搜索、取消品牌的参数用例（去掉修复后这两条会失败）。用户构建并上传 1.0.2 后，手机上仍是旧包（微信缓存了旧的开发 / 体验版，彻底关闭并重新进入后才换包），生产访问日志在 11:16:41 起出现 `GET /api/gear/catalog/products?category=cue&page=1&page_size=20` 返回 200，不再带 `undefined`。
- 只有 `admin/src/layout/index.vue` 与 `admin/src/router/index.ts` 是改动已有后台文件；其余后台改动为新增。
- 发现的既有问题（不在本次范围）：后台路由并非所有接口都校验管理员身份。已确认没有管理员校验的写接口有 `update_user_status`、`review_venue`、`update_venue_reward_config`、`update_reputation_config`、`update_llm_config`、`update_member_rights_config`，另有若干读接口。本次新增的 6 个装备后台接口都校验管理员（`requireGearAdmin`）。建议另开变更统一补上。
- 默认图的风格与重新导出方法见 [docs/previews/gear-default-images/README.md](../previews/gear-default-images/README.md)。
