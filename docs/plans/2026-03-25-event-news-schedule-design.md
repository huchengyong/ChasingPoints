# 赛事情报赛程模式设计

## 背景

当前“赛事情报”模块的数据模型是单条记录模式，一条数据同时承载赛事标题、阶段说明、赛果摘要、时间地点等信息。这个结构更像资讯流，不适合表达“一个赛事包含多个阶段赛程”的产品语义。

现有问题主要有三点：

- 列表页展示的是平铺事件，用户无法以“赛事”为单位浏览内容。
- 详情页展示的是单条记录详情，无法在同一页中理解完整赛事进程。
- 管理后台每录入一个阶段都要新建一条记录，运营上难以维护，数据语义也不稳定。

本次改造目标是将“赛事情报”重构为真正的赛程模式：

- 列表页先展示赛事父节点
- 点击详情后查看该赛事的阶段赛程
- 赛程粒度只到“阶段”，不下钻到具体对阵
- 旧 `event_news` 数据直接废弃，不做兼容迁移

## 目标

- 将赛事情报从“单条资讯记录”升级为“赛事 + 阶段赛程”模式。
- 用户端列表页按赛事维度浏览，而不是按阶段平铺浏览。
- 用户端详情页在一个赛事详情中看到完整阶段进展。
- 管理后台围绕“赛事主信息”维护内容，并在赛事下管理多个阶段。
- 保留现有首页焦点赛事能力，但底层数据改为赛事主信息。

## 非目标

首期明确不做以下内容：

- 具体对阵 / 每场比赛级别的赛程
- 签表、选手名单、奖金信息
- 自动归并旧 `event_news` 数据
- 富文本 CMS、图文时间线
- 与现有 `tournament` 用户创建赛事能力的合并

## 产品方案

### 用户端列表页

列表页展示“赛事卡片”，每张卡片代表一个完整赛事。

卡片展示信息：

- 赛事标题
- 球种
- 赛事状态
- 举办时间
- 地点
- 当前阶段摘要
- 最新赛果摘要
- 阶段总数

用户不再在列表页看到“资格赛 / 32 强 / 16 强”这些分散条目，而是看到“2025 世界锦标赛”这一条赛事入口。

### 用户端详情页

详情页分为两部分：

- 上半部分展示赛事主信息：标题、摘要、状态、时间、地点、来源
- 下半部分展示阶段赛程列表：按顺序展示资格赛、32 强、16 强、8 强、半决赛、决赛等

阶段赛程项展示信息：

- 阶段名称
- 阶段排序
- 阶段开始时间 / 结束时间
- 阶段状态
- 阶段赛果摘要

### 管理后台

后台的管理方式改成“两层”：

- 先创建赛事主信息
- 再进入赛事编辑态维护多个阶段

后台列表页显示赛事主列表，不再显示阶段平铺列表。

后台编辑弹窗或编辑区域分为两个模块：

- 赛事基础信息
- 阶段赛程管理

阶段管理支持：

- 新增阶段
- 编辑阶段
- 删除阶段
- 按 `stage_order` 排序展示

## 数据模型

### 方案选择

本次采用双表模式，而不是单表父子结构。

原因：

- 赛事主信息和阶段赛程语义明显不同
- 用户端列表和详情的查询模式天然分离
- 管理后台表单结构更清晰
- 未来若扩展签表链接、奖金、阶段扩展字段，也更容易演进

### 赛事主表

建议表名：`event_news_events`

字段建议：

- `id`
- `title`
- `game_type`
- `source_type`
- `source_name`
- `source_url`
- `cover_image`
- `summary`
- `content`
- `country`
- `city`
- `venue`
- `start_time`
- `end_time`
- `status`
- `featured`
- `sort_time`
- `published`
- `published_at`
- `created_at`
- `updated_at`
- `deleted_at`

字段语义说明：

- `status` 表示整个赛事状态，不表示单个阶段状态
- `start_time` / `end_time` 表示赛事整体周期
- `featured` 和 `published` 仍由赛事主表控制

### 阶段表

建议表名：`event_news_stages`

字段建议：

- `id`
- `event_id`
- `stage_name`
- `stage_order`
- `start_time`
- `end_time`
- `status`
- `result_text`
- `sort_time`
- `created_at`
- `updated_at`
- `deleted_at`

字段语义说明：

- `event_id` 关联赛事主表
- `stage_name` 表示资格赛、32 强、16 强等阶段名称
- `stage_order` 用于稳定排序，不能依赖文案排序
- `status` 表示该阶段状态
- `result_text` 表示该阶段的赛果摘要

## 接口设计

### 公开接口

保持前缀 `/api/event-news`，但返回结构改成赛程模式。

- `GET /list`
  - 返回赛事主列表
  - 每个赛事项包含轻量阶段摘要字段，例如：
    - `current_stage_text`
    - `latest_result_text`
    - `stage_count`

- `GET /detail`
  - 入参改为 `event_id`
  - 返回：
    - `event`：赛事主信息
    - `stages`：阶段赛程列表

- `GET /featured`
  - 返回焦点赛事主信息
  - 同时附带当前阶段摘要，用于首页焦点卡片展示

### 管理接口

保持前缀 `/api/admin/event-news`。

赛事主信息接口：

- `GET /list`
- `POST /create`
- `POST /update`
- `POST /publish`
- `POST /delete`

阶段赛程接口：

- `POST /stage/create`
- `POST /stage/update`
- `POST /stage/delete`

如需后台编辑时回显完整阶段列表，可在赛事详情接口或后台列表接口中额外提供阶段数据。

## 页面改造范围

### 后端

- 替换当前单表 `event_news` 模型
- 新增赛事主表和阶段表 migration
- 调整 `chasing_points.api` 的 `event-news` 类型与接口
- 补充模型层、logic 层、测试用例

### 管理后台

- [`admin/src/views/event-news/index.vue`](/Users/wisesearch/Projects/ChasingPoints/admin/src/views/event-news/index.vue) 改成赛事主列表
- [`admin/src/api/event-news.ts`](/Users/wisesearch/Projects/ChasingPoints/admin/src/api/event-news.ts) 更新类型与接口
- 编辑弹窗新增阶段赛程管理区域

### 用户端

- [`app/subPages/tournament/index.vue`](/Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament/index.vue) 改成赛事列表
- [`app/subPages/tournament/detail.vue`](/Users/wisesearch/Projects/ChasingPoints/app/subPages/tournament/detail.vue) 改成赛事详情 + 阶段赛程展示
- [`app/api/event-news.js`](/Users/wisesearch/Projects/ChasingPoints/app/api/event-news.js) 更新入参与返回语义
- 首页焦点赛事映射逻辑同步调整

## 数据迁移策略

旧 `event_news` 数据不保留，直接废弃。

迁移原则：

- 删除旧表 `event_news`
- 新建 `event_news_events` 和 `event_news_stages`
- 不做标题归并、不做自动迁移、不保留兼容接口

这能显著降低本次结构改造的复杂度，也避免错误归并造成脏数据。

## 校验规则

赛事主信息至少要求：

- `title`
- `game_type`
- `start_time` 或 `sort_time`

阶段信息至少要求：

- `event_id`
- `stage_name`
- `stage_order`

可选增强规则：

- 同一赛事下 `stage_order` 不允许重复
- 阶段时间可为空，但如果填写则 `end_time` 不能早于 `start_time`

## 测试策略

后端至少覆盖以下场景：

- 创建赛事
- 更新赛事
- 发布 / 下线赛事
- 删除赛事时级联删除阶段
- 创建阶段
- 更新阶段
- 删除阶段
- 详情接口正确聚合赛事与阶段
- 列表接口正确返回阶段摘要

前端至少验证：

- 管理后台可维护赛事和阶段
- 用户端列表只展示赛事
- 用户端详情能按顺序展示阶段赛程
- 首页焦点赛事不受结构改造影响

## 风险与取舍

- 风险一：接口 shape 会整体变化，前后端必须同步发布。
- 风险二：旧数据直接废弃后，发布前需要准备新的赛事种子数据或后台录入内容。
- 风险三：如果后续要扩展到具体场次，阶段表仍需继续拆分子表，但当前双表模式已经为后续扩展预留了合理边界。

## 推荐结论

本次采用“赛事主表 + 阶段表”的双表赛程模式，彻底替换当前单条资讯记录模式。

这样改造后，用户看到的是“赛事”，不是散乱的阶段记录；运营维护的是“赛事及其阶段”，不是不断新建重复标题的单条资讯；产品语义和技术结构将首次真正对齐。
