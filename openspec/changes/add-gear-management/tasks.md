## 1. 接口契约与迁移

- [x] 1.1 修改 `backend/chasing_points.api`，修改后立即运行 goctl：
  - 新增用户端 `/api/gear` 接口组（`jwt: Auth`）：`catalog/brands`、`catalog/products`、`mine`、`item/create`、`item/update`、`item/delete`、`default/set`、`user`、`match`、`match/update`，以及请求与响应类型。
  - 新增后台 `/api/admin/gear` 接口组：`brands`、`brand/save`、`products`、`product/save`、`pending`、`pending/link`。
- [x] 1.2 检查 goctl 生成结果：新 logic 落在 `internal/logic/gear/` 与 `internal/logic/admin/`，生成文件没有手改，新 handler 空壳都由后续任务覆盖。
- [x] 1.3 在 `backend/` 用 `./goose.sh create add_gear_tables` 新建迁移：
  - 建五张表及其索引：`gear_brands`、`gear_products`、`user_gear_items`、`user_gear_defaults`、`match_gear_snapshots`。
  - 用 `INSERT IGNORE` 预置「球房公杆」与「球房巧克」。
  - Down 删除五张表。
  - DDL 不指定字符集与排序规则。
- [x] 1.4 在 `migrations/migration_files_test.go` 增加迁移测试：五张表、关键字段、唯一键与预置数据都存在，且不含 `CHARSET`、`CHARACTER SET`、`COLLATE`。
- [x] 1.5 新增 Gorm 模型 `internal/model/gear.go` 和测试 schema helper `internal/testsupport/gear_schema.go`，并在 `ServiceContext` 注入新模型。模型 tag 不写字符集，生产构造函数不调用 `AutoMigrate`。

## 2. 后端目录

- [x] 2.1 在 model 层实现目录查询：
  - 品牌列表：只返回启用、且在该类别下有启用型号的品牌，用 `EXISTS` 过滤。
  - 型号列表：按类别、品牌、关键字（品牌名、别名、型号名）、状态和 `house` 在数据库内分页，按排序值与 ID 稳定排序。
  - 按 ID 批量读取型号与品牌。
- [x] 2.2 实现 `GET /api/gear/catalog/brands` 与 `GET /api/gear/catalog/products`：只返回启用条目，`page_size` 上限 50。补测试：停用条目不出现、别名能搜到、分页稳定、查询次数固定。
- [x] 2.3 实现后台品牌与型号的列表和保存接口，每个 logic 先调用 `utils.GetAdminIDFromCtx`。保存时校验：
  - 非球房的球杆必须有杆型。
  - 冲跳型号至少一个身份。
  - 皮头、巧克不带杆型和身份。
  - 图片路径只接受 `/static/images/gear/` 下的 png 或 webp。
  - 规格不能为负。
  - 唯一键冲突时给出明确提示。

  补测试：普通用户令牌和匿名请求被拒，各校验分支都覆盖。
- [x] 2.4 实现未收录型号的归并与关联：
  - `custom_key` 规范化纯函数（全角转半角、转小写、去首尾空格、合并空白），配表驱动测试。
  - 后台 `GET /pending`：按登记数倒序分组分页。
  - 后台 `POST /pending/link`：校验类别一致，每次最多更新 500 条，返回已更新数与剩余数，保留自填字段与归并键。

  补测试：大小写、空白不同的两条登记归为一组；类别不一致被拒；关联后，历史对局用杆记录显示为该型号。

## 3. 后端用户装备与主力设置

- [x] 3.1 实现 `GET /api/gear/mine`：固定 4 条 SQL 返回本人未删除装备、型号（含皮头与巧克）、品牌和主力设置。补查询次数测试：1 件和 30 件的 SQL 数相同。
- [x] 3.2 实现 `POST /api/gear/item/create`，支持两种登记：
  - 目录型号：必须启用，类别一致。
  - 未收录型号：品牌或其他品牌名不超过 32 字，型号名不超过 64 字；球杆要有杆型，冲跳要有身份。

  每人未删除装备不超过 30 件。补测试覆盖两种登记、各校验分支和上限。
- [x] 3.3 实现 `POST /api/gear/item/update`：
  - 皮头、巧克只对球杆（含公杆）有效，取值为启用的对应类别型号或 0。
  - 在用和退役可以互相切换；退役时在同一事务里清除指向它的主力设置。
  - 只能改本人的装备。

  补测试。
- [x] 3.4 实现 `POST /api/gear/item/delete`：软删除，并在同一事务里清除主力设置。补测试：删除后 `mine` 与 `user` 不再返回这件装备，历史对局记录仍能解析。
- [x] 3.5 实现 `POST /api/gear/default/set`：
  - 球种取值 1–4。
  - `cue` 槽位只能设本人的在用球杆。
  - `break` 槽位只用于球种 2、3、4，只能设本人带 `break` 身份的在用冲跳装备。
  - `item_id = 0` 表示清除。

  补测试：把只有 `jump` 身份的装备设为主力冲杆被拒；斯诺克设主力冲杆被拒。
- [x] 3.6 实现 `GET /api/gear/user`：
  - 只允许本人或好友（`FriendModel.AreFriends`）查看，只返回在用装备与主力设置。
  - 非好友直接拒绝，不读取装备。

  补测试。
- [x] 3.7 实现按查看者组装展示字段的纯函数，并补测试：
  - 本人视图带自填名称。
  - 他人视图显示「未收录型号」「其他品牌」，自填名称不出服务端。
  - 球房条目不显示品牌。

## 4. 后端对局用杆记录

- [x] 4.1 在 `internal/logic/gear_match_record.go` 实现 `RecordMatchGearSnapshots`，共 3 条 SQL：
  1. 一次读出双方的主力设置。
  2. 一次读出引用到的装备。
  3. 多行插入，唯一键冲突时什么都不做。

  规则：冲杆只在球种 2、3、4 记录；只认在用装备；没有主力设置不写；非注册对手不写。

  补单测：重复调用仍只有一条；不覆盖手动记录；斯诺克不含冲杆；记录的是当时的皮头与巧克。
- [x] 4.2 在 `finishMatchPostCommit` 里对已完成对局调用 `RecordMatchGearSnapshots`，出错只记日志。补集成测试：
  - 直接结束、排位确认、确认超时三条路径各写入记录。
  - 写入失败时完成接口仍然成功。
  - 斯诺克认输与回合结束两条路径确认经过同一个后处理。
- [x] 4.3 实现 `GET /api/gear/match`：
  - 只对已完成对局的选手开放；裁判、观战者和其他用户拒绝；进行中的对局返回不可用。
  - 固定 5 条 SQL 返回双方记录、展示字段、`editable` 与 `edit_deadline`，不写任何数据。

  补访问控制测试与查询次数测试。
- [x] 4.4 实现 `POST /api/gear/match/update`：
  - 截止时间为完成时间加 7 天，完成时间复用 `resolveMatchCompletedAt`。
  - 只能改本人的记录。
  - 球杆、冲杆必须是本人未删除、类别与身份正确的装备。
  - 皮头、巧克必须是对应类别的型号（含已停用）或 0。
  - 斯诺克不接受冲杆。
  - 记录不存在时新建，`source = manual`。

  补测试：补记、超期、改对手记录、类别错误。
- [x] 4.5 补测试或代码检查，确认竞技读模型重建路径不读取、不改写对局用杆记录。

## 5. 后端个人数据导出与注销

- [x] 5.1 `PersonalDataExportCategories` 增加 `gear_items`、`gear_defaults`、`match_gear_snapshots`，只导出本人数据（含未收录名称）。补导出测试。
- [x] 5.2 `DeleteAccountRelationsWithTx` 删除这三张表中的本人数据。补注销测试：对手在同一对局的记录保持不变。
- [x] 5.3 在 `backend/` 运行 `go test ./...` 与 `go build .`。在隔离的 MySQL 库上用代表性数据对新索引做 `EXPLAIN`，结果写入验收记录。

## 6. 默认图资源

- [x] 6.1 把 `docs/previews/gear-default-images/` 中 7 张默认图的 SVG 导出为 512×512 透明 PNG，放到 `uniapp/cdn-assets/static/images/gear/`，文件名不变。核对背景透明、单张体积合理。
- [x] 6.2 `uniapp/utils/cdn-asset.js` 的 `CDN_ASSET_PATH_PATTERN` 增加 `gear` 目录，补测试。
- [ ] 6.3 确认授权后把装备默认图上传 CDN，核对线上地址可以访问，记录在验收文档中。

## 7. UniApp 基础模块

- [x] 7.1 新增 `uniapp/api/gear.js`，覆盖全部 `/api/gear` 接口，遵循 `api/AGENTS.md`。
- [x] 7.2 新增 `uniapp/utils/gear-display.js` 纯函数，并补单测：
  - 默认图选择：公杆、斯诺克球杆、美式球杆、冲杆、跳杆、皮头、巧克。
  - 目录图路径校验，不合规时回退默认图。
  - 规格格式化：未提供的不显示，重量同时显示克与盎司。
  - 身份标签。
  - 修改截止判断。
- [x] 7.3 新增 `uniapp/store/gear.js`：本人装备缓存绑定 `userId + authGeneration`，并发请求合并为一次，60 秒有效，写操作后标脏。补测试：首次进入、重复 `onShow`、并发请求、退出与切号、迟到响应。

## 8. UniApp 装备页面

- [x] 8.1 在 `uniapp/pages.json` 注册 `subPages/gear` 分包（`index`、`add`、`item`、`friend`），不放进主包。
- [x] 8.2 实现 `subPages/gear/index`：
  - 主力设置：四个球种各一根主力球杆；九球追分、中八、美九各一根主力冲杆。
  - 球杆、冲跳、已退役三个分组。
  - 空状态引导添加装备和设置主力。
- [x] 8.3 实现 `subPages/gear/add`：
  - 按类别 → 品牌 → 型号选择，支持搜索和分页。
  - 球房公杆快捷入口。
  - 未收录登记表单：品牌或其他品牌、型号名、杆型或身份。
  - `mode=pick` 时用于选择皮头或巧克，结果通过 `eventChannel` 回传。
  - 达到上限时给出提示。
- [x] 8.4 实现 `subPages/gear/item`：大图、官方规格、皮头与巧克设置，以及退役、恢复、删除（删除需二次确认）。找不到皮头或巧克时引导到现有的帮助与反馈页。
- [x] 8.5 实现 `subPages/gear/friend`：好友装备只读页，不缓存。非好友或请求失败时展示对应状态。
- [x] 8.6 加入口：「我的」服务列表加「装备」；好友主页在荣誉墙入口旁加「TA 的装备」。
- [x] 8.7 页面按 `DESIGN.md` 实现亮暗主题与卡片样式，装备图位置不小于 160px。按现有 Vue SFC 测试方式补页面挂载测试。

## 9. UniApp 本场用杆

- [x] 9.1 新增 `uniapp/components/matchGearCard.vue`：
  - 展示双方的本场用杆，包括「未记录」和「未收录型号」两种状态。
  - 本人可改时提供修改弹层：球杆、皮头、巧克、冲杆；换杆时用这根杆当前的皮头和巧克预填。
- [x] 9.2 对局详情页与结果页接入卡片：
  - 只在对局已完成、且查看者是选手时请求 `/api/gear/match`。
  - 请求失败只影响卡片，可以重试。
  - 不新增重复读取或 WS 订阅。

  补请求图测试：已完成、进行中、裁判、观战四种情况；失败降级；切换账号后丢弃迟到响应。

## 10. 管理后台

- [x] 10.1 新增 `admin/src/api/gear.ts`：类型声明与请求门面。
- [x] 10.2 新增 `admin/src/views/gear/` 下三个页面：`brands.vue`、`products.vue`、`pending.vue`。
  - 型号表单的字段随类别切换。
  - 图片路径的前端校验与后端一致。
  - 关联前弹窗确认，显示将要更新的登记数。
- [x] 10.3 路由与菜单新增「装备目录」组，核对合规开关逻辑。
- [x] 10.4 补 admin 测试（参数整形或校验工具），在 `admin/` 运行 `node --experimental-strip-types --test tests/*.test.mjs` 与 `npm run build`。

## 11. 文档

- [x] 11.1 `DESIGN.md` 增加「装备图」条目，链接到 `docs/previews/gear-default-images/README.md`。
- [x] 11.2 `uniapp/AGENTS.md` 的 CDN 大图说明里加入装备图，并同步镜像 `GEMINI.md`，用 `cmp AGENTS.md GEMINI.md` 核对。
- [x] 11.3 检查文档链接，运行 `git diff --check`。

## 12. 验证与验收

- [x] 12.1 在 `uniapp/` 运行 `node --test tests/*.test.mjs`。
- [x] 12.2 微信小程序生产构建：核对主包体积、分包注册正确，以及默认图走 CDN。
- [ ] 12.3 在微信小程序（开发者工具与真机）和鸿蒙上验收以下场景：
  - 添加目录型号与未收录型号。
  - 设置皮头、巧克与主力。
  - 完成一场中八对局后，双方的本场用杆。
  - 7 天内修改本场用杆。
  - 好友能查看，非好友不可见。
  - 深色模式下默认图的轮廓。

  结果写入 `docs/testing/add-gear-management-acceptance.md`，只记录实际执行过的平台。
- [x] 12.4 后台验收：录入品牌与型号，归并并关联未收录登记，确认普通用户令牌调用后台装备接口被拒。
