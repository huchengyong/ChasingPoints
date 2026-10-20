# 个人数据导出与账号注销验收记录（P0 任务 8.6）

> 对应 OpenSpec 变更 `fix-client-p0-release-blockers` 的任务 8.6。只使用专门的测试账号。注销会修改数据，生产环境执行前需确认授权。

## 1. 自动化覆盖（已通过）

- `internal/logic/user/export_personal_data_logic_test.go`：导出内容与分段行为（含不得输出认证秘密）。
- `internal/logic/user/delete_account_logic_test.go`：注销后用户记录标记为已注销、直接身份字段清空、OAuth 关联删除、历史对局去标识化、进行中对局收口、旧 token 失效。
- `go test ./...` 全量通过（2026-10-20）。

自动化只能证明测试覆盖的路径。以下人工抽检在真实构建和真实数据上进行。

## 2. 导出文件抽检

测试账号在设置页发起导出，保存文件后逐项检查：

- [ ] 文件包含数据分类、生成时间和格式版本
- [ ] 不包含 access token、refresh token
- [ ] 不包含短信验证码
- [ ] 不包含 Provider 授权凭据、完整 OpenID / UnionID、推送 token
- [ ] 涉及他人（对手、好友）的内容只有公开展示信息，没有对方手机号或认证标识
- [ ] 中断导出（断网）时，App 显示失败并允许重试，没有生成宣称成功的残缺文件

检查结果：发布负责人于 2026-10-20 确认已人工查看导出文件（本人口头确认，未提供逐项勾选结果；断网中断导出的场景未单独确认）。导出文件不提交到仓库。

## 3. 注销前后快照

注销前，对测试账号（ID 记为 `<TEST_USER_ID>`）记录以下信息。SQL 为示例，列名以 `backend/internal/model` 与现有测试为准，执行前先在只读副本上确认。

```sql
SELECT id, status, deleted_at, nickname, avatar, phone, push_token FROM users WHERE id = <TEST_USER_ID>;
SELECT COUNT(*) AS oauth_links FROM user_oauth WHERE user_id = <TEST_USER_ID>;
SELECT id, status, opponent_name, remark FROM matches WHERE id IN (<该账号参与的对局 ID>);
SELECT COUNT(*) AS unfinished FROM matches WHERE id IN (<该账号参与的对局 ID>) AND status = <进行中状态值>;
```

注销后再次执行同样的查询，核对：

- [ ] `users`：`deleted_at` 不为空，直接身份字段（昵称、头像、手机号、推送 token）已清空
- [ ] `user_oauth`：该用户的关联记录为 0
- [ ] 历史对局：比分和时间保留，昵称、头像、手机号、认证标识不再出现在展示数据中
- [ ] 没有残留的进行中对局
- [ ] 注销前签发的 token 调用受保护接口返回 `SESSION_INVALID` 或等价的失效结果
- [ ] 使用已删除的 OAuth 身份重新登录，不会恢复原账号

## 3.1 执行记录（2026-10-20，测试环境，测试账号 ID 3）

自动化执行结果（只记录字段是否为空与数量，不记录具体值）：

| 项目 | 注销前 | 注销后 | 结论 |
| --- | --- | --- | --- |
| 账号存在 | 是 | 是 | 记录保留并标记注销 |
| 状态 / 删除时间 | 正常 / 未设置 | 已注销 / 已设置 | 通过 |
| 手机号 | 有 | 已清空 | 通过 |
| 昵称 | 有 | 已替换为统一占位（`DeletedUserDisplayName`） | 通过 |
| 头像 | 无 | 无 | 通过（注销前即为空） |
| 推送 token | 无 | 无 | 通过 |
| OAuth 关联 | 0 | 0 | 通过 |
| 对局（作为参赛者或对手） | 0 | 0 | 未覆盖，该账号无对局 |
| 旧 token 调用受保护接口 | — | 401，`SESSION_INVALID` | 通过 |

导出（注销前）：导出内容的顶层结构为 `success`、`items` 等；自动扫描未发现 token、验证码、第三方标识字段，也未出现当前 token 的值。人工抽检仍待完成。

生产环境同日用生产测试账号（ID 1）做了同样的注销（未做注销前快照，只核对注销后状态）：注销接口返回成功，旧 token 返回 401；数据库中该账号状态为已注销，手机号与推送 token 已清空，没有第三方绑定。

未做真实环境验证的部分（测试账号没有对局记录，也没有 OAuth 绑定）：
- 历史对局去标识化、进行中对局收口。
- 用已删除的 OAuth 身份重新登录。

这两部分由 `TestDeleteAccountOAuthClearsPersonalDataAndAnonymizesFacts`（`internal/logic/user/delete_account_logic_test.go`）覆盖：进行中的对局被关闭（状态置为取消、备注为 `account_deleted`）且对手名改为统一占位；对手快照与对局投影中的昵称、头像被去标识化；OAuth 关联被删除；重复注销收敛为成功；同一 OAuth 身份再次登录得到的是新账号，不会恢复原账号。发布负责人于 2026-10-20 确认以单元测试作为这两部分的证据。

## 4. 结论

| 角色 | 姓名 | 日期 | 结论 |
| --- | --- | --- | --- |
| 测试人 |  |  |  |
| 发布负责人 |  |  |  |
