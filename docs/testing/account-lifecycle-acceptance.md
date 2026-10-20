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

检查结果：______（附文件所在位置，不要提交到仓库）

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

## 4. 结论

| 角色 | 姓名 | 日期 | 结论 |
| --- | --- | --- | --- |
| 测试人 |  |  |  |
| 发布负责人 |  |  |  |
