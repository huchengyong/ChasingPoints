# WST Image Mirroring Specification

## Purpose

定义 WST 球员头像和赛事封面在同步阶段安全、幂等地镜像到自有七牛 CDN，并确保微信小程序只使用可访问的自有图片资源或包内默认图。

## Requirements

### Requirement: 系统仅镜像允许的 WST 图片
系统 SHALL 仅在 WST 同步过程中镜像球员头像和赛事封面，并 MUST 拒绝非 HTTPS、非 `images.gc.wstservices.co.uk` 主机、非允许图片类型或超过大小上限的来源。

#### Scenario: 镜像 WST 球员头像
- **WHEN** WST 球员记录包含合法的官方头像 URL
- **THEN** 系统 SHALL 将该图片作为球员头像镜像到自有七牛存储

#### Scenario: 镜像 WST 赛事封面
- **WHEN** WST 赛事页面解析出合法的真实封面 URL
- **THEN** 系统 SHALL 将该图片作为赛事封面镜像到自有七牛存储

#### Scenario: 拒绝任意第三方来源
- **WHEN** 镜像候选 URL 使用其他主机、非 HTTPS 协议或不允许的内容类型
- **THEN** 系统 MUST 拒绝下载和上传该资源，且不得将该能力暴露为任意 URL 代理

### Requirement: 镜像对象具有确定性和幂等性
系统 SHALL 按图片类别和规范化来源 URL 派生确定性七牛对象键，并在对象已存在时直接复用对应公开 URL。

#### Scenario: 首次镜像图片
- **WHEN** 来源 URL 对应的确定性对象尚不存在
- **THEN** 系统 SHALL 下载、校验并上传图片，然后返回自有七牛公开 URL

#### Scenario: 重复同步同一图片
- **WHEN** 来源 URL 对应的确定性对象已经存在
- **THEN** 系统 SHALL 复用已有对象 URL，且不得再次下载或上传该图片

#### Scenario: 来源 URL 发生变化
- **WHEN** 同一球员或赛事提供了新的合法图片 URL
- **THEN** 系统 SHALL 为新来源生成新的确定性对象键并更新业务记录

### Requirement: WST 同步只持久化可用的自有图片 URL
系统 SHALL 将本次镜像成功的七牛 URL 写入现有球员头像和赛事封面字段，并 MUST NOT 将 `images.gc.wstservices.co.uk` URL 作为公开赛讯图片返回给客户端。

#### Scenario: 镜像成功后写入业务记录
- **WHEN** 球员头像或赛事封面镜像成功
- **THEN** 系统 SHALL 将对应的自有七牛公开 URL 写入现有数据库 URL 字段

#### Scenario: 镜像失败但已有自有图片
- **WHEN** 本次镜像失败且当前记录已有属于当前七牛公开域名的图片 URL
- **THEN** 系统 SHALL 保留已有自有图片 URL

#### Scenario: 镜像失败且没有自有图片
- **WHEN** 本次镜像失败且当前记录没有可保留的自有图片 URL
- **THEN** 系统 SHALL 使用空图片值并由客户端展示应用自有默认图，而不得回退到 WST URL

#### Scenario: 获取赛讯详情
- **WHEN** 客户端获取包含 WST 球员或赛事数据的赛讯详情
- **THEN** 图片字段 SHALL 为当前七牛公开域名 URL 或空值，不得包含 `images.gc.wstservices.co.uk`

### Requirement: 图片镜像不扩大同步事务和 dry-run 副作用
系统 SHALL 在数据库事务开始前完成正常同步所需的图片镜像，并 SHALL 保持 dry-run 不修改七牛存储或数据库。

#### Scenario: 执行正常同步
- **WHEN** WST 同步以非 dry-run 模式执行
- **THEN** 系统 SHALL 在进入数据库写事务前完成图片对象检查及必要上传

#### Scenario: 执行 dry-run
- **WHEN** WST 同步以 dry-run 模式执行
- **THEN** 系统 SHALL 只计算同步摘要，不得检查、下载、上传七牛对象或修改数据库

#### Scenario: 单张图片镜像失败
- **WHEN** 某张允许范围内的图片下载、校验或上传失败
- **THEN** 系统 SHALL 记录不含凭证的错误并按图片回退规则继续同步其他业务数据

### Requirement: 现有 WST 图片可通过标准同步回填
系统 SHALL 使用现有 WST 同步入口回填历史球员头像和赛事封面，不要求数据库迁移或独立图片回填接口。

#### Scenario: 重跑已有 WST 赛季
- **WHEN** 运维对已有赛季执行完整非 dry-run WST 同步
- **THEN** 系统 SHALL 镜像该同步范围内的合法图片并更新对应现有记录

#### Scenario: 回填后再次自动同步
- **WHEN** 自动同步再次处理已经回填的球员和赛事
- **THEN** 系统 SHALL 复用已有七牛对象并保持业务 URL 稳定

### Requirement: 微信小程序直接使用自有图片资源
移动端 SHALL 直接渲染 API 返回的自有 CDN 图片 URL，并 SHALL 使用包内静态资源作为缺失赛事封面的默认图，不得依赖从 WST 域名下载并保存到本地的绕过流程。

#### Scenario: 展示已镜像图片
- **WHEN** 赛讯详情返回自有七牛头像或封面 URL
- **THEN** 微信小程序 SHALL 直接从已配置的七牛下载合法域名加载图片

#### Scenario: 展示缺失封面
- **WHEN** 赛讯详情没有可用赛事封面 URL
- **THEN** 微信小程序 SHALL 展示包内自有默认赛事封面

#### Scenario: 发布微信小程序
- **WHEN** 包含该能力的微信小程序版本准备发布
- **THEN** 七牛公开域名 MUST 已配置为微信小程序下载合法域名并通过真机 200 响应验证
