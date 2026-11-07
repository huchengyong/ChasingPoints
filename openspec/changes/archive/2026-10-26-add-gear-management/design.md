## Context

现状（2026-10-25 核对）：

- **没有装备概念。** 最接近的是称号 `user_titles`，有归属、来源和佩戴，但只有一个品类。
- **对局完成的路径。** 共五条：直接结束、排位确认（`finish_action_logic.go`）、确认超时（`finish_request_timeout.go`）、斯诺克认输与局动作（`snooker_action_logic.go`）、回合结束达到赛制（`end_round_logic.go`）。
  - 事务内都走 `settleMatchResultTx`，提交后都调用 `finishMatchPostCommit`（`backend/internal/logic/match/finish_match_logic.go`）。
  - `finishMatchPostCommit` 里已经有信誉、广播、缓存失效、通知等尽力而为的步骤。
- **以后统计要关联的数据已经有了。** `match_participant_results` 每场每位注册选手一行，含结果、比分、最高单杆、完成时间。V2 排位每场在 `rank_change_logs.remark` 写入赛前期望胜率。
- **读取约束。**
  - `bounded-read-queries`：查询有界、批量加载、GET 纯读。
  - `read-api-composition`：可选区块独立失败。
  - `shared-read-caching`：用户派生数据首期直接读数据库。
- **个人数据导出与注销按表列举。** 在 `backend/internal/model/user_data_lifecycle.go` 的 `PersonalDataExportCategories` 与 `DeleteAccountRelationsWithTx` 里。
- **好友可见的先例。** 荣誉墙只对本人或好友开放，用 `FriendModel.AreFriends` 判断（`backend/internal/logic/achievement/get_honor_wall_logic.go`）。
- **图片。**
  - 七牛上传凭证目前只签发头像。
  - 客户端的大图放在 `uniapp/cdn-assets/static/images/` 并上传 CDN；`uniapp/utils/cdn-asset.js` 用路径白名单把它们转成 CDN 地址。
  - `wst-image-mirroring` 规格要求小程序只使用自有图片或包内默认图。
- **后台鉴权。** 管理员令牌使用负数 ID，由 `utils.GetAdminIDFromCtx` 校验。现有后台接口里有一部分没有调用它，本变更不修，但新接口必须调用。
- **迁移。** 最近一次迁移 `20261016100000_create_llm_configs.sql` 没有指定字符集，并且有测试守护，本变更沿用这个做法。
- **客户端。** UniApp 只编译微信小程序与鸿蒙；小程序主包上限 2MB，新页面放分包。
- **默认图已确认。** 默认图和风格说明在 `docs/previews/gear-default-images/`：SVG 源文件、1024px PNG、README。球杆构图为与竖直方向成 25°，已确认。

## Goals / Non-Goals

**Goals:**

- 用户能登记球杆和冲跳装备，设置皮头、巧克和主力装备。
- 每场完成的对局自动记录双方当时的用杆组合；写入失败不影响对局完成。
- 平台目录可以在后台维护，未收录型号能归并后关联。
- 所有读取有界，可见范围由服务端执行，新数据纳入导出与注销。
- 为以后的统计留好原始数据，本期不做统计。

**Non-Goals:**

- 装备、皮头、巧克的胜率或任何统计展示，以及装备排行。
- 用户上传照片和后台图片上传。目录图由运营上传 CDN 后在后台填写路径。
- 原生 iOS 与官网；观战页和公开对局详情的用杆展示。
- 手套、杆盒、加长把等其他配件；把美式杆的前节作为子属性。
- 交易、商城与品牌合作。
- 修复现有后台接口缺少管理员身份校验的问题，另开变更处理。

## Decisions

### D1：数据模型——五张表，用 0 表示“无”

| 表 | 作用 | 关键字段 | 约束与索引 |
| --- | --- | --- | --- |
| `gear_brands` | 品牌 | `name`、`aliases`（逗号分隔的搜索别名）、`sort`、`status` | `uniq(name)` |
| `gear_products` | 型号 | `category`（`cue` / `break_jump` / `tip` / `chalk`）、`brand_id`（0 = 无品牌）、`name`、`house`、`cue_style`（`snooker` / `pool`）、`roles`（`break`、`jump`）、`length_mm`、`weight_g`、`tip_diameter_mm`（可空）、`image_path`、`sort`、`status` | `uniq(category, brand_id, name)`；`idx(category, status, brand_id, sort, id)` |
| `user_gear_items` | 用户的球杆与冲跳装备 | `user_id`、`category`（`cue` / `break_jump`）、`product_id`（0 = 未收录）、`custom_brand_id`、`custom_brand_name`、`custom_model_name`、`custom_key`、`cue_style`、`roles`、`tip_product_id`、`chalk_product_id`、`status`（1 在用 / 2 退役 / 3 已删除） | `idx(user_id, status, id)`；`idx(product_id, custom_key)` |
| `user_gear_defaults` | 主力设置 | `user_id`、`game_type`、`slot`（`cue` / `break`）、`item_id` | `uniq(user_id, game_type, slot)` |
| `match_gear_snapshots` | 对局用杆记录 | `match_id`、`user_id`、`cue_item_id`、`tip_product_id`、`chalk_product_id`、`break_item_id`、`source`（`auto` / `manual`） | `uniq(match_id, user_id)`；`idx(user_id, match_id)` |

- **用 0 表示“无”。** 沿用 `user_titles.source_ref_id` 的写法，不引入可空外键。
- **不建外键约束**，与现有表一致。
- **DDL 不写字符集和排序规则。** 迁移测试检查 `CHARSET` / `CHARACTER SET` / `COLLATE` 不出现。
- **皮头、巧克不单独入表。** 它们只是目录型号，被球杆和对局记录引用，不做用户拥有的独立装备。用户说的是“杆带两个子属性”，单独拥有会多出一套无用的增删界面。

备选方案：一张通用物品表，皮头、巧克也作为用户拥有的物品。没有采用，原因同上。

### D2：对局记录存装备 ID 和当时的皮头、巧克型号

- 球杆与冲杆存**用户装备 ID**，读取时经装备找到型号。
  - 后台把未收录装备关联到正式型号后，历史对局自动显示为该型号。
  - 用户纠正错选的型号时，历史也随之纠正。这正是希望的效果：装备 ID 代表的是那根实物杆。
- 皮头与巧克存**当时的型号 ID**，之后用户换皮头不影响历史。
- **删除用软删除**（`status = 3`），历史记录仍能解析。只有账号注销时才和对局记录一起物理删除。
- 以后做统计时：用 `(match_id, user_id)` 关联 `match_participant_results` 取胜负、赛制与完成时间，再关联 `rank_change_logs` 取期望胜率，算“比预期多赢多少”。本期不建统计表、不加统计索引。

备选方案：快照里同时冻结球杆型号 ID。没有采用：后台关联未收录型号时要连带改写历史，而且球杆型号本来就不会随时间变化。

### D3：写入点放在 `finishMatchPostCommit`，尽力写入且幂等

- 新增共享逻辑 `backend/internal/logic/gear_match_record.go`，入口 `RecordMatchGearSnapshots(svcCtx, match)`，由 `finishMatchPostCommit` 对已完成的对局调用。
- 执行步骤，最多 3 条 SQL，没有任何主力设置时只有 1 条：
  1. 读双方的主力设置：`user_id IN (选手1, 注册的选手2) AND game_type = ?`。
  2. 一次读出引用到的装备，取它们当前的皮头与巧克。
  3. 多行插入，`uniq(match_id, user_id)` 冲突时什么都不做，等价于 `INSERT IGNORE`。
- 写入规则：
  - 冲杆只在九球追分、中式八球、美式九球记录。
  - 只认在用的装备。退役和删除时已经清掉主力设置，这里再校验一次。
- 出错只记日志（带 `match_id`），不改变完成接口的响应，也不重试。
- 竞技读模型重建（`competitive_read_rebuild_service.go`）不调用这段逻辑。

备选方案：

- 写在 `settleMatchResultTx` 事务内。没有采用：结算事务关系到对局结果与排位公平，装备逻辑的缺陷或锁等待不能让对局完成回滚。代价是提交后、写入前进程崩溃，这场就没有记录，按“未记录”显示，用户 7 天内可以补记。
- 走异步队列。没有采用：现在没有可复用的队列，而结算后处理里本来就是这类尽力而为的步骤。

### D4：接口

用户端 `/api/gear`（`jwt: Auth`，当前用户用 `utils.GetUserIDFromCtx` 取得）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/catalog/brands?category=` | 启用、且该类别下有启用型号的品牌 |
| GET | `/catalog/products?category=&brand_id=&keyword=&house=&page=&page_size=` | 分页，`page_size` 上限 50 |
| GET | `/mine` | 本人全部未删除装备，含型号、品牌、皮头、巧克与主力设置 |
| POST | `/item/create` | 登记目录型号或未收录型号 |
| POST | `/item/update` | 修改皮头、巧克，或在用 / 退役状态 |
| POST | `/item/delete` | 软删除，同一事务清除指向它的主力设置 |
| POST | `/default/set` | `game_type + slot + item_id`，`item_id = 0` 表示清除 |
| GET | `/user?user_id=` | 好友视图：在用装备与主力设置，隐藏自填名称 |
| GET | `/match?match_id=` | 双方的本场用杆、本人是否可改、修改截止时间 |
| POST | `/match/update` | 修改或补记本人的本场用杆（7 天内） |

后台 `/api/admin/gear`（`jwt: Auth`，每个 logic 开头都调用 `utils.GetAdminIDFromCtx`）：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET / POST | `/brands`、`/brand/save` | 品牌列表、新增与修改 |
| GET / POST | `/products`、`/product/save` | 型号列表（类别、品牌、关键字、状态过滤，分页）、新增与修改 |
| GET / POST | `/pending`、`/pending/link` | 待收录分组列表（按登记数倒序）、按 `custom_key` 关联到型号 |

- **展示字段在服务端按查看者组装。** 本人的视图带自填名称；其他人的视图里，型号显示「未收录型号」，自填品牌显示「其他品牌」。自填名称不会离开服务端，隐私不依赖客户端。
- **新增字段都在新接口里。** 现有接口与 iOS 不受影响。

### D5：查询次数与访问控制

- `/mine`，固定 4 条 SQL：装备（≤ 30 件）、型号（球杆、冲杆、皮头、巧克一起 `IN`）、品牌（`IN`）、主力设置。
- `/user`：好友判断加上同样的 4 条。查看者不是好友也不是本人时直接拒绝，不再读装备。
- `/match`，固定 5 条 SQL：对局（必须已完成，且查看者是选手 1 或注册的选手 2）、记录（≤ 2 条）、装备（`IN`）、型号（`IN`）、品牌（`IN`）。裁判与观战者拒绝。
- 目录：
  - 品牌列表 1 条 SQL，用 `EXISTS` 过滤出有启用型号的品牌。
  - 型号列表 3 条 SQL：计数、分页、品牌。
  - 关键字用 `LIKE` 匹配品牌名、别名与型号名，带 `LIMIT`。目录是运营维护的小表，规模可控。
- 后台待收录列表：`product_id = 0 AND status < 3` 按 `custom_key` 分组，按数量倒序分页，走 `idx(product_id, custom_key)`。
- 后台关联：每次最多更新 500 条，返回已更新数和剩余数，管理员可以重复执行。
- 热点接口（`/mine`、`/match`）补查询次数测试；新索引用代表性 MySQL `EXPLAIN` 验证。

### D6：未收录型号与归并键

- 用户选目录品牌时记 `custom_brand_id`，选「其他品牌」时记 `custom_brand_name`。型号名记在 `custom_model_name`。
- 球杆还要选杆型，用来决定默认图；冲跳装备至少选一个身份。
- 归并键 `custom_key = 类别 | (b<品牌ID> 或 n<规范化品牌名>) | 规范化型号名`。规范化规则：
  - 全角 ASCII（U+FF01–U+FF5E）和全角空格转半角。
  - 转小写，去掉首尾空格，连续空白合并为一个。
  - 在本地实现，不新增依赖。
- 关联时保留自填字段与归并键，只设置 `product_id`。这样误关联时还能按归并键找回原登记，本期不提供撤销界面。

### D7：修改窗口与校验

- 时间窗口：以对局完成时间加 7 天为截止，完成时间复用 `resolveMatchCompletedAt` 的取值顺序（`completed_at`、`end_time`、`match_time`）。
- 修改接口一次提交球杆、皮头、巧克、冲杆四个字段，记录不存在时新建，`source = manual`。字段校验：
  - 球杆、冲杆必须是本人未删除的装备，类别与身份要符合。
  - 皮头、巧克必须是目录中的对应类别型号（停用的也接受，以便保留原值）。
  - 斯诺克不接受冲杆。
- 客户端换球杆时，用这根杆当前的皮头与巧克预填，用户可以再改。

### D8：客户端（UniApp）

- **新分包 `subPages/gear/`：**
  - `index`：我的装备。按球杆、冲跳、已退役分组；顶部是各球种的主力设置。
  - `add`：添加装备。流程为类别 → 品牌 → 型号；找不到时登记未收录型号；有「球房公杆」快捷入口。`mode=pick` 时同一页面用于选择皮头或巧克，结果通过 `eventChannel` 回传。
  - `item`：单件装备。展示大图、官方规格、皮头与巧克，提供退役、恢复、删除。找不到皮头或巧克时，引导到现有的「帮助与反馈」页。
  - `friend`：TA 的装备，只读。
- **其他模块：**
  - `api/gear.js`：门面。
  - `store/gear.js`：本人装备缓存。绑定 `userId + authGeneration`，并发请求合并为一次，60 秒有效，写操作后标脏。好友装备不缓存。
  - `utils/gear-display.js`：纯函数，包括默认图选择、目录图路径校验、规格格式化（重量显示克与盎司）、身份标签、修改截止判断。
  - `components/matchGearCard.vue`：「本场用杆」卡片和修改弹层，供对局详情与结果页复用。只在已完成、且查看者是选手时请求 `/api/gear/match`，失败时显示可重试状态，不影响页面其他部分。
- **入口：** 「我的」服务列表加「装备」；好友主页在荣誉墙入口旁加「TA 的装备」。
- **默认图：**
  - 从 SVG 导出 512×512 PNG，放到 `uniapp/cdn-assets/static/images/gear/`，文件名与 `docs/previews/gear-default-images/` 一致。
  - `CDN_ASSET_PATH_PATTERN` 增加 `gear` 目录。
  - 发布前上传 CDN，需要确认操作授权。
  - 目录图同样走 `/static/images/gear/` 路径，不在白名单内的一律回退默认图。
- **球杆图很细长。** 装备页的图片位置不小于 160px；对局卡片里的小图只用来识别类别。

### D9：管理后台

- 页面放在 `admin/src/views/gear/`：
  - `brands.vue`：品牌。
  - `products.vue`：型号。表单字段随类别切换：球杆显示杆型，冲跳显示身份。图片路径按 `/static/images/gear/*.png|webp` 校验。
  - `pending.vue`：待收录。展示分组与数量；「关联到型号」时只能选同类别型号，提交前确认弹窗显示将要更新的登记数。
- 接口门面 `src/api/gear.ts`；路由加 `/gear` 一组，菜单名为「装备目录」。

### D10：默认图与风格规范

- 风格以 `docs/previews/gear-default-images/README.md` 为准：
  - 透明背景，球杆与竖直方向成 25°。
  - 硬边分层上色，光源在左上，暖深棕描边，外加一圈浅色外缘以适配深色模式。
  - 杆身风格化加粗。
- `DESIGN.md` 新增简短的「装备图」条目，链接到上面的 README，不在两处重复维护细则。

### D11：迁移与预置数据

- 一个 goose 迁移文件：建五张表，用 `INSERT IGNORE` 预置「球房公杆」（`cue`，`house = 1`）和「球房巧克」（`chalk`，`house = 1`）。Down 删除五张表。
- 迁移测试检查：表与关键字段存在、没有字符集与排序规则、包含预置数据。

### D12：个人数据导出与注销

- 导出类别增加 `gear_items`、`gear_defaults`、`match_gear_snapshots`，只导出本人的数据。
- 注销时按 `user_id` 删除这三张表中的本人数据。对手在同一对局的记录不受影响。

## Risks / Trade-offs

- [提交后、写入前进程崩溃，这场没有记录] → 显示「未记录」，选手 7 天内可以补记。
- [用户不设主力，大部分对局没有记录，攒不到数据] → 装备页突出主力设置；本场用杆卡片在未记录时引导设置主力或补记。
- [目录覆盖不全，大量未收录登记] → 待收录列表按登记数倒序，运营优先补；关联后历史自动更新。
- [自填名称是用户输入] → 只有本人与后台可见，不对外展示，因此不需要内容安全审核；后台按纯文本渲染。
- [管理员误关联] → 提交前确认数量；保留自填字段与归并键，可按归并键找回。本期不提供撤销界面。
- [对局完成多出一次写入] → 最多 3 条 SQL，只在完成时执行，失败不重试，不在结算事务内。
- [球杆细长，小尺寸看不清] → 装备页图片位置不小于 160px；对局卡片的小图只作类别识别。
- [好友关系解除后，对方装备页残留旧数据] → 好友装备不缓存，每次进入都回源，服务端按当前好友关系判断。
- [目录图版权] → 只用平台生成的风格化图，版权由产品负责人处理（已确认）。

## Migration Plan

1. 确认目标数据库后执行迁移，再部署后端。新接口在客户端发布前不会被调用。
2. 默认图上传 CDN，并核对线上地址可以访问。
3. 运营在后台录入第一批品牌与型号。球房公杆、球房巧克已由迁移预置。
4. 小程序与鸿蒙分别构建发布：小程序按既有流程上传开发版本，由用户自行提审。

回滚：

- 客户端发版去掉入口即可停用。
- 后端回滚到旧版本后不再写入记录，新表保留不动。
- 迁移的 Down 会删除装备数据，只在确认放弃这批数据时执行。

## Open Questions

- 本期不讨论：将来如果出现非好友可见的个人主页，装备是否对外公开。
- 冲杆与开球结果（炸清、大金、小金）的统计口径，留给以后的统计变更。
- 目录图的命名约定（例如按型号 ID 命名）由运营在出图时确定；后台只校验目录与扩展名。
