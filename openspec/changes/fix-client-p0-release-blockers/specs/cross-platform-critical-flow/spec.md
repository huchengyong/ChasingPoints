## ADDED Requirements

### Requirement: 核心扫码入口必须覆盖 App、HarmonyOS 和微信小程序
首页、对局页、排行榜和“我的”等现有发起 PK 入口，以及裁判扫码入口，SHALL 在 App、HarmonyOS 和 MP-WEIXIN 使用统一扫码能力；平台条件编译 MUST NOT 将微信小程序静默排除。

#### Scenario: 微信小程序发起 PK 扫码
- **WHEN** 已登录用户在微信小程序选择球种和比赛模式后进入扫码
- **THEN** 客户端 SHALL 调用微信小程序支持的扫码能力
- **AND** 成功扫码后 SHALL 进入与 App 等价的预览或开局流程

#### Scenario: 微信小程序扫码担任裁判
- **WHEN** 已登录用户在微信小程序扫描有效裁判码
- **THEN** 客户端 SHALL 展示比赛预览并要求用户确认
- **AND** 确认后 SHALL 使用服务端 join token加入裁判链路

#### Scenario: 用户取消扫码
- **WHEN** 用户主动取消扫码
- **THEN** 客户端 SHALL 返回原页面且不显示“暂无数据”或系统故障
- **AND** MUST NOT 创建对局或消耗邀请凭据

### Requirement: 匹配二维码必须使用服务端可验证邀请凭据
匹配二维码 MUST 携带有用途、签发用户和有效期约束的服务端可验证邀请凭据，不得仅依赖客户端可修改的 `user_id/nickname/avatar/ts` JSON。服务端 SHALL 以验证结果确定对手身份，客户端展示字段不得成为开局授权依据。

#### Scenario: 扫描有效匹配码
- **WHEN** 用户扫描未过期且签名有效的匹配邀请
- **THEN** 服务端 SHALL 返回可信的对手预览信息
- **AND** 后续开局 SHALL 绑定邀请中解析出的对手 ID

#### Scenario: 修改二维码中的对手 ID
- **WHEN** 攻击者修改二维码正文、替换对手 ID 或伪造时间戳
- **THEN** 服务端 MUST 拒绝预览或开局
- **AND** MUST NOT 根据客户端昵称或头像创建对局

#### Scenario: 邀请已过期
- **WHEN** 扫描凭据超过有效期
- **THEN** 客户端 SHALL 明确提示二维码已失效并允许对方刷新二维码

### Requirement: 已接受 PK 邀约必须闭环到唯一真实对局
线上 PK 邀约被接受后，任一参与者通过线下扫码开局时 SHALL 携带并校验 `challenge_id`、双方身份和球种；成功创建的真实对局 MUST 原子关联该邀约，重复操作 MUST 返回同一对局或稳定的已关联结果。

#### Scenario: 接受邀约后线下扫码
- **WHEN** 邀约双方、球种和扫码对手均与已接受邀约一致
- **THEN** 服务端 SHALL 创建真实对局并原子写入邀约的 `match_id`
- **AND** 双方客户端 SHALL 能进入该对局

#### Scenario: 邀约与扫码对手不一致
- **WHEN** `challenge_id` 的另一参与者或球种与扫码结果不一致
- **THEN** 服务端 MUST 拒绝创建对局
- **AND** 原邀约 MUST 保持未关联

#### Scenario: 双方同时尝试开局
- **WHEN** 双方几乎同时使用同一已接受邀约创建对局
- **THEN** 系统 MUST 最多创建一个关联对局
- **AND** 两个客户端 SHALL 收敛到同一 `match_id`

### Requirement: 不支持的运行时必须显式降级
若某一构建目标无法提供扫码、文件保存或其他关键平台能力，客户端 MUST 在进入操作前隐藏入口或展示明确说明；不得保留点击后无响应的入口，也不得谎称流程成功。

#### Scenario: 构建目标不支持相机扫码
- **WHEN** 运行时能力检测确认无法扫码
- **THEN** 页面 SHALL 禁用或隐藏扫码操作并说明该平台暂不支持
- **AND** SHALL 保留返回或使用受支持设备的指引

#### Scenario: 相机权限被拒绝
- **WHEN** 平台支持扫码但用户拒绝相机权限
- **THEN** 客户端 SHALL 说明权限用途并提供重新授权或退出操作
- **AND** MUST NOT 把权限拒绝显示为二维码无效

### Requirement: 跨端关键链路必须使用同一解析与测试契约
二维码解析、类型识别、过期判断、挑战上下文和开局 payload SHALL 由共享 helper 统一生成，并 MUST 具有纯逻辑测试；发布前 SHALL 完成 App、HarmonyOS 与微信小程序真机的正向、取消、权限拒绝、过期和篡改矩阵。

#### Scenario: 不同入口扫描同一匹配码
- **WHEN** 用户从首页、对局页、排行榜或“我的”扫描同一有效匹配码
- **THEN** 各入口 SHALL 生成等价的服务端预览与开局请求

#### Scenario: 静态测试通过但平台行为不同
- **WHEN** 某平台真机矩阵出现无法扫码、无法本地渲染或上下文丢失
- **THEN** 跨端 P0 项 MUST 保持阻断状态直到该平台修复或明确下线入口
