# HarmonyOS 签名材料轮换验收记录（P0 任务 8.7）

> 对应 OpenSpec 变更 `fix-client-p0-release-blockers` 的任务 8.7。记录只含公开的证书指纹、profile 标识与验证结果，不含任何密码或私钥。

## 1. 背景

- 初始提交 `9a48bba`（2026-03-23）的 `app/manifest.json` 中含有非空的签名密码字段和机器绝对路径，已进入 git 历史。仓库为公开仓库，历史不可撤回。
- 旧证书与旧 profile 已由发布负责人在 AppGallery Connect / DevEco 后台删除。

## 2. 新材料

| 项目 | 内容 |
| --- | --- |
| 发布证书主题 | `CN="上海点早早互联网络科技有限公司(1594307369066624769)\,Release"` |
| 发布证书有效期 | 2026-10-20 11:11:29 ~ 2029-10-20 11:11:29 |
| 发布证书 SHA256 | `EA:C5:ED:AB:31:69:69:A9:EC:48:57:17:E9:CB:F0:11:52:B2:2A:DA:A2:AD:E9:A6:58:A1:E0:BF:57:CA:C8:7F` |
| 发布 profile | 类型 `release`，分发方式 `app_gallery`，包名 `hm.dianzaozao.zhuifen`，uuid 前缀 `ebc929c2` |
| 开发证书 | 已重新签发，本次未对开发包做签名验证 |
| 存放位置 | 仓库外的本地证书目录，通过环境变量 `ZHUIFEN_HM_*` 引用 |
| 配置文件 | `uniapp/.harmony-signing.local.json`（被 git 忽略，权限 600），密码只存在于此文件和密码管理器中 |

## 3. 验证步骤与结果

1. 使用新的本地签名配置（发布组）通过 HBuilderX 构建 HarmonyOS release 包：`app-harmony-release-signed.app`（约 11.5 MB）。
2. 使用 DevEco 自带的 `hap-sign-tool.jar verify-app` 验证该包：
   - 结果：`Verify success`，`Digest verify result: true`（SHA-256）。
   - 证书链共 3 张：根 CA（`Huawei CBG Root CA G2`）、开发者关系 CA（`Huawei CBG Developer Relations CA G2`）、应用发布证书。
3. 包内应用证书的 SHA256 与本地发布证书一致（见第 2 节）。
4. 包内导出的 profile 类型、包名、分发方式与 uuid 与本地发布 profile 一致。
5. 构建期 `build-profile.json5` 由 `scripts/prepare-harmony-signing.mjs` 从本地配置生成，生成后已被 git 忽略。旧的 `build-profile.json5` 已删除。

## 4. 旧材料废弃

- 废弃状态：已在后台删除（发布负责人确认）。
- 废弃日期：2026-08-27（发布负责人提供）。

## 5. 遗留事项

- 旧密码仍存在于 git 历史中（`9a48bba`）。密码已轮换失效，但历史未清理；如需清理需改写历史并强推，需单独批准。
## 6. 结论

签名材料已完成轮换，旧材料废弃日期为 2026-08-27，新发布包的签名验证通过。8.7 的证据已齐备。
