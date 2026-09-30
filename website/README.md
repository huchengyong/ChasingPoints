# 追分官网 `website/`

这是独立的 Nuxt 3 官网子项目，用于承接品牌官网、下载引导、隐私政策、用户协议和联系页。开发约束见 [AGENTS.md](AGENTS.md)。

## 常用命令

```bash
cd website

# 安装依赖
npm install

# 本地开发
npm run dev

# 生产构建
npm run build

# 静态生成
npm run generate

# 测试
npm run test
```

## 内容维护入口

- 下载链接：`website/data/download.ts`
- 首页文案与 FAQ：`website/data/home.ts`
- 隐私政策与用户协议：`website/data/legal.ts`
- 联系方式：`website/data/contact.ts`
- 基础站点配置：`website/data/site.ts`
- 页面 SEO helper：`website/composables/usePageSeo.ts`

## 素材与静态资源

- Logo 与二维码：`website/public/`
- 全局样式：`website/assets/styles/main.css`

## 环境变量

- `NUXT_PUBLIC_SITE_URL`
  - 官网域名，用于 canonical 与 OG URL，默认 `https://www.zhuifen.cn`。
  - `public/sitemap.xml` 与 `robots.txt` 为静态文件，需另行核对，不会随该变量自动改写。
- `NUXT_PUBLIC_CONTACT_EMAIL` / `NUXT_PUBLIC_CONTACT_PHONE`
  - 官网联系方式，默认值见 [data/contact.ts](data/contact.ts)。
- `NUXT_PUBLIC_API_BASE_URL`
  - 联系页投诉举报接口基地址，配置来源同上。
- `NUXT_PUBLIC_ICP_RECORD_NUMBER`
  - 官网底部 ICP 备案号，默认值见 [data/site.ts](data/site.ts)。

数据配置通过 [data/runtime.ts](data/runtime.ts) 等入口读取环境；构建/静态生成前设置所需值，修改后核对实际页面产物。

## 当前约定

- 页面统一通过 `pages/*.vue` 暴露公开路由，不和 `admin/` 共用路由体系。
- 长文协议内容放在 `data/legal.ts`，不要把正文直接写死在页面组件里。
- iOS/Android 下载按钮及二维码目标当前统一指向 `https://www.zhuifen.cn/coming-soon`，见 [data/download.ts](data/download.ts)；代码未消费 `NUXT_PUBLIC_IOS_DOWNLOAD_URL` / `NUXT_PUBLIC_ANDROID_DOWNLOAD_URL`，设置它们不会切换入口。
- 正式下载开放时同步修改按钮、二维码资源与目标、文案及 [下载契约测试](tests/download-content.test.ts)。联系方式、备案与 sitemap 已有具体内容，发布前核实，不一概视为占位值。
