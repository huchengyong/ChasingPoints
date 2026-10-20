# P0 发布证据汇总（任务 8.8）

> 对应 OpenSpec 变更 `fix-client-p0-release-blockers` 的任务 8.8。只记录标识、配置状态与验证结果，不记录任何密码、token 或凭据值。任一证据缺失，P0 发布仍然阻断。

## 1. 状态

部分完成。以下 1–4 项已有证据；5 项（三端真机证据）由 `device-matrix-acceptance.md` 承接，尚未填写。

## 2. 正式允许 Origin

来源：`backend/etc/chasing_points-api.yaml` 的 `Security` 段（线上配置与仓库一致）。

- HTTP：`https://www.zhuifen.cn`、`https://api.zhuifen.cn`、`https://ws.zhuifen.cn`、`https://admin.zhuifen.cn`
- WebSocket：`https://www.zhuifen.cn`、`https://api.zhuifen.cn`、`https://ws.zhuifen.cn`
- `AllowNativeMissingOrigin: true`：原生客户端在 TLS 下可不带 Origin，见 `fail-closed-verification.md`。

线上只读验证结果见 `fail-closed-verification.md` 第 3 节。

注意：yaml 的 `HTTPAllowedOrigins` 和 `WebSocketAllowedOrigins` 中还保留着 `https://admin-zhuifen.kekemate.cn` 与 `https://api-zhuifen.kekemate.cn` 两个测试环境来源。它们只出现在测试和生产共用的配置里，发布前应确认生产是否需要保留。

## 3. 华为服务端配置

| 项目 | 测试环境 | 生产环境 |
| --- | --- | --- |
| `ZHUIFEN_HM_OAUTH_ENABLED` | 已设置为 true（2026-10-20） | 已设置为 true（2026-10-20） |
| `ZHUIFEN_HM_OAUTH_CLIENT_ID` | 已设置（来自本机配置，与 yaml 中的 `ClientID` 一致） | 同左 |
| `ZHUIFEN_HM_OAUTH_CLIENT_SECRET` | 已设置（值不记录） | 已设置（值不记录） |
| `VerifyURL` | yaml 中为 `https://oauth-login.cloud.huawei.com/oauth2/v3/tokeninfo` | 同左 |
| 失败关闭检查 | 假凭据返回 400 `Huawei rejected OAuth credential` | 同左 |

待确认：`CLIENT_SECRET` 是否为本次轮换后的值（华为侧的 secret 是否重置过）。

## 4. Manifest 权限用途

App（`uniapp/manifest.json`）：

| 权限 | 用途 | 状态 |
| --- | --- | --- |
| `CAMERA` | 扫码（匹配码、裁判码） | 有用途，需真机验证 |
| `ACCESS_NETWORK_STATE` | 网络状态判断 | 用途待发布负责人确认 |
| `ACCESS_WIFI_STATE` | 全部源码中未发现使用 | 建议移除 |
| `VIBRATE` | 全部源码中未发现使用 | 建议移除 |

HarmonyOS（`uniapp/harmony-configs/entry/src/main/module.json5`）：

| 权限 | 用途 | 状态 |
| --- | --- | --- |
| `INTERNET` | 网络请求 | 有用途 |
| `LOCATION` / `APPROXIMATELY_LOCATION` | 附近球房（reason：`location_reason`） | 有用途 |
| `CAMERA` | 扫码 | **未声明**，鸿蒙扫码是否可用需真机确认 |

微信小程序：`scope.userLocation`，用途描述为"用于获取当前位置并展示附近球房"。

## 5. 签名轮换

见 `harmony-signing-rotation-acceptance.md`。旧材料废弃日期 2026-08-27；新发布包签名验证通过。

## 6. 三端真机证据

待 `device-matrix-acceptance.md` 填写。

## 7. 部署与备份记录

| 项目 | 记录 |
| --- | --- |
| 后端（生产） | 发布 `20261020113937-0568b7df5a24`，提交 `0568b7d`，待执行迁移 0 |
| 后端（测试） | 发布 `20261020113824-0568b7df5a24`（提交 `0568b7d`，包含华为变量改名）；之后仅修改了 `.env` 的华为启用项并重启，未重新发布 |
| 管理后台（生产） | 发布 `20261020102418-51c6b433ab34` |
| 生产数据库备份 | `deploy-tools/db-backups/pre-deploy-*` 下的 `.sql.gz`，最近一次 `pre-deploy-0568b7df-20261020T113940` |

## 8. 签署

| 角色 | 姓名 | 日期 | 结论 |
| --- | --- | --- | --- |
| 发布负责人 |  |  |  |
