# admin-api-access Specification

## Purpose
后台接口（`/api/admin/*`）的访问控制：管理员身份守门、公开入口白名单、路径规范化与路由器一致，以及对所有已注册后台路由的回归测试。
## Requirements
### Requirement: 后台接口必须由管理员身份访问
系统 SHALL 要求 `/api/admin/` 下的所有接口使用管理员身份访问，管理员身份指令牌中的用户 ID 为负数。普通用户令牌（正数 ID）MUST NOT 能进入任何后台 handler，无论该接口的 logic 自身是否检查身份。

#### Scenario: 普通用户令牌调用后台接口
- **WHEN** 请求带有效的普通用户令牌，访问任意 `/api/admin/` 下需要登录的接口
- **THEN** 系统 SHALL 返回 HTTP 403
- **AND** 响应体 SHALL 为 `success` 为 false 的 JSON，并带有“无管理员权限”的提示
- **AND** 请求 MUST NOT 进入该接口的 handler

#### Scenario: 管理员令牌调用后台接口
- **WHEN** 请求带有效的管理员令牌，访问 `/api/admin/` 下的接口
- **THEN** 系统 SHALL 把请求交给该接口的 handler，行为与之前一致

#### Scenario: 没有令牌
- **WHEN** 请求不带令牌，访问 `/api/admin/` 下需要登录的接口
- **THEN** 系统 SHALL 返回 HTTP 401

#### Scenario: 被禁用的普通用户
- **WHEN** 已被禁用的普通用户用自己的令牌访问后台接口
- **THEN** 系统 SHALL 拒绝该请求，且 MUST NOT 进入后台 handler

### Requirement: 后台公开入口保持匿名可访问
`/api/admin/login`、`/api/admin/init`、`/api/admin/exists` SHALL 保持匿名可访问，并且不受访问控制影响。除这三个入口外，`/api/admin/` 下的路径 MUST NOT 匿名访问。新增公开入口 SHALL 通过显式加入白名单的方式声明。

#### Scenario: 匿名登录
- **WHEN** 未登录的请求访问 `/api/admin/login`
- **THEN** 系统 SHALL 把请求交给登录 handler

#### Scenario: 请求头里带着普通用户令牌访问公开入口
- **WHEN** 请求头里带着普通用户令牌，访问 `/api/admin/login`、`/api/admin/init` 或 `/api/admin/exists`
- **THEN** 系统 SHALL 照常处理，MUST NOT 因为令牌身份返回 403

#### Scenario: 新增了未声明的后台路径
- **WHEN** 后端新增了一个 `/api/admin/` 下的路径，且没有加入公开白名单、也没有要求登录
- **THEN** 匿名请求访问该路径 SHALL 返回 HTTP 401

### Requirement: 路径判断与路由器保持一致
系统 SHALL 在判断请求是否属于 `/api/admin/` 之前，对请求路径做与路由器相同的规范化。多余的斜杠、`.` 和 `..` 段 MUST NOT 让后台接口绕开访问控制。

#### Scenario: 多余斜杠
- **WHEN** 普通用户令牌访问 `/api//admin/users/list`
- **THEN** 系统 SHALL 返回 HTTP 403，MUST NOT 进入 handler

#### Scenario: 点段
- **WHEN** 普通用户令牌访问 `/api/admin/./users/list` 或 `/api/x/../admin/users/list`
- **THEN** 系统 SHALL 返回 HTTP 403，MUST NOT 进入 handler

#### Scenario: 非后台路径不受影响
- **WHEN** 普通用户令牌访问 `/api/user/info` 等非后台接口
- **THEN** 系统 SHALL 按原有逻辑处理，MUST NOT 因后台访问控制返回 403

### Requirement: 全部已注册的后台路由有回归测试覆盖
系统 SHALL 有自动化测试：启动真实的服务，遍历所有已注册的 `/api/admin/` 路由，逐一验证普通用户令牌被拒绝、匿名请求被拒绝（公开入口除外）。后台新增路由 MUST 自动纳入该测试。

#### Scenario: 新增后台路由漏了访问控制
- **WHEN** 有人新增了一个 `/api/admin/` 路由，而访问控制没有覆盖它
- **THEN** 该回归测试 SHALL 失败
