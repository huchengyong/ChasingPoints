## ADDED Requirements

### Requirement: 入口链接使用统一格式
系统 SHALL 用统一的入口链接承载所有可扫码进入追分的二维码，基础地址为 `https://www.zhuifen.cn/go/`：
- 匹配码：`https://www.zhuifen.cn/go/?j=<短码>`
- 对局：`https://www.zhuifen.cn/go/?m=<对局 ID>`
- 首页：`https://www.zhuifen.cn/go/`（不带参数）

解析 MUST 只接受 `https` 协议、`www.zhuifen.cn` 主机和 `/go` 路径（带或不带结尾斜杠）。`j` 与 `m` 同时出现、取值格式不合法（短码不符合字符集与长度、对局 ID 不是正整数）时，链接 MUST 视为首页链接，不得据此发起任何业务请求。

#### Scenario: 解析匹配码链接
- **WHEN** 解析 `https://www.zhuifen.cn/go/?j=<合法短码>`
- **THEN** 结果 SHALL 为匹配码类型，并带出该短码

#### Scenario: 解析对局链接
- **WHEN** 解析 `https://www.zhuifen.cn/go/?m=123`
- **THEN** 结果 SHALL 为对局类型，对局 ID 为 123

#### Scenario: 参数不合法
- **WHEN** 解析 `https://www.zhuifen.cn/go/?m=abc`，或同时带 `j` 和 `m`
- **THEN** 结果 SHALL 为首页类型

#### Scenario: 其他域名或协议
- **WHEN** 解析 `https://evil.example.com/go/?j=<合法短码>` 或 `http://www.zhuifen.cn/go/?j=<合法短码>`
- **THEN** 结果 MUST 为非入口链接，不得取出短码

### Requirement: 小程序承接页按链接类型分发
微信通过「扫普通链接二维码打开小程序」规则打开承接页时，承接页 SHALL 从启动参数 `q` 中还原入口链接并分发：匹配码进入扫码即入流程；对局进入该对局的观战详情；首页或无法解析的链接进入首页。承接页 MUST NOT 把短码写入本地存储、日志或埋点，也 MUST NOT 把签名 token 放进页面路由。

#### Scenario: 冷启动打开对局链接
- **WHEN** 用户用微信扫描对局链接，小程序冷启动进入承接页
- **THEN** 承接页 SHALL 替换为该对局的观战详情页，不需要登录

#### Scenario: 打开首页链接
- **WHEN** 承接页收到不带参数的首页链接
- **THEN** 承接页 SHALL 跳转到首页

#### Scenario: 启动参数缺失或无法解析
- **WHEN** 承接页没有收到 `q`，或 `q` 无法解码为入口链接
- **THEN** 承接页 SHALL 跳转到首页，不弹出错误

#### Scenario: 小程序已在后台
- **WHEN** 小程序已在后台运行，用户用微信扫描匹配码链接
- **THEN** 承接页 SHALL 以新的启动参数打开，并按匹配码类型分发

### Requirement: 官网提供非微信扫码的兜底页
官网 SHALL 提供 `/go/` 页面，承接用系统相机或其他 App 扫描入口链接的访问者。页面 SHALL 按链接类型说明用途并引导用微信扫一扫：匹配码说明「这是追分的匹配码，请用微信扫一扫，或在追分 App 内扫码」；对局说明「用微信扫一扫查看这场比赛」；并 SHALL 提供下载入口。兜底页 MUST NOT 请求后端、MUST NOT 在页面上展示短码，且 MUST 在静态生成的产物中可用。

#### Scenario: 系统相机扫描匹配码
- **WHEN** 访问者在浏览器中打开 `https://www.zhuifen.cn/go/?j=<短码>`
- **THEN** 页面 SHALL 显示匹配码的引导文案和下载入口
- **AND** 页面 MUST NOT 显示短码，也 MUST NOT 发起后端请求

#### Scenario: 打开对局链接
- **WHEN** 访问者在浏览器中打开 `https://www.zhuifen.cn/go/?m=123`
- **THEN** 页面 SHALL 显示查看比赛的引导文案和下载入口

#### Scenario: 静态发布
- **WHEN** 官网以 `npm run generate` 静态生成并发布
- **THEN** `/go/` 及带查询参数的访问 SHALL 正常返回该页面

### Requirement: 官网下载口径与已上线渠道一致
官网首页常见问题与下载页 SHALL 只展示实际提供的渠道：微信小程序与鸿蒙。官网 MUST NOT 承诺 iOS 或 Android 下载。尚未上架的渠道 SHALL 显示「即将上线」，不得给出无法打开的下载链接。

#### Scenario: 访问下载页
- **WHEN** 访问者打开官网下载页
- **THEN** 页面 SHALL 展示微信小程序入口与鸿蒙入口
- **AND** 页面 MUST NOT 出现 iOS 或 Android 下载入口

#### Scenario: 鸿蒙尚未上架
- **WHEN** 鸿蒙应用尚未在华为应用市场上架
- **THEN** 鸿蒙入口 SHALL 显示「即将上线」，且不可点击跳转

#### Scenario: 常见问题
- **WHEN** 访问者查看首页常见问题中关于下载渠道的条目
- **THEN** 回答 SHALL 与下载页展示的渠道一致
