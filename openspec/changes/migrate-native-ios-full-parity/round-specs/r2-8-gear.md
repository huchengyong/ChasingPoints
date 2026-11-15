# R2.8：装备（我的装备、添加装备、装备详情、TA 的装备）

## 本轮要读的小程序源码

- `uniapp/subPages/gear/index.vue`、`add.vue`、`item.vue`、`friend.vue` 及各自样式；组件 `uniapp/components/gearImage.vue`、`gearItemCard.vue`
- `uniapp/utils/gear-display.js`、`gear-form.js`、`gear-cache.js`（了解缓存与失效点）、`uniapp/store/gear.js`；测试 `gear-display.test.mjs`、`gear-form.test.mjs`
- `uniapp/api/gear.js`；规格 `openspec/specs/gear-catalog/spec.md`、`openspec/specs/user-gear/spec.md`（业务规则的权威描述）
- 契约：`backend/chasing_points.api` 的 `/api/gear/*`（品牌 / 型号目录、我的装备、增删改、设默认、他人装备）及嵌套类型；实际返回读 `backend/internal/logic/gear/` 确认。
- 本场装备（`/api/gear/match*`、`matchGearCard.vue`、`gear-match.js`）属于对局模块，本轮不做。

## 任务

1. **规则迁移**（`iOS/iOS/Features/Gear/`）：`gear-display.js`、`gear-form.js` 的全部导出逐条迁移并搬运测试。
2. **共享组件**：`GearImageView`（对应 `gearImage.vue`：装备图走 CDN，亮暗主题下都要托得住深色杆身，占位与失败态照搬）、`GearItemCard`（对应 `gearItemCard.vue`）。
3. **我的装备**：`.gearIndex` → `GearIndexView`，标题「我的装备」。分类、默认装备标记、空态、添加入口以 `index.vue` 为准。
4. **添加装备**：新增 `case gearAdd(category: String, pickToken: String?)`（参数以小程序 `add.vue` 的 onLoad 为准，`mode=pick` 的选择模式也要支持），标题「添加装备」。品牌 → 型号的选择流程、搜索、自定义填写（待审核目录项）、表单校验与提交以 `add.vue` 和 `gear-form.js` 为准。
5. **装备详情**：新增 `case gearItem(id: Int64)`，标题「装备详情」。查看、编辑、设为默认、删除（确认弹窗文案照搬）、更换型号（跳到选择模式的添加页并带回结果——小程序用 token 交接，原生端用回调或共享的页面模型都行，但行为要一致）以 `item.vue` 为准。
6. **TA 的装备**：新增 `case gearFriend(userId: Int64)`，标题「TA 的装备」，以 `friend.vue` 为准（对方隐藏 / 无装备时的文案照搬）。
7. 小程序的 `store/gear.js` + `gear-cache.js` 是跨页面缓存：原生端用一个 `GearStore`（`@MainActor ObservableObject`，按身份隔离，放进环境或由 `GearIndexView` 持有并传给子页面）承载「我的装备」列表，增删改 / 设默认成功后更新它，使返回列表页时不必整页重读；身份变化清空。

## 测试

搬运 JS 用例；接口门面；`GearStore`（读取、增删改后的本地更新、身份变化清空、迟到结果丢弃）；添加 / 编辑表单模型（校验文案、提交参数、失败保留表单、提交中不可重复提交）；快照：`gear-index`、`gear-index-empty`、`gear-add-brands`、`gear-add-form`、`gear-item`、`gear-friend`、`gear-friend-empty`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/News/**`、`Features/Match/**`、`Features/Home/**`、`Features/Messages/**`、`Features/Friends/**`、`iOS/iOS/App/MainTabView.swift` 及其测试，本轮不要碰这些文件。
