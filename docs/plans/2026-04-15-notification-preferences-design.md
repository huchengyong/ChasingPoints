# 通知偏好与消息中心闭环设计

## 目标

把“设置里的通知管理”与“消息中心 / 未读数 / 实时刷新 / Push”接成一条真实闭环链路。

用户关闭某类通知后，该类型的后续新通知应同时停止：

1. 写入消息中心列表
2. 增加未读数
3. 触发 `notification_update` 刷新
4. 触发 Push

本次默认规则：

- 只影响未来新通知
- 不删除历史通知
- 不区分“只关 Push、不关站内消息”这类多通道偏好

---

## 范围结论

本次范围包括：

1. 把通知管理从本地占位页升级为服务端真实配置
2. 将设置项粒度改为与真实业务通知类型一致的 `5` 个开关
3. 在后端新增统一通知分发入口
4. 把消息中心展示类型与后端真实通知类型对齐
5. 让未读数、消息列表、WebSocket 刷新与 Push 都受同一套偏好控制

本次不做：

- 历史通知批量清理或回溯过滤
- “站内消息”和 “Push” 分开控制
- 后台运营配置通知模板
- 新增 `system` / `rank_change` 真实发消息能力

---

## 现状问题

### 1. 设置页不是业务真源

`app/subPages/user/notification.vue` 当前只把 `notification_settings` 写进本地存储，且源码仍保留 `TODO` 注释。它既不调用后端，也不会影响消息中心、未读数、Push 注册或后端通知创建。

### 2. 消息中心走另一条独立链路

消息中心直接请求：

- `GET /api/notification/list`
- `GET /api/notification/unread-count`

因此只要后端继续写 `notifications` 表，关闭设置页开关也不会阻止消息进入列表。

### 3. 通知生产分散

通知当前在多个业务 logic 中直接创建：

- `friend_request`
- `challenge`
- `follow`
- `tournament`
- `match_result`

同时这些位置还会各自尝试发 Push。没有统一入口，自然也没有统一偏好判断。

### 4. 前端类型映射与后端真实类型不一致

消息中心页面当前预留的是：

- `challenge`
- `tournament`
- `rank_change`
- `friend_request`
- `system`

但后端当前真实会发的还包括：

- `follow`
- `match_result`

这会导致展示层只能走默认图标 / 默认样式兜底。

---

## 方案结论

采用“服务端偏好 + 后端统一通知分发 + 前端类型对齐”方案。

核心原则：

1. `用户通知偏好` 由服务端持久化，前端只做展示与编辑
2. `通知是否产生` 在后端统一判断，而不是前端收到后再隐藏
3. `一类通知一个开关`，与真实业务类型保持一一对应
4. `消息中心 / 未读数 / WebSocket / Push` 共用同一份开关结果

---

## 开关设计

设置页改为以下 `5` 个开关：

1. `match_result`：对局结果
2. `friend_request`：好友申请
3. `challenge`：挑战提醒
4. `tournament`：赛事报名提醒
5. `follow`：新关注提醒

默认值：

- 全部 `true`

原因：

- 与当前线上行为兼容
- 老用户升级后不会突然失去通知
- 默认逻辑清晰，缺省记录也好处理

---

## 数据设计

新增表：`user_notification_preferences`

建议字段：

- `id`
- `user_id`，唯一索引
- `match_result_enabled`
- `friend_request_enabled`
- `challenge_enabled`
- `tournament_enabled`
- `follow_enabled`
- `created_at`
- `updated_at`

设计选择：

- 不使用 JSON 字段，直接按列存布尔值
- 读取更直接
- SQL 条件和默认值更清晰
- 后续新增单个开关时迁移和模型也更可控

缺省策略：

- 用户没有记录时，视为全部开启
- 第一次保存时落库

---

## 接口设计

沿用 `notification` 领域，新增两组用户端接口：

1. `GET /api/notification/preferences`
2. `POST /api/notification/preferences`

返回结构建议统一为：

- `match_result_enabled`
- `friend_request_enabled`
- `challenge_enabled`
- `tournament_enabled`
- `follow_enabled`

前端设置页进入时：

- 先读服务端
- 成功后渲染开关
- 可选写本地缓存加速二次进入

前端切换时：

- 调用保存接口
- 以后端返回为准更新页面状态

---

## 后端架构设计

### 1. 新增偏好模型

新增：

- `backend/internal/model/user_notification_preference.go`

负责：

- 读取用户偏好
- 没有记录时返回默认配置
- 保存 / Upsert 用户偏好

### 2. 新增统一通知分发服务

新增共享层服务，建议放在 `backend/internal/logic/` 根目录，因为它会被多个业务域复用。

建议文件：

- `backend/internal/logic/notification_dispatch_service.go`
- `backend/internal/logic/notification_preferences.go`

服务职责：

1. 接收标准化通知请求
2. 根据 `type` 查用户偏好
3. 若关闭则直接跳过，不写库、不推送、不发 WebSocket
4. 若开启则：
   - 写 `notifications`
   - 发 `notification_update`
   - 有 `push_token` 时尝试 Push

建议输入结构：

- `user_id`
- `type`
- `title`
- `content`
- `data`
- `push_payload`
- `ws_category`

### 3. 业务 logic 改为调用统一服务

替换以下位置的直接通知创建逻辑：

- `friend/send_friend_request_logic.go`
- `friend/accept_friend_request_logic.go`
- `challenge/send_challenge_logic.go`
- `challenge/accept_challenge_logic.go`
- `challenge/reject_challenge_logic.go`
- `follow/follow_user_logic.go`
- `tournament/join_tournament_logic.go`
- `match/finish_match_logic.go`

改造后这些 logic 只负责组织业务语义，不再直接决定“是否写通知”。

---

## 前端设计

### 1. 设置页重做为真实配置页

改造：

- `app/subPages/user/notification.vue`

改动点：

1. 开关文案改成 `5` 个真实业务类型
2. 首次进入调用获取偏好接口
3. 切换开关调用保存接口
4. 失败时回滚开关并提示
5. 删除当前本地占位式 `notification_settings` 作为唯一真源的做法

### 2. API 门面补齐

扩展：

- `app/api/notification.js`

新增：

- `getNotificationPreferences`
- `saveNotificationPreferences`

### 3. 消息中心类型映射对齐

改造：

- `app/subPages/notification/index.vue`

补齐：

- `follow`
- `match_result`

并保留未知类型的默认兜底图标。

### 4. 实时刷新保持现有轻量方案

当前“我的”页已经接入用户级 WebSocket，并监听 `notification_update` 后刷新未读数。

本次不额外做全局通知中心总线，只要后端统一服务在“通知真正创建成功后”再发 `notification_update`，当前刷新链路即可与偏好自然一致。

---

## 行为规则

### 1. 关闭开关后的行为

以关闭 `friend_request` 为例：

- 新的好友申请通知不进入 `notifications`
- 未读数不增加
- 不发送对应 `notification_update`
- 不触发 Push

### 2. 历史消息处理

不删除，不隐藏。

原因：

- 风险更低
- 用户已读语义稳定
- 不会把“偏好设置”做成“消息历史治理”

### 3. 开关恢复后的行为

恢复开启后，只影响之后的新消息。

---

## 测试策略

### 后端

重点覆盖：

1. 无偏好记录时默认全开
2. 保存后再次读取结果正确
3. 关闭某类型时统一分发服务会跳过写库 / Push / WebSocket
4. 开启某类型时统一分发服务会按预期执行
5. 旧业务 logic 改造后，通知创建仍保留原有标题、内容、跳转数据

### 前端

重点覆盖：

1. 设置页渲染 `5` 个真实类型开关
2. 设置页使用通知 API 门面，而不是本地孤立逻辑
3. 消息中心包含 `follow` / `match_result` 类型映射

---

## 风险与控制

### 1. 风险：业务改造点分散

控制：

- 引入统一分发服务
- 分任务逐个迁移通知创建点
- 用 focused backend tests 锁住原行为

### 2. 风险：前端旧类型文案和真实类型不一致

控制：

- 设置页与消息中心同一轮一起改
- 使用统一常量映射，避免页面各写一份

### 3. 风险：Push 当前仍是 stub

控制：

- 本次验证以“是否调用统一服务分支”与日志行为为准
- 不把真实设备 Push 到达率作为本轮阻塞条件

---

## 最终结论

本次应把通知管理升级为“真实业务偏好”，并让它成为通知生产链路的前置判断。

只有把“是否生成通知”统一收在后端，消息中心、未读数、WebSocket 刷新与 Push 才会真正形成闭环。
