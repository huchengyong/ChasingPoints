# Member Subscription And Friend Homepage Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 打通 APP 端月卡会员真实支付闭环，同时精简好友主页并把“隐藏战绩”改成真实服务端隐私能力，且不改动现有“上传球馆领会员”奖励卡。

**Architecture:** 前端复用现有好友与战绩页面能力，把好友主页改成“好友名片 + 直接战绩列表/隐藏态”，新增独立会员中心页；后端新增用户隐私接口与会员支付域，使用 go-zero `.api` 契约驱动生成代码，并通过订单表 + 支付回调幂等更新 `users.member_expires_at`。所有用户可见时间统一返回 `UTC+8` 字符串。

**Tech Stack:** UniApp Vue3、SCSS、Pinia、Node `node:test`、go-zero、Gorm、MySQL migration、`goctl api go`、`uni.requestPayment`、支付宝/微信 APP 支付。

---

### Task 1: 为好友主页“纯好友视角”写前端纯函数测试

**Files:**
- Modify: `app/tests/friend-homepage.test.mjs`
- Modify: `app/utils/friend-homepage.js`

**Step 1: Write the failing test**

- 在 `app/tests/friend-homepage.test.mjs` 新增断言：
  - 新的 Hero 文案不再出现“你领先”“你们交手”等“我方视角”文案
  - 输出结构包含对方名片所需字段和战绩区标题
  - 当 `isHidden` 为 `true` 时返回 `对方已隐藏战绩` 空态

**Step 2: Run test to verify it fails**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/app && node --test tests/friend-homepage.test.mjs`

Expected: FAIL，提示现有 summary 仍输出对比文案或缺少隐藏态。

**Step 3: Write minimal implementation**

- 在 `app/utils/friend-homepage.js` 改造 view-model 纯函数：
  - 去掉双方对比 Hero 文案
  - 增加好友名片描述
  - 增加隐藏态构造

**Step 4: Run test to verify it passes**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/app && node --test tests/friend-homepage.test.mjs`

Expected: PASS

### Task 2: 完成好友主页页面结构精简

**Files:**
- Modify: `app/subPages/social/friendHomepage.vue`
- Modify: `app/subPages/social/friendHomepage.scss`
- Modify: `app/utils/friend-entry.js`

**Step 1: Write minimal implementation**

- 在 `friendHomepage.vue`：
  - Hero 只保留头像、昵称、ID、段位、`PK 报表` 按钮
  - 删除“继续查看”按钮区
  - 直接内嵌对方战绩列表区域
- 在 `friendHomepage.scss` 同步整理布局
- 如有必要在 `friend-entry.js` 补辅助参数透传

**Step 2: Run targeted verification**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/app && node --test tests/friend-homepage.test.mjs`

Expected: PASS

### Task 3: 为隐藏战绩后端开关写后端失败测试

**Files:**
- Create: `backend/internal/logic/user/privacy_logic_test.go`
- Modify: `backend/internal/model/user.go`
- Modify: `backend/internal/testsupport/*`（按需要）

**Step 1: Write the failing test**

- 新增用户隐私逻辑测试，覆盖：
  - 获取隐私配置默认返回 `hide_match_record = false`
  - 更新后可以返回 `true`
  - 非法请求不会写脏数据

**Step 2: Run test to verify it fails**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/user -run TestUserPrivacy`

Expected: FAIL，提示缺少字段、接口或逻辑。

**Step 3: Write minimal implementation**

- 在 `users` 表设计对应字段
- 在 `User` model 中新增字段和更新方法，但先不改 API

**Step 4: Run test to verify it passes partially**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/user -run TestUserPrivacy`

Expected: 仍可能 FAIL，但失败点应收敛到 API/logic 未接入。

### Task 4: 扩展用户隐私 API 契约并生成代码

**Files:**
- Modify: `backend/chasing_points.api`
- Modify: `backend/internal/config/config.go`（如需新增通用时间配置则在此处处理）
- Modify: `backend/internal/types/types.go`
- Modify: `backend/internal/handler/routes.go`

**Step 1: Write the failing test**

- 在 `privacy_logic_test.go` 中补针对生成后的 request/response 结构断言：
  - `GET /api/user/privacy`
  - `POST /api/user/privacy`

**Step 2: Run test to verify it fails**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/user -run TestUserPrivacy`

Expected: FAIL，提示缺少 types 或 handler 路由。

**Step 3: Write minimal implementation**

- 在 `.api` 新增隐私接口与类型
- 立刻运行：
  - `cd /Users/wisesearch/Projects/ChasingPoints/backend && goctl api go --api chasing_points.api --dir . --style go_zero --home ~/.goctl/default`

**Step 4: Run test to verify it fails in the intended place**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/user -run TestUserPrivacy`

Expected: FAIL，失败点转移到具体 logic 未实现。

### Task 5: 实现用户隐私 logic 与 UTC+8 时间工具

**Files:**
- Create: `backend/internal/logic/user/get_user_privacy_logic.go`
- Create: `backend/internal/logic/user/update_user_privacy_logic.go`
- Create: `backend/internal/logic/timezone.go`
- Modify: `backend/internal/model/user.go`
- Modify: `backend/internal/logic/user/privacy_logic_test.go`

**Step 1: Write minimal implementation**

- 实现获取 / 更新隐藏战绩 logic
- 提供统一 `UTC+8` 时间格式化 helper，供后续会员与订单复用
- `UserModel` 新增隐私字段更新方法

**Step 2: Run test to verify it passes**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/user -run TestUserPrivacy`

Expected: PASS

**Step 3: Commit**

```bash
git add backend/chasing_points.api backend/internal/model/user.go backend/internal/logic/user/get_user_privacy_logic.go backend/internal/logic/user/update_user_privacy_logic.go backend/internal/logic/user/privacy_logic_test.go backend/internal/logic/timezone.go backend/internal/types/types.go backend/internal/handler/routes.go
git commit -m "feat: add server-backed user privacy settings"
```

### Task 6: 为好友主页隐藏战绩权限写后端失败测试

**Files:**
- Modify: `backend/internal/logic/opponent/get_opponent_list_logic_test.go`
- Modify: `backend/internal/logic/opponent/get_opponent_list_logic.go`

**Step 1: Write the failing test**

- 新增断言：
  - 好友目标用户 `hide_match_record = true` 时，好友模式不返回对手列表明细
  - 同一目标用户在 `PK 报表` / H2H 逻辑不受该开关影响

**Step 2: Run test to verify it fails**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/opponent -run TestGetOpponentList`

Expected: FAIL，提示仍返回明细或无隐藏态字段。

**Step 3: Write minimal implementation**

- 在对手列表响应中增加隐藏态字段
- 好友模式下，如果目标用户隐藏战绩，直接返回空列表 + `hidden = true`

**Step 4: Run test to verify it passes**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/opponent -run TestGetOpponentList`

Expected: PASS

### Task 7: 为会员订单模型与顺延规则写后端失败测试

**Files:**
- Create: `backend/internal/model/member_subscription_order.go`
- Create: `backend/internal/logic/payment/member_subscription_service_test.go`
- Create: `backend/internal/logic/payment/member_subscription_service.go`

**Step 1: Write the failing test**

- 覆盖：
  - 用户无会员时购买月卡，从当前 `UTC+8` 时间起顺延 1 个月
  - 用户会员未过期时，在原到期时间基础上顺延
  - 重复回调不会重复发放
  - 非法金额/状态不会把订单标记为支付成功

**Step 2: Run test to verify it fails**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/payment -run TestMemberSubscription`

Expected: FAIL，提示缺少模型或服务实现。

**Step 3: Write minimal implementation**

- 提供纯服务方法：
  - 订单号生成
  - 到期时间计算
  - 幂等支付成功处理骨架

**Step 4: Run test to verify it passes**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/payment -run TestMemberSubscription`

Expected: PASS

### Task 8: 增加会员订单迁移与 model

**Files:**
- Create: `backend/migrations/20260331xxxxxx_add_member_subscription_order.sql`
- Create: `backend/internal/model/member_subscription_order.go`
- Modify: `backend/internal/svc/service_context.go`

**Step 1: Write minimal implementation**

- 新增会员订单表 migration
- 新增 model 和查询/更新方法
- 注入 `ServiceContext`

**Step 2: Run focused verification**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/model ./internal/logic/payment`

Expected: PASS

### Task 9: 扩展支付与会员 API 契约并生成代码

**Files:**
- Modify: `backend/chasing_points.api`
- Modify: `backend/internal/config/config.go`
- Modify: `backend/etc/chasing_points-api.yaml`
- Modify: `backend/internal/types/types.go`
- Modify: `backend/internal/handler/routes.go`

**Step 1: Write the failing test**

- 为支付 logic 测试补齐 request/response 类型预期：
  - 套餐查询
  - 会员状态
  - 创建订单
  - 查询订单状态
  - 支付宝回调
  - 微信回调

**Step 2: Run test to verify it fails**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/payment -run TestMemberSubscription`

Expected: FAIL，提示缺少 API 类型或配置字段。

**Step 3: Write minimal implementation**

- 在配置中加入支付宝/微信支付字段映射
- 在 `.api` 增加 payment/member 相关接口
- 立刻运行：
  - `cd /Users/wisesearch/Projects/ChasingPoints/backend && goctl api go --api chasing_points.api --dir . --style go_zero --home ~/.goctl/default`

**Step 4: Run test to verify it fails in the intended place**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/payment -run TestMemberSubscription`

Expected: FAIL，失败点收敛到具体 handler/logic 未实现。

### Task 10: 实现支付创建、回调与会员到账逻辑

**Files:**
- Create: `backend/internal/logic/payment/get_member_plans_logic.go`
- Create: `backend/internal/logic/payment/get_member_status_logic.go`
- Create: `backend/internal/logic/payment/create_member_order_logic.go`
- Create: `backend/internal/logic/payment/get_member_order_status_logic.go`
- Create: `backend/internal/logic/payment/alipay_member_notify_logic.go`
- Create: `backend/internal/logic/payment/wechat_member_notify_logic.go`
- Modify: `backend/internal/logic/payment/member_subscription_service.go`
- Modify: `backend/internal/logic/payment/member_subscription_service_test.go`

**Step 1: Write minimal implementation**

- 查询月卡套餐
- 查询会员状态
- 创建会员订单并生成 APP 支付参数
- 回调验签、金额校验、订单状态幂等更新、用户会员到账

**Step 2: Run targeted tests**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./internal/logic/payment`

Expected: PASS

**Step 3: Commit**

```bash
git add backend/chasing_points.api backend/internal/config/config.go backend/etc/chasing_points-api.yaml backend/internal/svc/service_context.go backend/internal/model/member_subscription_order.go backend/internal/logic/payment backend/internal/types/types.go backend/internal/handler/routes.go backend/migrations
git commit -m "feat: add app member subscription payment flow"
```

### Task 11: 为会员中心前端纯函数与 API 门面写失败测试

**Files:**
- Create: `app/api/payment.js`
- Create: `app/utils/member-center.js`
- Create: `app/tests/member-center.test.mjs`

**Step 1: Write the failing test**

- 覆盖：
  - 会员状态卡文案映射
  - 月卡套餐按钮文案
  - 支付方式切换
  - `UTC+8` 时间字符串展示直出，不再做二次转换

**Step 2: Run test to verify it fails**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/app && node --test tests/member-center.test.mjs`

Expected: FAIL，提示缺少纯函数或 API 门面。

**Step 3: Write minimal implementation**

- 新增 `payment.js` API 门面
- 新增 `member-center.js` 纯函数

**Step 4: Run test to verify it passes**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/app && node --test tests/member-center.test.mjs`

Expected: PASS

### Task 12: 完成“我的”页会员入口与隐藏战绩真实接入

**Files:**
- Modify: `app/pages/user/index.vue`
- Modify: `app/pages/user/index.scss`
- Modify: `app/api/user.js`
- Modify: `app/store/user.js`（仅在确有必要时）

**Step 1: Write minimal implementation**

- 保持现有“上传球馆领会员”奖励卡不动
- 新增独立会员中心入口卡
- 页面加载时调用真实隐私接口回显 `隐藏战绩`
- 切换开关时调用后端更新接口，不再只写本地存储

**Step 2: Run targeted verification**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/app && node --test tests/member-center.test.mjs tests/friend-homepage.test.mjs`

Expected: PASS

### Task 13: 新增会员中心页并接入 APP 支付

**Files:**
- Create: `app/subPages/user/memberCenter.vue`
- Create: `app/subPages/user/memberCenter.scss`
- Modify: `app/pages.json`
- Modify: `app/api/payment.js`

**Step 1: Write minimal implementation**

- 新建会员中心页
- 展示当前会员状态、月卡套餐、支付方式切换
- 调用创建订单接口
- 使用 `uni.requestPayment` 拉起支付宝 / 微信支付
- 支付完成后查询订单状态并刷新会员状态

**Step 2: Run targeted verification**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/app && node --test tests/member-center.test.mjs`

Expected: PASS

### Task 14: 把好友主页接入真实隐藏态与内嵌战绩列表

**Files:**
- Modify: `app/subPages/social/friendHomepage.vue`
- Modify: `app/api/match.js`
- Modify: `app/utils/friend-homepage.js`

**Step 1: Write minimal implementation**

- 读取后端对手列表接口的隐藏态字段
- 隐藏时展示 `对方已隐藏战绩`
- 非隐藏时直接展示对方战绩列表
- `PK 报表` 按钮继续进入原页面

**Step 2: Run targeted verification**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/app && node --test tests/friend-homepage.test.mjs tests/member-center.test.mjs`

Expected: PASS

### Task 15: 全量验证

**Files:**
- Verify only

**Step 1: Run front-end tests**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/app && node --test tests/*.test.mjs`

Expected: PASS

**Step 2: Run back-end tests**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && GOCACHE=/tmp/chasing-points-go-cache go test ./...`

Expected: PASS

**Step 3: Smoke-check generated code**

Run: `cd /Users/wisesearch/Projects/ChasingPoints/backend && go test ./internal/logic/payment ./internal/logic/user ./internal/logic/opponent`

Expected: PASS

**Step 4: Commit**

```bash
git add app backend docs/plans
git commit -m "feat: add app member subscription and simplify friend homepage"
```
