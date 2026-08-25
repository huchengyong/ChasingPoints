# HBuilderX 本地 Harmony 签名

HBuilderX 在编译 HarmonyOS 前会把本目录复制到临时 Harmony 工程。因此，`build-profile.json5` 可以在不修改受 Git 管理的 `manifest.json` 的前提下提供本机签名材料。

首次配置：

1. 将 `app/scripts/harmony-signing.local.example.json` 复制为 `app/.harmony-signing.local.json`（这里的项目根目录指 `app/`）。
2. 仅在该本地文件中填写证书、profile、p12 路径、别名和密码。它已被 Git 忽略，文件权限由生成脚本收紧为仅当前用户可读写。
3. 在 `app/` 目录执行：`node scripts/prepare-harmony-signing.mjs`。
4. 回到 HBuilderX 执行运行或打包。调试运行使用 `default`，本地发布包使用 `release`。

不要把签名信息填回 `manifest.json`，也不要提交 `.harmony-signing.local.json` 或本目录生成的 `build-profile.json5`。若升级 HBuilderX 后其 Harmony 模板发生变化，请审阅并同步 `scripts/harmony-build-profile.template.json`，再重新生成本地文件。
