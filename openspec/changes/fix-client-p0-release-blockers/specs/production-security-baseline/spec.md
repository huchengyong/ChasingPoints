## ADDED Requirements

### Requirement: 生产 HTTP CORS 必须使用明确允许列表
后端生产环境 MUST 从配置读取允许的 HTTP Origin，并 SHALL 拒绝通配符 `*`、空配置的隐式全放行和未登记来源。开发环境 MAY 使用单独的本地允许列表，但不得影响生产默认值。

#### Scenario: 生产配置包含通配符
- **WHEN** `APP_ENV` 为生产且 CORS 允许列表包含 `*`
- **THEN** 服务 MUST 拒绝启动或使发布检查失败

#### Scenario: 已登记客户端或管理来源
- **WHEN** 请求 Origin 与配置中的正式来源精确匹配
- **THEN** 服务端 SHALL 返回适用的 CORS Header
- **AND** 不得自动反射任意请求 Origin

#### Scenario: 来源未登记
- **WHEN** 浏览器请求来自未登记 Origin
- **THEN** 服务端 MUST 不授予跨域访问

### Requirement: 发布签名材料必须迁出仓库并完成轮换
仓库内的 Manifest、源码、示例配置和历史可提交文件 MUST NOT 包含非空签名密码、私钥、证书、profile 或机器绝对路径。所有曾进入仓库的现有发布签名材料 SHALL 在下一次正式发布前完成轮换，并由受控的本地或 CI 密钥存储注入。

#### Scenario: 扫描当前 Harmony 签名配置
- **WHEN** 发布门禁扫描 `app/manifest.json` 和受版本控制文件
- **THEN** `keyPassword`、`storePassword`、私钥、证书和 profile 路径 MUST 为空、缺失或仅为不含秘密的占位说明

#### Scenario: 使用旧签名材料发布
- **WHEN** 构建仍引用审查时已存在于仓库的签名材料
- **THEN** 发布 MUST 被阻止
- **AND** 维护者 SHALL 提供轮换后材料已外置的确认

#### Scenario: 开发者本地构建
- **WHEN** 开发者需要签名构建
- **THEN** 签名配置 SHALL 从版本控制外的本地配置或受控密钥存储加载
- **AND** 对应私密文件模式 SHALL 被忽略和扫描

#### Scenario: HBuilderX 本地运行或打包
- **WHEN** 开发者在 HBuilderX 对 HarmonyOS 执行本地运行或打包
- **THEN** 客户端 SHALL 从被 Git 忽略的本地签名输入生成 `app/harmony-configs/build-profile.json5`
- **AND** 调试运行 SHALL 使用 `default` 签名，本地发布包 SHALL 使用 `release` 签名
- **AND** 受版本控制的 `app/manifest.json` MUST 保持不含签名材料

### Requirement: 客户端权限必须遵循最小授权
App Manifest SHALL 只声明当前发布功能确实需要的模块、系统权限和隐私权限。高敏权限 MUST 具有可追溯使用场景、运行时提示和验收证据，否则 MUST 从发布配置移除。

#### Scenario: 审查无业务依据的高敏 Android 权限
- **WHEN** 权限清单包含 `READ_LOGS`、`GET_ACCOUNTS`、`READ_PHONE_STATE`、`WRITE_SETTINGS`、存储挂载或网络状态修改等高敏能力且没有已批准用例
- **THEN** 发布门禁 MUST 失败
- **AND** 对应权限 MUST 在发布前移除

#### Scenario: 用户使用扫码
- **WHEN** 用户主动进入扫码流程
- **THEN** 客户端 MAY 请求相机权限
- **AND** 拒绝权限时 SHALL 说明用途并提供返回或重试路径

#### Scenario: 用户未使用位置功能
- **WHEN** 用户没有进入依赖位置的球房能力
- **THEN** 客户端 MUST NOT 因启动或无关页面提前请求位置权限

### Requirement: 发布品牌资源必须可复现且不与签名材料混放
Android/iOS 图标、Harmony 图标和 Harmony 启动页资源 MUST 作为受版本控制的构建输入存在于干净工作区；`manifest.json` 只引用稳定的相对资源路径。签名证书、profile 与密码 MUST NOT 与品牌资源混放，且仍 SHALL 仅通过本地或 CI 私密配置注入。

#### Scenario: 干净工作区执行本地构建
- **WHEN** 开发者在无预先生成 `unpackage` 产物的干净工作区中使用 HBuilderX 构建
- **THEN** Manifest 引用的 Android/iOS 图标文件 SHALL 存在且尺寸与声明相符
- **AND** Harmony 原生工程 SHALL 具备应用图标、启动页图标和启动背景资源

#### Scenario: 更换品牌视觉
- **WHEN** 维护者替换应用图标或 Harmony 启动页视觉
- **THEN** 只需提交版本化资源文件
- **AND** MUST NOT 修改或暴露本地签名材料

### Requirement: 二维码图像必须在受控环境生成
匹配码和裁判码 SHALL 在客户端本地离线渲染，或由受控的一方服务生成；二维码正文 MUST NOT 发送给 `api.qrserver.com` 或其他未批准的第三方图像服务，也 MUST NOT 包含长期访问令牌或不必要的个人资料。

#### Scenario: 客户端展示匹配码
- **WHEN** 服务端返回有效的二维码正文
- **THEN** 客户端 SHALL 在本地生成二维码图像
- **AND** 断开第三方网络后仍能完成渲染

#### Scenario: 静态扫描发现第三方二维码 URL
- **WHEN** 受版本控制的 App 代码重新出现 `api.qrserver.com` 或其他未批准二维码生成域名
- **THEN** 自动化发布检查 MUST 失败

#### Scenario: 二维码正文被记录
- **WHEN** 二维码生成或扫码发生错误
- **THEN** 日志 MUST NOT 输出完整邀请 token、裁判 join token 或其他敏感正文

### Requirement: P0 安全基线必须成为上线门禁
正式发布 MUST 同时通过自动化安全契约、后端测试、App 测试、App / Harmony / 微信小程序真机矩阵和人工配置核验；任一 P0 项未关闭时 MUST 不得给出上线通过结论。

#### Scenario: 自动化检查全部通过但签名未轮换
- **WHEN** 代码测试通过但没有旧签名材料已轮换和外置的确认
- **THEN** 发布状态 MUST 保持阻断

#### Scenario: 小程序扫码未经真机验证
- **WHEN** 仅有源码测试而缺少微信小程序真机扫码证据
- **THEN** 跨端 P0 门禁 MUST 视为未完成

#### Scenario: 所有 P0 证据齐全
- **WHEN** 每项 P0 均有自动化结果或明确的人工验收记录
- **THEN** 该变更 MAY 被标记为上线门禁通过
