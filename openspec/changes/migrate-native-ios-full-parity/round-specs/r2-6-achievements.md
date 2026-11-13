# R2.6：荣誉墙、成就详情

## 本轮要读的小程序源码

- `uniapp/subPages/achievement/index.vue`（945 行，+ 同目录 scss）、`detail.vue`（+ scss）
- `uniapp/utils/honor-wall.js`、`achievement-page.js`、`achievement-detail-cache.js` 及测试 `honor-wall.test.mjs`、`achievement-page.test.mjs`
- `uniapp/api/achievement.js`；页面里对 `api/notification.js`、`store/notification.js` 的用法（换季 / 成就通知的已读处理）
- 契约：`backend/chasing_points.api` 的 `/api/achievement/*`（`GetHonorWallReq/Resp`、`GetAchievementListReq/Resp`、`GetAchievementDetailReq/Resp`、`GetUserTitlesResp`、`EquipTitleReq` 等）及其嵌套类型；空数据 / 未解锁时的实际返回读 `backend/internal/logic/achievement/` 确认。

## 任务

1. **规则迁移**（`iOS/iOS/Features/Achievement/`）：`honor-wall.js`、`achievement-page.js` 的全部导出逐条迁移并搬运测试（`buildHonorWallUrl` 这类生成小程序路径的函数，迁成返回路由参数的函数；`presentLatestSeasonRollover` 依赖弹窗和通知 store，迁移其中的纯判定 / 文案部分，弹窗展示本轮不做，在交付说明里列出）。`achievement-detail-cache.js` 是小程序的跨页面缓存，不迁。
2. **荣誉墙**：`.achievementWall(gameType:)` → `HonorWallView`，标题「荣誉墙」。页面的全部区块、球种切换、称号佩戴（`POST /api/achievement/title/equip`，成功 / 失败提示照搬）、成就分组与进度、空态、失败态以 `index.vue` 为准。成就 / 段位 / 称号图标用 `RemoteImage(url: CDNAsset.url(for:))`。点成就 → `.achievementDetail(achievementId:)`（如果小程序跳转还带其他参数，给路由增加关联值）。
3. **成就详情**：`.achievementDetail` → `AchievementDetailView`，标题「成就详情」。内容以 `detail.vue` 为准。
4. 页面里与通知已读相关的副作用（进入页面把某类通知标记已读、角标减一）：只做「调用已读接口」这一步（`POST /api/notification/read` 等，以页面实际调用为准），角标联动等消息模块迁移后再接，在交付说明里列出。

## 测试

搬运 JS 用例；接口门面；页面模型（首次加载、切换球种、佩戴称号成功 / 失败回滚、空、失败重试、未登录、身份变化）；快照：`honor-wall`（有称号、有已解锁和未解锁成就）、`honor-wall-empty`、`honor-wall-error`、`achievement-detail-unlocked`、`achievement-detail-locked`。

## 并行开发提醒

其他开发线正在修改 `iOS/iOS/Features/News/**`、`Features/Match/**`、`Features/Home/**`、`Features/Messages/**`、`Features/Friends/**`、`iOS/iOS/App/MainTabView.swift` 及其测试，本轮不要碰这些文件。
