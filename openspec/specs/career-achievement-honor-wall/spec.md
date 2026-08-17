# Career Achievement Honor Wall

## Purpose

定义生涯成就荣誉墙的元数据、分组展示、完成度摘要、好友隐私与详情反馈。

## Requirements

### Requirement: 成就接口提供球种与奖励称号元数据
所有返回 `AchievementDef` 的成就接口 SHALL 在保持既有字段兼容的同时返回 `game_type` 和可为空的 `reward_title_name`，使客户端无需根据 Key 猜测球种或奖励。

#### Scenario: 获取生涯成就列表
- **WHEN** 已登录用户请求成就列表或荣誉墙
- **THEN** 每条成就 SHALL 包含 `game_type`
- **AND** 带奖励称号的成就 SHALL 返回对应 `reward_title_name`
- **AND** 无称号奖励的成就 SHALL 返回空值而不是虚构奖励

#### Scenario: 旧客户端字段保持兼容
- **WHEN** 后端部署新增字段后旧客户端继续读取原有 `id/key/name/description/icon/category/threshold/progress/unlocked/unlocked_at`
- **THEN** 原有字段语义和数据类型 MUST 保持不变

### Requirement: 荣誉墙分别统计通用与球种专精完成度
荣誉墙摘要 SHALL 保留全部生涯成就的兼容总数，并新增当前所选球种的拆分摘要：`universal_unlocked`、`universal_total`、`specialty_game_type`、`specialty_unlocked`、`specialty_total`。

#### Scenario: 统计通用成就完成度
- **WHEN** 用户请求自己的荣誉墙
- **THEN** `universal_total` SHALL 等于所有启用且 `game_type = 0` 的定义数量
- **AND** `universal_unlocked` SHALL 等于该用户已解锁的对应数量

#### Scenario: 统计当前球种专精完成度
- **WHEN** 用户以 `game_type = 3` 请求荣誉墙
- **THEN** `specialty_game_type` SHALL 为 3
- **AND** `specialty_total` SHALL 只统计启用的中式八球专属定义
- **AND** `specialty_unlocked` SHALL 只统计用户已解锁的中式八球专属定义

#### Scenario: 保留全目录兼容统计
- **WHEN** 荣誉墙返回拆分摘要
- **THEN** 原有 `career_total` SHALL 继续表示全部启用生涯定义数量
- **AND** 原有 `career_unlocked` SHALL 继续表示用户在全部启用定义中的已解锁数量

### Requirement: 生涯页采用通用里程碑与球种绝技分区
本人荣誉墙 SHALL 保持“生涯成就、当前赛季、历届荣誉”一级结构，并在生涯成就中先展示通用里程碑，再展示可切换球种的专属绝技。

#### Scenario: 通用里程碑按固定顺序分组
- **WHEN** 用户查看生涯成就
- **THEN** `game_type = 0` 的定义 SHALL 按“对局、胜场、连胜、赛事历程”顺序展示
- **AND** 球种专属 `special` 定义不得混入通用完成率或通用分组

#### Scenario: 切换球种绝技
- **WHEN** 用户在斯诺克、九球追分、中式八球和美式九球之间切换
- **THEN** 页面 SHALL 只展示 `game_type` 与所选球种一致的绝技
- **AND** 同步更新该球种的专精完成度
- **AND** 不得重置或修改任何进度

#### Scenario: 当前赛季和历届荣誉保持可用
- **WHEN** 新版生涯成就布局上线
- **THEN** 当前赛季挑战、历史挑战快照、赛季荣誉和赛事荣誉的既有 Tab 与数据 SHALL 继续可访问

### Requirement: 本人视图展示即将达成的适用成就
本人荣誉墙 SHALL 从未解锁且适用于通用或当前球种的成就中计算最多三项“即将达成”，按完成百分比降序、定义排序升序排列。

#### Scenario: 选择最接近完成的三项
- **WHEN** 用户拥有多项未解锁的通用或当前球种成就
- **THEN** 页面 SHALL 按 `min(progress / threshold, 1)` 的百分比从高到低显示最多三项
- **AND** 百分比相同时 SHALL 使用定义排序和稳定 ID 保持顺序稳定

#### Scenario: 排除其他球种成就
- **WHEN** 当前选择球种为中式八球
- **THEN** “即将达成”可以包含通用成就和中式八球绝技
- **AND** 不得包含斯诺克或两种九球的专属成就

#### Scenario: 零进度仍可作为后备目标
- **WHEN** 适用范围内不足三项未解锁成就具有正进度
- **THEN** 页面可以按定义顺序使用零进度的适用成就补足三项

#### Scenario: 全部适用成就已解锁
- **WHEN** 用户已经解锁所有通用及当前球种成就
- **THEN** 页面 SHALL 显示完成状态而不是空白或错误的待完成项目

### Requirement: 好友荣誉墙保护未解锁进度
好友只读荣誉墙 SHALL 继续只展示目标用户已经解锁的生涯成就和永久荣誉，不得暴露未解锁成就的进度或“即将达成”列表。

#### Scenario: 好友查看生涯成就
- **WHEN** 获准访问的好友打开目标用户荣誉墙
- **THEN** 返回和展示的生涯成就 SHALL 全部为已解锁项
- **AND** 页面不得展示目标用户的锁定进度或剩余条件

#### Scenario: 好友查看球种绝技
- **WHEN** 好友切换球种专精视图
- **THEN** 页面 SHALL 只筛选并展示目标用户在该球种已经解锁的绝技

### Requirement: 成就详情展示精确条件和奖励
成就详情 SHALL 使用定义中的精确描述展示解锁条件，并展示球种、奖励称号、当前进度、解锁状态和解锁时间；不得对所有成就统一使用“累计达成 N 次”。

#### Scenario: 查看 max 模式连胜成就
- **WHEN** 用户查看“十连制霸”详情
- **THEN** 条件文案 SHALL 表达“个人最高连胜达到 10 场”
- **AND** 不得显示为“累计达成 10 次”

#### Scenario: 查看球种绝技详情
- **WHEN** 用户查看“满分时刻”详情
- **THEN** 页面 SHALL 标识球种为斯诺克
- **AND** 展示精确条件、当前进度和奖励称号“满分王”

#### Scenario: 查看无称号成就详情
- **WHEN** 用户查看不发称号的成就
- **THEN** 页面 SHALL 不显示虚假的称号奖励

### Requirement: 启用成就具有正式系列图标
每条启用的生涯成就定义 SHALL 配置非空图标，客户端优先展示该图标，并只在资源缺失或加载失败时使用分类兜底图形。

#### Scenario: 展示正式成就图标
- **WHEN** 荣誉墙或详情页收到非空 `icon`
- **THEN** 客户端 SHALL 展示对应静态资源
- **AND** 同系列不同里程碑 SHALL 能通过图标阶段变体进行区分

#### Scenario: 图标资源异常时保持可用
- **WHEN** 某个图标资源无法加载
- **THEN** 客户端 SHALL 使用对应分类或球种的兜底图形
- **AND** 列表、进度和详情内容仍须正常展示

### Requirement: 新客户端不以全目录完成率制造跨球种压力
新版荣誉墙 SHALL 以通用完成度和当前球种专精完成度作为主要进度展示，不得只使用包含四球种绝技的单一 `career_unlocked/career_total` 比例。

#### Scenario: 中式八球用户查看摘要
- **WHEN** 只参与中式八球的用户打开荣誉墙
- **THEN** 页面 SHALL 分别展示通用进度和中式八球专精进度
- **AND** 斯诺克及九球未解锁项不得降低这两个主要比例

#### Scenario: 用户切换专精球种
- **WHEN** 用户从中式八球切换到斯诺克
- **THEN** 通用完成度 SHALL 保持不变
- **AND** 专精完成度 SHALL 切换为斯诺克的已解锁数与总数
