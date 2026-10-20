# P0 发布证据汇总（任务 8.8）

> 对应 OpenSpec 变更 `fix-client-p0-release-blockers` 的任务 8.8。只记录标识、配置状态与验证结果，不记录任何密码、token 或凭据值。任一证据缺失，P0 发布仍然阻断。

## 1. 状态

部分完成。第 2–5 节已有证据，第 6 节（三端真机证据）由 `device-matrix-acceptance.md` 承接，尚未填写。

## 2. 正式允许 Origin

来源：`backend/etc/chasing_points-api.yaml` 的 `Security` 段（线上配置与仓库一致）。

- HTTP：`https://www.zhuifen.cn`、`https://api.zhuifen.cn`、`https://ws.zhuifen.cn`、`https://admin.zhuifen.cn`
- WebSocket：`https://www.zhuifen.cn`、`https://api.zhuifen.cn`、`https://ws.zhuifen.cn`
- `AllowNativeMissingOrigin: true`：原生客户端在 TLS 下可不带 Origin，见 `fail-closed-verification.md`。

线上只读验证结果见 `fail-closed-verification.md` 第 3 节。

注意：仓库 yaml 的 `HTTPAllowedOrigins` 仍含测试管理后台来源 `https://admin-zhuifen.kekemate.cn`（该测试站在使用）；`WebSocketAllowedOrigins` 已在提交 `a770775` 移除测试 API 来源。生产服务器上的 yaml 本来就不含这两项。

## 3. 华为服务端配置

| 项目 | 测试环境 | 生产环境 |
| --- | --- | --- |
| `ZHUIFEN_HM_OAUTH_ENABLED` | 已设置为 true（2026-10-20） | 已设置为 true（2026-10-20） |
| `ZHUIFEN_HM_OAUTH_CLIENT_ID` | 已设置（来自本机配置，与 yaml 中的 `ClientID` 一致） | 同左 |
| `ZHUIFEN_HM_OAUTH_CLIENT_SECRET` | 已设置（值不记录） | 已设置（值不记录） |
| `VerifyURL` | yaml 中为 `https://oauth-login.cloud.huawei.com/oauth2/v3/tokeninfo` | 同左 |
| 失败关闭检查 | 假凭据返回 400 `Huawei rejected OAuth credential` | 同左 |

`CLIENT_SECRET`：发布负责人确认为华为侧重置后的值；本机、测试、生产三处的指纹一致（只比对指纹，不记录值）。

## 4. Manifest 权限用途

App（`uniapp/manifest.json`）：

| 权限 | 用途 | 状态 |
| --- | --- | --- |
| `CAMERA` | 扫码（匹配码、裁判码） | 保留，需真机验证 |
| `ACCESS_NETWORK_STATE` | `App.vue` 的网络状态监听与网络类型判断，用于实时连接重连和弱网提示 | 保留（发布负责人 2026-10-20 确认） |
| `VIBRATE`、`ACCESS_WIFI_STATE`、`WAKE_LOCK`、`FLASHLIGHT` | 全部源码中未发现使用 | 已移除（尚未提交） |

HarmonyOS（`uniapp/harmony-configs/entry/src/main/module.json5`）：

| 权限 | 用途 | 状态 |
| --- | --- | --- |
| `INTERNET` | 网络请求 | 有用途 |
| `LOCATION` / `APPROXIMATELY_LOCATION` | 附近球房（reason：`location_reason`） | 有用途 |
| `CAMERA` | 扫码（reason：`camera_reason`） | 已声明（尚未提交），鸿蒙扫码需真机确认 |

微信小程序：`scope.userLocation`，用途描述为"用于获取当前位置并展示附近球房"。

## 5. 签名轮换

见 `harmony-signing-rotation-acceptance.md`。旧材料废弃日期 2026-08-27；新发布包签名验证通过。

## 6. 三端真机证据

待 `device-matrix-acceptance.md` 填写。

## 7. 部署与备份记录

| 项目 | 记录 |
| --- | --- |
| 后端（生产） | 发布 `20261020142107-877dc00febf8`，提交 `877dc00`，待执行迁移 0；另在 `api.zhuifen.cn` 的 nginx 反向代理增加 `X-Forwarded-Proto`（见 `fail-closed-verification.md`） |
| 后端（测试） | 发布 `20261020155625-a770775f80e5`，提交 `a770775` |
| 管理后台（生产） | 发布 `20261020102418-51c6b433ab34` |
| 生产数据库备份 | `deploy-tools/db-backups/pre-deploy-*` 下的 `.sql.gz`，最近一次 `pre-deploy-877dc00f-20261020T142110` |

## 8. 签署

| 角色 | 姓名 | 日期 | 结论 |
| --- | --- | --- | --- |
| 发布负责人 |  |  |  |
