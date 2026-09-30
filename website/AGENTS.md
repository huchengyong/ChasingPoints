# WEBSITE FRONTEND GUIDE

适用于 `website/`（Nuxt 3 + Vue3 + TypeScript），同时遵循[仓库指南](../AGENTS.md)。依赖版本与 scripts 以 [package.json](package.json) 为准。

## 关键入口
| 任务 | 位置 |
| --- | --- |
| 公开路由 | [pages/](pages/)，包括下载引导 `/coming-soon` |
| 页面文案与配置 | [data/](data/)，不把协议正文、联系方式散写到组件 |
| 页面 SEO | [composables/usePageSeo.ts](composables/usePageSeo.ts) |
| 构建与预渲染 | [nuxt.config.ts](nuxt.config.ts) |
| Sitemap、robots、二维码与分享图 | [public/](public/) |
| 内容与样式回归 | [tests/](tests/) |

## 配置与发布边界
- 站点域名、备案配置看 [data/site.ts](data/site.ts)；联系方式与投诉接口地址看 [data/contact.ts](data/contact.ts)。这些文件已有默认值，发布前逐项核对，不笼统当作占位内容。
- 环境变量由 [data/runtime.ts](data/runtime.ts) 等现有入口读取，配置说明见 [README.md](README.md)；修改后核对实际构建/生成页面，不假设只改服务器环境即可更新静态产物。
- [data/download.ts](data/download.ts) 当前将 iOS/Android 按钮和二维码目标统一指向 `https://www.zhuifen.cn/coming-soon`；未消费 `NUXT_PUBLIC_IOS_DOWNLOAD_URL` / `NUXT_PUBLIC_ANDROID_DOWNLOAD_URL`，仅设置这两个变量不会切换下载入口。
- 正式下载开放时同时核对按钮、二维码资源及目标、文案与 [下载契约测试](tests/download-content.test.ts)，不在无关任务中把预告页替换为下载页。
- `public/sitemap.xml`、`robots.txt` 是静态文件，域名或路由改变需一并核对，不会仅因 `NUXT_PUBLIC_SITE_URL` 改动自动同步。

## UI 与工程约束
- 品牌、布局与可访问性遵循 [DESIGN.md](../DESIGN.md)，颜色等语义 Token 消费 [assets/styles/main.css](assets/styles/main.css)。
- 页面统一使用 `usePageSeo`，不多处复制 meta；路由通过 `pages/*.vue` 暴露，不接入后台布局或路由。
- 不提交 `.nuxt/`、`.output/`、`node_modules/` 等构建与依赖产物。

## 验证与开发命令
以下命令在 `website/` 执行，按需选择；依赖通过 `npm install` 安装。

```bash
npm run test
npm run build
# 静态发布时另外核对生成结果
npm run generate
# 本地开发
npm run dev
```
