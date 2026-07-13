# 追分官网 `website/`

这是一个独立于 `app/`、`admin/`、`backend/` 的 Nuxt 3 官网子项目，首期用于承接品牌官网、下载引导、隐私政策、用户协议和联系页。

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
  - 官网正式域名，用于 canonical、OG URL 和 sitemap 对齐。
  - 未配置时默认回退到 `https://www.zhuifen.cn`。
- `NUXT_PUBLIC_IOS_DOWNLOAD_URL`
  - iOS 正式下载地址。
- `NUXT_PUBLIC_ANDROID_DOWNLOAD_URL`
  - Android 正式下载地址。
- `NUXT_PUBLIC_CONTACT_EMAIL`
  - 官网联系邮箱。
- `NUXT_PUBLIC_CONTACT_PHONE`
  - 官网联系座机。
- `NUXT_PUBLIC_ICP_RECORD_NUMBER`
  - 官网底部展示的 ICP 备案号。

## 当前约定

- 页面统一通过 `pages/*.vue` 暴露公开路由，不和 `admin/` 共用路由体系。
- 长文协议内容放在 `data/legal.ts`，不要把正文直接写死在页面组件里。
- 正式 iOS/Android 下载地址上线前需要替换 `website/data/download.ts` 中的占位链接。
