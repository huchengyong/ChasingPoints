## 1. 建立性能观测基线

- [x] 1.1 盘点仓库、Makefile、启动命令和部署脚本对两份后端 YAML 的引用，确认 `backend/etc/chasing_points-api.yaml` 为唯一运行真源，并移除或明确废弃未使用的 `backend/etc/chasingpoints-api.yaml`
- [x] 1.2 保留 JSON 文件日志并启用 go-zero 标准统计或等价低开销请求完成观测，记录规范化 route、method、status、duration、response bytes 和 request id
- [x] 1.3 为 Gorm 增加请求级 SQL 次数与总耗时统计、慢查询阈值和脱敏结构化日志，并提供测试可读取的 query-count 计数器
- [x] 1.4 为现有 event-news Redis 缓存增加 hit、miss、decode error、Redis error、write error 和 fallback 指标，形成后续缓存复用模板
- [x] 1.5 为移动端请求层增加仅用于测试/调试的按 route 请求计数能力，不缓存响应且不记录 Token 等敏感值
- [x] 1.6 记录优化前 App 冷启动、前台恢复、首页、观赛、赛讯、“我的”、排行榜、统计详情和 H2H 首屏的请求数、SQL 次数、响应大小及可取得的 P50/P95/P99
- [x] 1.7 根据基线为核心页面请求数和热点接口查询次数确定回归上限，并将结果写入测试或发布检查文档

## 2. 完成低风险客户端重复请求治理

- [x] 2.1 新增认证隔离的 activity Store，使用 `ownerUserId`、`authGeneration`、`loaded/dirty/loadedAt/inFlight` 管理当前对局、未读通知数、好友申请数和换季提醒，初期复用现有细粒度 API
- [x] 2.2 新增认证隔离的 userOverview Store，先复用现有统计、信誉、会员和常玩球馆奖励接口，并支持 single-flight、SWR、手动强刷及刷新失败保留旧值
- [x] 2.3 为赛讯、排行榜摘要/首屏、附近球馆和静态公共数据建立最小领域缓存状态，按球种、分页或位置桶区分键，不在 `app/utils/request.js` 增加通用 GET 缓存
- [x] 2.4 修改 `app/App.vue`、首页、观赛页和“我的”页复用 activity/userOverview/rank/user Store，消除当前对局、角标、奖励状态和用户资料的并发重复请求
- [x] 2.5 修复 `app/pages/ranking/index.vue` 首次 `onMounted` 与 `onShow` 双请求，并取消 App 每次前台恢复对 rankStore 的无条件失效
- [x] 2.6 修改首页和 `app/pages/social/index.vue` 的赛讯加载规则，使用 loaded/TTL/SWR，保留用户下拉刷新强制更新
- [x] 2.7 修改 `app/subPages/social/challenges.vue` 为一次加载挑战集合、三类 Tab 本地筛选，并保留业务写操作后的 dirty 刷新
- [x] 2.8 修改设置、编辑资料、会员中心、信誉、段位说明页优先复用现有 Store/静态配置，并只在缺失、dirty、TTL 到期或手动刷新时请求
- [x] 2.9 修改荣誉墙、统计详情、PK 报告、好友主页、我的动态等保留页面，移除无条件 `onShow` 全量刷新并补齐明确业务失效事件
- [x] 2.10 补充 App 逻辑测试，覆盖账号切换不串缓存、旧账号 in-flight 不回写、single-flight、SWR 失败保留旧值、排行榜首次单请求和挑战 Tab 零额外请求

## 3. 扩展 API 契约与读模型 Schema

- [x] 3.1 在 `backend/chasing_points.api` 一次性定义 Bootstrap、用户概览、统计概览、H2H 概览、对手候选、赛季概览、榜单摘要、段位配置、成就详情、notification unread_count、availability/partial errors 和 revision 字段，并在修改后立即运行 goctl 生成代码
- [x] 3.2 同步新增/修改 `app/api/*.js` 门面，页面仍不得直接调用 `uni.request`
- [x] 3.3 新增 migration，为 `matches` 增加可索引 `completed_at` 并为后续按状态、模式、球种和完成时间查询增加经 EXPLAIN 验证的组合索引
- [x] 3.4 新增 migration，创建 `match_participant_results`、`user_competitive_stats`、`user_opponent_stats`、对手强度桶和读模型 rebuild checkpoint 所需表、唯一键及索引
- [x] 3.5 新增 migration，为好友申请、挑战过期、赛季榜单、赛季 settlement、achievement season-wide 聚合、球馆地理候选、公开比赛和其他已确认热查询增加必要组合索引，避免重复或未使用索引
- [x] 3.6 新增对应 Gorm 模型和 Model 方法并注入 `backend/internal/svc/ServiceContext`，不得通过 AutoMigrate 代替生产 migration
- [x] 3.7 补充 migration、模型唯一键、索引存在性、Up/Down 兼容性和 `.api` 契约测试

## 4. 实现比赛参与者投影与竞技快照

- [x] 4.1 定义注册用户和访客对手的稳定身份规范化规则，覆盖空白、大小写、昵称变化及同名访客边界
- [x] 4.2 实现从单场已完成 match 构建双方视角 `match_participant_results` 的共享 projector，正确翻转结果、比分、对手和完成归因
- [x] 4.3 实现只读取当前 `match_id` 局/动作的比赛级最高分、单杆最高分和时长计算，不读取用户历史
- [x] 4.4 实现用户总体及分球种 `user_competitive_stats` 原子 upsert，维护胜负、当前/最大连胜、最高分、时长聚合、last_match 和单调 revision
- [x] 4.5 实现双方视角 `user_opponent_stats` 原子 upsert，维护总体/分球种 H2H 场次、胜负和最后交手时间
- [x] 4.6 实现按比赛结算时对手段位分档的强度桶增量更新，确保对手后续升降段不重写历史口径
- [x] 4.7 实现按比赛 `completed_at` 归属赛季并增量 upsert `season_records` 的起始分、当前分、峰值、场次和胜场
- [x] 4.8 以首次插入 `match_id + user_id` 投影或等价唯一门闩控制统计增量，保证重复 Finish、重放和多实例并发不重复计数
- [x] 4.9 为 projector 增加注册双方、访客对手、四球种、发起方/对手翻转、连胜、最高分、赛季边界、重复执行和事务回滚测试
- [x] 4.10 实现可恢复的 read-model rebuild 命令，支持 dry-run、按 match id 游标固定 batch、checkpoint、限速、暂停恢复和幂等重跑
- [x] 4.11 实现读模型审计命令，按抽样用户/比赛比较旧口径与投影/快照结果，并输出可定位差异而不修改业务真源

## 5. 将读模型接入比赛结算和实时失效

- [x] 5.1 在所有成功完成比赛路径中持久化规范化 `completed_at`，并保证取消、无效或 result=3 的比赛不进入排位统计投影
- [x] 5.2 将参与者投影、用户统计、对手统计、强度桶和当前赛季记录接入比赛完成业务事务，任一必要写入失败时不得提交部分统计
- [x] 5.3 保持现有 Finish 幂等协议，补充重复 client action、多实例竞争及裁判/双方不同完成来源下的双写一致性测试
- [x] 5.4 在事务提交后发送与通知偏好无关的 `user_data_updated` 用户 WebSocket 消息，携带受影响 scopes、competitive revision、match id 和 game type
- [x] 5.5 扩展通知/好友关系用户事件携带最新未读数或精确计数变化，避免客户端收到 category 后重新查询无关角标
- [x] 5.6 修改 App 用户 WebSocket 处理，按 scopes 精确 dirty rank、stats、H2H、opponents、history、honor、member、reputation、season 和 leaderboard 资源
- [x] 5.7 补充“关闭比赛结果通知仍收到竞技失效”、WebSocket 发送失败后 Bootstrap revision 收敛、Redis 失效失败不回滚比赛的测试

## 6. 切换历史、统计和结果读取到新模型

- [x] 6.1 修改现有用户总体统计接口从固定快照行读取，删除 `GetUserStats` 和最大连胜的用户全历史扫描
- [x] 6.2 修改分球种统计、比赛最高分、单杆最高分和时长接口从竞技快照读取，并保持现有响应口径兼容
- [x] 6.3 修改对手强度接口读取结算时分档快照，补充对手后续升段不改变历史分档的测试
- [x] 6.4 修改近期胜率趋势按参与者投影索引读取最近 N 场；段位趋势继续使用有界 rank change log 查询
- [x] 6.5 修改用户对局列表使用参与者投影做数据库过滤、排序、COUNT 和分页，不得加载全部历史后在 Go 中切片
- [x] 6.6 修改 H2H 历史使用用户/对手/球种/完成时间索引分页，修改 H2H 聚合读取 `user_opponent_stats`
- [x] 6.7 修改对手列表和好友主页对手摘要读取对手快照并按 last_match_at 分页，不得重复扫描用户全部比赛
- [x] 6.8 抽取完成对局核心摘要 builder，只读取目标 match、相关局/动作、段位变化、成就和完成归因，并统一 participant/viewer 视角
- [x] 6.9 修改私有详情、公开详情、结果页和分享服务复用同一核心摘要，移除双方历史胜率/最高分的重复计算
- [x] 6.10 结果页与分享组装时从双方最新竞技快照读取当前胜率和最高分，验证旧比赛在用户完成新比赛后显示新指标而核心比分不变
- [x] 6.11 为上述接口增加新旧数据抽样一致性、固定 query-count、稳定分页和大历史数据量测试

## 7. 消除列表 N+1 与不可索引地理查询

- [x] 7.1 重写好友列表为固定数量 JOIN/批量查询用户和段位，验证返回 1 条与 100 条时 SQL 次数不线性增长
- [x] 7.2 为待处理好友申请提供专用 COUNT，并将申请列表的发起人资料改为 JOIN/批量读取，停止以 `page_size=1` 加载完整集合取 total
- [x] 7.3 优化好友搜索，一次批量取得候选用户的好友/申请/拉黑关系和必要段位，消除逐用户关系查询
- [x] 7.4 优化挑战列表为批量取得双方资料，并在 SQL 中按 `expires_at` 表达有效状态
- [x] 7.5 优化球馆列表/附近球馆使用聚合子查询或 JOIN 一次取得签到数，消除逐球馆 COUNT
- [x] 7.6 为附近球馆实现可索引经纬度边界框或标准空间候选预筛选，再对限定候选计算 Haversine，并用 EXPLAIN 验证没有整表距离计算
- [x] 7.7 优化赛季排行榜为 JOIN 用户资料并使用覆盖排序组合索引，消除逐记录 `FindById`
- [x] 7.8 优化裁判历史和公开比赛列表，以 JOIN/聚合一次取得双方资料及 round count，消除逐场查询
- [x] 7.9 为好友、申请、挑战、球馆、赛季榜单、裁判历史和公开比赛补充 query-count 与代表性 MySQL EXPLAIN 测试

## 8. 清理 GET 写副作用并迁移到有界任务

- [x] 8.1 将规则默认数据播种移至 migration 或服务启动一次初始化，删除规则分类、内容、术语和搜索 GET 中的 `SeedData`
- [x] 8.2 将四球种段位行初始化接入用户创建/首次认证初始化，并提供幂等修复命令，删除 rank GET 的 `FindOrCreate` 写行为
- [x] 8.3 实现按 `status + expires_at + id` 索引分批的挑战过期 Worker，GET/接受/拒绝按有效时间判断且不执行全局 `ExpireOld`
- [x] 8.4 将赛事 bracket 生成移至赛事创建、显式管理操作或独立幂等任务，GET 只读取已生成 bracket 或返回明确 availability
- [x] 8.5 将 completed match 成就/奖励补偿移至按 `achievement_synced_at/status/id` 索引分批的可重试 Worker，reward summary GET 只返回 pending/ready
- [x] 8.6 将过期 finish request 收敛移至按状态/时间索引 Worker，当前对局/详情 GET 返回时间计算后的有效视图但不写数据库
- [x] 8.7 增加 GET 纯读回归测试，断言规则、挑战、段位、bracket、当前对局和 reward summary 读取不会产生业务表写操作
- [x] 8.8 验证所有新增 Worker 有固定 batch、稳定游标、唯一/条件更新幂等、可停止和错误重试，不持有跨批次事务

## 9. 实现后端聚合与直接详情接口

- [x] 9.1 扩展 ActiveUserSessionMiddleware 将已校验用户放入 Context，并修改用户相关 Logic 在同一请求内复用且不跨请求缓存活动状态
- [x] 9.2 实现用户 Bootstrap，聚合用户资料、纯读当前对局、通知/好友角标、最近未读换季通知和竞技 revision，支持 availability
- [x] 9.3 实现用户 overview，从统计快照及现有信誉、会员、球馆奖励服务返回四个区块，并保留部分区块失败语义
- [x] 9.4 实现 stats overview，从快照和有界趋势查询返回六个统计区块，限制独立查询并发且不顺序重算历史
- [x] 9.5 实现 H2H overview，一次返回对手资料、快照统计、指定月份首屏历史、total 和下一页信息，支持注册与访客身份
- [x] 9.6 实现 opponent candidates，批量合并好友和有限近期对手并按稳定身份去重
- [x] 9.7 实现 season overview，一次返回 season_state、权威边界、指定球种本人实时记录和排行榜第一页，赛季不可用时不执行后续查询
- [x] 9.8 实现 leaderboard summary，只读取前三名和当前用户排名，不查询第 4～6 名或完整 total/list
- [x] 9.9 修改通知列表首屏响应附带一致 unread_count 或 revision，并保留独立分页
- [x] 9.10 实现静态 rank configs 接口和按 ID 成就详情接口，使段位说明不再个性化查询配置、成就详情不再下载完整列表
- [x] 9.11 为所有聚合接口补充主资源失败、可选区块部分失败、有界并发、固定 query-count、认证隔离和旧细粒度接口同口径测试

## 10. 将移动端切换到聚合接口与 revision 协调

- [x] 10.1 修改 App 登录恢复、冷启动和前台恢复调用一次 Bootstrap，并用返回值原子更新 user、activity 和竞技 revision 状态
- [x] 10.2 修改首页、观赛和“我的”从 activity Store 读取当前对局与角标，删除紧随 Bootstrap 的重复细粒度调用
- [x] 10.3 修改“我的”页使用 user overview，并保留段位 Store、当前对局和角标的独立资源边界
- [x] 10.4 修改统计详情页使用 stats overview，球种切换只刷新球种相关区块，月份切换继续本地筛选已加载趋势
- [x] 10.5 修改 H2H 页面和 PK 报告使用 H2H overview，首屏不再对同月份发出两次 history 请求，后续仅分页加载历史
- [x] 10.6 修改对手选择页使用 opponent candidates，删除“好友 100 条 + 比赛 100 条”下载和客户端历史去重
- [x] 10.7 修改赛季页使用 season overview，后续 load more 仅调用排行榜分页，并根据 revision/换季事件 dirty
- [x] 10.8 修改通知页使用列表响应/Store 中的 unread_count，不在首屏后立即追加 count 请求
- [x] 10.9 修改段位说明页复用 rankStore + rank configs，成就详情优先复用荣誉墙实体并在深链时调用按 ID 详情
- [x] 10.10 补充页面请求图测试，验证冷启动、前台恢复和各核心首屏达到阶段 0 确定的请求数上限且账号切换安全

## 11. 接入共享和静态数据缓存

- [x] 11.1 为 event-news 列表/详情增加客户端 3～5 分钟 SWR，复用现有后端 5/3 分钟版本缓存并覆盖首页、赛讯和赛事入口
- [x] 11.2 为公共排行榜共享部分实现按球种/分页/version 的 Redis 缓存，个人排名独立查询后组合；排位结算和资料更新按领域失效
- [x] 11.3 为首页榜单摘要和排行榜页面增加客户端按球种 TTL/single-flight 缓存，并验证首次生命周期不双请求
- [x] 11.4 为规则、术语、地区、段位配置和会员套餐实现 app/schema/config version 的客户端持久缓存及服务端进程内或 Redis 缓存
- [x] 11.5 为球馆列表、详情和附近候选实现短 TTL Redis 缓存，附近 key 使用离散位置桶、radius、limit 和领域版本，不使用完整经纬度
- [x] 11.6 为当前/历史赛季公共信息和赛季榜单共享部分实现按窗口/赛季/球种版本缓存，并在比赛结算、换季和资料更新时失效
- [x] 11.7 为 ready 的完成对局核心摘要实现长 TTL Redis 缓存和进程内 single-flight，pending reward/achievement 状态不得写入长期缓存
- [x] 11.8 保证结果页当前胜率和最高分不固化在核心缓存中，每次按双方最新 snapshot revision 读取/组装
- [x] 11.9 为所有新缓存增加 hit/miss/error/fallback 指标、Redis 故障回源、损坏 JSON 回源、热点同时过期和用户/参数键隔离测试
- [x] 11.10 增加禁缓存回归测试，覆盖认证、Token、支付状态、上传凭证、裁判二维码/加入、用户搜索、进行中比赛和所有写操作

## 12. 优化连续赛季生命周期热路径

- [x] 12.1 在 season 共享层实现根据 policy 计算期望窗口并按唯一 start_date/窗口索引读取有限候选的 resolver，替代在线 `ListAll`
- [x] 12.2 修改当前赛季、默认本人记录、默认榜单和本人荣誉墙复用同一 resolver，删除 achievement 与 season 目录中的重复生命周期解析代码
- [x] 12.3 重写 SeasonLifecycleService 正常快路径，只确保期望当前和下一窗口；完整历史冲突扫描保留在 repair/dry-run
- [x] 12.4 重写 SeasonRolloverWorker 按 end/status/settlement 索引分页查询到期未结算窗口，稳定 tick 不得多次 `ListAll`
- [x] 12.5 验证比赛结算增量 `season_records` 后，当前赛季本人记录和榜单实时可读，并按赛季/球种 bump 排名版本
- [x] 12.6 重写 SeasonSettlementService 基于增量 season records 最终化排名，不再调用 `ListCompletedRankedBetween` 或逐用户球种查询 rank logs
- [x] 12.7 为换季增加可恢复阶段和最终小事务发布，确保中间结果不可对外标记 completed，失败重试不重复记录、称号或通知
- [x] 12.8 将赛季挑战归档改为按用户/球种/指标的一次或有界批量 GROUP BY，并增加支持 metric/time season-wide 查询的专用索引
- [x] 12.9 将赛季通知和其他大批量写入按固定批次幂等处理，实时 Push/WS 在数据库提交后发送
- [x] 12.10 修改 season lifecycle repair/rebuild 使用窗口/主键游标固定 batch，支持 dry-run、checkpoint 和多年历史中断恢复
- [x] 12.11 补充当前窗口固定查询次数、稳定 Worker tick、跨多届追赶、大赛季结算、挑战归档无 N+1、多实例幂等和赛季边界测试

## 13. 验收、清理与运行手册

- [x] 13.1 运行 `cd app && node --test tests/*.test.mjs`，修复所有客户端缓存、生命周期、页面请求图和 API 契约回归
- [x] 13.2 运行 `cd backend && go test ./...`，修复所有读模型、比赛结算、查询次数、缓存、Worker、迁移和连续赛季回归
- [x] 13.3 在代表性数据量上执行 completed_at 回填和 read-model rebuild dry-run/正式重建，验证 checkpoint 恢复、重复执行零重复和审计差异归零
- [ ] 13.4 使用阶段 0 相同场景复测请求数、SQL 次数、P50/P95/P99、响应大小、Redis 命中率和冷 miss 性能，并记录每阶段收益（生产流量与网络指标须在发布后补采）
- [x] 13.5 使用静态搜索和 query-count 测试确认在线路径不再存在已识别的 `ListAll` 热调用、全历史 `Find` 后内存分页、逐条关联查询、GET `SeedData/ExpireOld/FindOrCreate` 和 `COALESCE` 完成时间热查询
- [x] 13.6 删除已被新投影/聚合替代的旧全量扫描 helper、重复结果 builder、无用 imports 和已确认未使用的配置文件，仅清理本变更产生的 orphan
- [x] 13.7 编写运维运行手册，说明日志位置、指标、Redis 回源、缓存版本、读模型 dry-run/rebuild/audit、赛季 repair、回滚和异常恢复步骤
- [x] 13.8 运行 `openspec validate --change optimize-client-api-read-performance` 并确认所有规格、任务和实现证据满足 apply-ready/发布验收要求

## 14. 发布前复审整改

- [x] 14.1 修复赛季首屏加载、Bootstrap reservation 清理、认证代次隔离和用户数据 scope 驱动的保留页面刷新，并补充回归测试
- [x] 14.2 为允许匿名访问的公共榜单/好友观赛路径安全解析可选 access token，保证个人排名和好友范围在真实 HTTP 路由可用
- [x] 14.3 使 HTTP 请求上下文绑定 Gorm Model 查询、认证校验也计入 SQL 指标，并确保异常请求完成观测和慢 SQL 日志不泄露原始数据库错误值
- [x] 14.4 将读模型回建改为先有界回填 completed_at、再按 `(completed_at, id)` 回放；checkpoint 只前进到成功记录，并覆盖失败恢复与乱序历史测试
- [x] 14.5 增加生产读模型切换门禁和回退语义，更新 runbook 为“先双写/回建/审计、后切读”的可执行顺序
- [x] 14.6 修复赛季业务时区归属、挑战读取上限和热点索引迁移的 Up/Down 对称性，并补充验证
- [x] 14.7 复跑本地回归、MySQL 迁移验收与严格 OpenSpec 校验；保留生产真实性能采集为未完成的发布后任务
