# R2.2：设置、编辑资料、通知管理（对应小程序 `subPages/user/settings`、`editProfile`、`notification`）

你在一个 git worktree 的根目录里工作（当前目录就是仓库根）。总目标是把微信小程序（`uniapp/`）几乎 1:1 迁移到原生 SwiftUI（`iOS/`）。骨架、共享组件和「我的」主页已经迁完。本轮迁移三个页面：设置、编辑资料、通知管理。

先读：`AGENTS.md`、`iOS/AGENTS.md`、`DESIGN.md`；`iOS/iOS/` 下全部 Swift 文件（重点 `SessionStore.swift`、`SettingsView.swift`、`ProfileEditView.swift`、`LoginView.swift`、`App/`、`Shared/`、`Features/Profile/`）；`iOS/iOSAuthTests/` 的 `TestSupport.swift`、`MockHTTPServer.swift`、`AuthFixtures.swift`、`SessionStoreProfileTests.swift`、`SnapshotSupport.swift`、`MyPageSnapshotTests.swift`；`openspec/specs/native-ios-profile-basics/spec.md`（既有需求，不能违反）。
小程序侧：`uniapp/subPages/user/settings.vue` + `settings.scss`、`editProfile.vue` + 同目录样式、`notification.vue` + `notification.scss`、`uniapp/utils/settings-profile.js`、`uniapp/api/user.js`、`uniapp/api/notification.js`、`uniapp/tests/settings-profile.test.mjs`（如存在）。
接口契约（`backend/chasing_points.api`）：`GetUserPrivacyResp`、`UpdateUserPrivacyReq/Resp`、`DeleteAccountReq`、`SendSmsReq`（`scene`）、`GetQiniuUploadTokenReq/Resp`、`UpdateUserProfileReq/Resp`、`NotificationPreferencesResp`、`SaveNotificationPreferencesReq`。

## 硬性约束（违反任何一条都算失败）

1. 只允许新增/修改 `iOS/iOS/**` 与 `iOS/iOSAuthTests/**`。不得修改 `iOS/iOS.xcodeproj/**`、`uniapp/**`、`backend/**`、`openspec/**` 及其他目录。
2. 不执行任何 git 写操作，只允许 `git status`、`git diff` 等只读命令。
3. 不联网、不 ssh、不 curl；不读取任何 `.env`、密钥、Keychain、真实会话；不访问当前目录之外的路径（Xcode 默认 DerivedData 除外）。**绝不**对真实后端发注销、短信、上传请求——所有验证只用 `MockHTTPServer`。
4. 不引入第三方依赖。最低部署目标 iOS 17.0。
5. 现有全部测试必须继续通过；`SessionStore` 现有方法的行为不变（只新增方法 / 给 `sendSMS` 加带默认值的参数）。昵称编辑页 `NicknameEditView` 的界面和行为不变。
6. 颜色走 `AppTheme` / `AppPalette`；文案与小程序逐字一致（本规格明确说明的差异除外）。
7. 不创建 Markdown / 文档文件。
8. **并行开发提醒**：另一条开发线在修改 `iOS/iOS/Shared/ResourceLoader.swift`、`iOS/iOS/Shared/UI/LoadableContent.swift`、`iOS/iOS/Features/Rules/**`、`iOS/iOS/Features/Ranking/**`，本轮不要动这些文件；`AppRoute.swift` 只做本规格点名的改动。

## 工程须知

- App target 默认 MainActor 隔离、Swift 5 语言模式；测试类标 `@MainActor`。
- 写请求用 `session.authorizedPost`（请求体 snake_case）。可选字段用 Swift 可选类型，`nil` 不会被编码进请求体——`UpdateUserPrivacyReq` 正是靠「只带其中一个字段」来区分改哪一项。
- 所有异步结果都要绑定发起时的 `session.identity`：退出或切号后到达的结果不得改动界面状态、不得弹提示。
- rpx 换算大致 `pt ≈ rpx / 2`。
- 快照测试的固定做法：页面的 `init` 接受一个可选的页面模型（`init(..., model: XxxModel? = nil)`，内部 `_model = StateObject(wrappedValue: model ?? XxxModel(...))`）；测试里先创建模型、`await` 它读完数据，再传给页面渲染。不要指望页面内的 `.task` 在快照里把网络请求跑完，也不要为此改 `SnapshotSupport.swift` / `MockHTTPServer.swift` 或写探针测试排查（参照 `MyPageSnapshotTests.swift`）。
- 遇到与任务无关的疑难，最多尝试两种办法；不行就换一种实现继续，并在交付说明里写明。
- bash 工具跑 xcodebuild 时超时设为 600000 毫秒，输出必须用 grep 过滤。

## 任务

### 0. 修正一个全局显示问题（数字被加了千分位）

「我的」页段位卡现在把排位分显示成 `排位分 1,320`（带千分位）。原因是 SwiftUI 的 `Text("排位分 \(score)")` 会按系统区域格式化整数；小程序显示的是 `排位分 1320`。把 `iOS/iOS/` 下所有把**整数**直接插进 `Text("...\(n)...")` 的地方改成先 `String(n)` 再拼（或 `Text(verbatim:)`）。已知位置：`Features/Profile/MyPageSections.swift`（排位分）、`Features/Venue/VenueListView.swift`（`N人签到`）、`Features/Venue/VenueDetailView.swift`（`N人到访`）、`Features/Rules/RuleDetailView.swift`（序号）、`MatchHistoryViews.swift`（两处比分）、`MatchReportView.swift`（`Text("\(score)")`）。这是本轮唯一允许改动这些文件的地方，每处只改这一行。检查方法：`grep -rn 'Text("[^"]*\\(' iOS/iOS --include='*.swift'` 逐条确认插值的是字符串还是整数。加一个回归测试不现实的地方不用强求，但要在 `MyPageSnapshotTests` 的夹具里保持四位数排位分，方便人工看快照。本轮新写的页面同样遵守这条规则。

### 1. 会话层（修改 `iOS/iOS/SessionStore.swift`）

- `sendSMS(phone:)` 改为 `sendSMS(phone: String, scene: String = "login")`，请求体的 `scene` 用传入值。
- 新增：

```swift
/// 更新头像：提交当前昵称与新头像地址，成功且返回匹配当前身份的完整资料后回写共享会话（规则同 updateNickname）。
func updateAvatar(_ avatarURL: String) async throws

/// 注销账号（短信复核）。成功后清理本机会话（等同 logout，不弹「会话失效」提示）。
func deleteAccount(smsCode: String) async throws
```

  - `updateAvatar`：`POST /api/user/profile`，请求体 `{ "nickname": 当前昵称, "avatar": avatarURL }`。
  - `deleteAccount`：`POST /api/user/account/delete`，请求体 `{ "confirm_text": "注销账号", "verify_type": "sms", "sms_code": smsCode }`；响应按 `{ success, message }` 解码。失败时会话保持不变并抛出错误。请求期间身份变化则抛「登录状态已变更」且不清理新会话。

### 2. 设置页（重写 `iOS/iOS/SettingsView.swift`，区块可拆到 `iOS/iOS/Features/Settings/`）

对应 `settings.vue`。一张列表卡 + 底部「退出登录」。行的顺序、标题、副标题照搬模板：

1. 主题模式（三段：跟随系统 / 浅色 / 深色，沿用 `@AppStorage("theme_mode")`）
2. 通知管理 → `.notificationSettings`
3. 隐藏战绩（副标题「控制是否在公开场景展示你的战绩」，右侧开关）
4. 黑名单（副标题「查看并移出已拉黑的用户」）→ `.blacklist`
5. 仅允许好友约球（副标题「开启后，非好友无法向你发送约球邀请」，右侧开关）
6. 隐私政策、7. 用户协议：沿用现有的两个外链（`https://www.zhuifen.cn/privacy`、`/agreement`）
8. 帮助、投诉与举报（副标题「提交问题建议、投诉举报或获取帮助」）→ `.feedback`
9. 常用球种（副标题「「我的」段位卡优先展示，发起 PK 时自动预选」）：五个选项「每次询问」+ 四个球种，用 `GameTypePreference` 读写；切换后 Toast「常用球种已更新」/「已改为每次询问」
10. 注销账号（危险色，副标题「删除个人资料与关系，此操作不可恢复」）
11. 底部按钮「退出登录」：沿用现有确认弹窗与文案，确认后另加 Toast「已退出登录」

**访客**只显示：主题模式、隐私政策、用户协议、帮助、投诉与举报（既有规格要求访客能用主题和协议）。其余行与退出按钮不显示。

隐私开关（对应 `loadUserPrivacy` / `toggleHideMatch` / `toggleFriendsOnlyChallenges`），逻辑放进 `@MainActor final class PrivacySettingsModel: ObservableObject`：

- `GET /api/user/privacy` 读取；读取中开关位置显示「读取中…」，失败显示「读取失败，点击重试」，点击该行重读。
- 切换「隐藏战绩」：先乐观更新，`POST /api/user/privacy` 只带 `hide_match_record`；成功以服务端返回值为准并 Toast「已隐藏战绩」/「已公开战绩」；失败回滚并 Toast 错误文案（没有文案时「更新失败」）。
- 切换「仅允许好友约球」：不做乐观更新，只带 `friends_only_challenges`；成功 Toast「已开启仅好友约球」/「已允许非好友约球」；失败 Toast「保存失败，请重试」。
- 同一项保存中再次点击无效；身份变化后重置并重读。
- Toast 文案由 model 的方法返回给视图去显示（model 不直接依赖 `ToastCenter`），便于测试。

注销账号（对应模板里的 `deletion-dialog`，原生端用 `.sheet`），逻辑放进 `@MainActor final class DeleteAccountModel: ObservableObject`：

- 第一步「注销账号」：说明文案「注销后，账号资料、认证关联和社交数据会被删除；必要的历史比赛会匿名保留。此操作不可恢复。」，按钮「取消」/「我已知晓，继续」。
- 第二步「身份复核」：说明「请输入已绑定的完整手机号并完成注销验证码验证。」，手机号输入框（占位「已绑定的完整手机号」）、验证码输入框（占位「注销验证码」）+「获取验证码」（发送中显示「发送中…」），按钮「取消」/「确认注销」（提交中显示「处理中…」）。
- 获取验证码：手机号不符合 `^1[3-9]\d{9}$` → 提示「请输入已绑定的完整手机号」；否则 `session.sendSMS(phone:scene: "delete_account")`，成功提示「注销验证码已发送」，失败提示错误文案（没有时「验证码发送失败」）。
- 确认注销：手机号或 6 位验证码不合规 → 提示「请填写完整手机号和注销验证码」；否则 `session.deleteAccount(smsCode:)`，成功后关闭弹层、Toast「账号已注销」（会话已清理，导航会自动回到根页面）；失败提示错误文案（没有时「注销失败，请重试」），停留在当前步骤。提交中不可重复提交、不可关闭。
- **平台差异**：原生端只有手机号登录，不做第三方授权复核。当前账号没有手机号时，第二步只显示「当前账号未绑定手机号，请在微信小程序中完成注销。」和「取消」。

### 3. 编辑资料（新建 `iOS/iOS/Features/Profile/ProfileEditView.swift`；现有 `iOS/iOS/ProfileEditView.swift` 里的 `NicknameEditView` 不动）

对应 `editProfile.vue`。新增路由 `case editProfile`（标题「编辑资料」）→ `ProfileEditView`；`.editNickname` 的标题改为「编辑昵称」。「我的」页资料头的点击目标从 `.editNickname` 改为 `.editProfile`（只改这一处）。

- 顶部：头像（96pt，`AvatarView`）+ 右下角相机角标；提示文字「点击更换头像」，上传中显示「上传中...」。点击头像用系统 `PhotosPicker`（只选 1 张图片）。**平台差异**：不提供「拍照」（需要相机权限，后续统一处理）。
- 资料卡两行：「昵称」→ 右侧当前昵称 + 箭头 → `.editNickname`；「手机号」→ 右侧 `formatSettingsPhone` 的结果，没有手机号显示「未绑定」，不可点击（原生端账号必有手机号，不做绑定流程）。
- 头像上传流程（对应 `uploadAvatar`），放进 `@MainActor final class AvatarUploadModel: ObservableObject` + 一个可注入网络的 `AvatarUploader`：
  1. 图片处理：最长边缩到不超过 512px（不放大），编码为 JPEG，质量 0.8（对应 `AVATAR_IMAGE_MAX_EDGE` / `AVATAR_IMAGE_QUALITY`）。缩放尺寸的计算照搬 `getAvatarCompressionDimensions`。
  2. `GET /api/user/upload-token?file_ext=.jpg`（走 `session.authorizedRequest`）；响应字段 `upload_token`、`key`、`upload_url`、`domain`。`upload_url` 为空时用 `https://up-z2.qiniup.com`；`key` 为空时按 `buildQiniuAvatarObjectKey` 生成（`avatars/<userId>/<yyyyMMdd>/<毫秒时间戳>.jpg`）。`upload_token` 为空 → 失败。
  3. 向 `upload_url` 发 `multipart/form-data` 的 POST：表单字段 `token`、`key`，文件字段名 `file`（文件名用 key 的最后一段，`Content-Type: image/jpeg`）。这是发往七牛的第三方请求，不带 Authorization；用注入的 `URLSession`（默认 `.shared`）。HTTP 非 2xx 视为失败。响应 JSON 里的 `key` 非空时以它为准。
  4. 头像地址 = `normalizeQiniuUploadResult(domain, key)`（去掉 domain 末尾斜杠和 key 开头斜杠后用 `/` 连接；任一为空 → 失败「头像地址生成失败」）。
  5. `session.updateAvatar(url)`；成功 Toast「头像更新成功」；任何一步失败 Toast 错误文案（没有时「头像上传失败」）。
  6. 上传中再次选择无效；上传期间退出登录或切号，后续步骤全部中止且不弹提示。
- 纯函数迁移（新建 `iOS/iOS/Features/Profile/SettingsProfileRules.swift`）：`formatSettingsPhone`、`getFileExtensionFromPath`、`buildQiniuAvatarObjectKey`、`normalizeQiniuUploadResult`、`getAvatarCompressionDimensions`，逐条对应 `settings-profile.js`，并把 `uniapp/tests/` 里对应的用例搬过来（搜 `settings-profile` 找测试文件）。

### 4. 通知管理（新建 `iOS/iOS/Features/Settings/NotificationSettingsView.swift`）

对应 `notification.vue`，`.notificationSettings` → 该页面。

- 开关项、标题、顺序、哪些项受 `canShowSocialPublishing()` 控制，全部以 `notification.vue` 模板为准（原生端对应 `AppFlags.canShowSocialPublishing`）。
- 读取 `GET /api/notification/preferences`：加载中显示「正在加载通知偏好...」；失败显示错误与重试。
- 切换任一开关：按 `togglePreference` / `buildPreferencesPayload` 的做法提交**全部五个字段**到 `POST /api/notification/preferences`；保存中所有开关禁用；成功以服务端返回为准；失败回滚到保存前的值并提示（文案照搬页面里的 toast）。
- 逻辑放进 `@MainActor final class NotificationPreferencesModel: ObservableObject`。

## 必须新增的测试

- `SessionStoreAccountTests.swift`：`sendSMS` 默认 scene 是 `login`、可传 `delete_account`；`updateAvatar` 请求体含当前昵称与头像、成功回写 `session.user.avatar`、返回的 id 不匹配时不回写并抛错；`deleteAccount` 请求体三个字段正确、成功后会话与凭据清空且 `sessionExpiredNotice == nil`、失败时会话不变、请求中退出再登录不清理新会话。
- `PrivacySettingsModelTests.swift`：读取成功 / 失败重试；隐藏战绩的乐观更新、成功以服务端为准、失败回滚；仅好友约球不乐观更新；请求体只含被修改的那个字段（断言另一个键不存在）；保存中重复点击只发一次请求；退出后迟到结果不改状态。
- `DeleteAccountModelTests.swift`：两步流转；手机号 / 验证码校验文案；发送验证码的 scene；成功后会话清空；失败停留并可重试；提交中不可重复提交。
- `AvatarUploadTests.swift`：缩放尺寸计算；对象 key 生成（固定时间戳）；multipart 请求体包含 `token`、`key`、文件字段且不带 Authorization；默认上传地址；token 为空 / 上传返回 500 / domain 为空各自失败且不调用资料更新；完整成功路径回写头像；上传中退出登录后不再调用 `/api/user/profile`。
- `SettingsProfileRulesTests.swift`：搬运 JS 用例。
- `NotificationPreferencesModelTests.swift`：读取；切换后提交全部五个字段；失败回滚；保存中禁用。
- `SettingsSnapshotTests.swift`（亮暗各一张）：`settings-signed-in`（隐私已读到）、`settings-guest`、`delete-account-verify`（注销第二步的弹层内容直接渲染，不经 sheet）、`profile-edit`、`notification-settings`。

## 自检（在 `iOS/` 目录执行，超时 600000 毫秒，输出必须过滤）

```bash
cd iOS
xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "error:|BUILD (SUCCEEDED|FAILED)" | head -60

xcodebuild test -project iOS.xcodeproj -scheme iOS -configuration Debug \
  -destination "platform=iOS Simulator,id=1A59DDBE-8E27-488F-864C-E7D44C52B490" \
  -parallel-testing-enabled NO 2>&1 | grep -E "error:|: error|failed|Executed [0-9]+ tests|TEST (SUCCEEDED|FAILED)" | tail -60

xcodebuild build -project iOS.xcodeproj -scheme iOS -configuration Release \
  -destination 'generic/platform=iOS Simulator' CODE_SIGNING_ALLOWED=NO 2>&1 | grep -E "error:|BUILD (SUCCEEDED|FAILED)" | head -20
```

要求：Debug 与 Release 构建成功；全部测试通过。测试失败就修到通过，不允许靠删除、跳过或弱化测试来通过。

## 交付

完成后用中文给出：新增/修改的文件清单、测试总数与结果（贴出 `Executed N tests` 那一行）、与本规格或小程序不一致的地方及原因。没做到的写明没做到。
