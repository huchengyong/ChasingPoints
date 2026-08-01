## Why

WST 图片 CDN 会针对微信小程序固定的 `servicewechat.com` Referer 返回 403，导致赛讯中的球员头像和赛事封面在小程序内无法展示。现有客户端 `downloadFile` 缓存仍经过同一受限网络链路，无法绕过该限制，因此需要在后端同步阶段将允许使用的 WST 图片镜像到自有七牛 CDN。

## What Changes

- 在 WST 同步过程中，仅镜像球员头像和赛事封面到现有七牛存储，并使用确定性对象键保证重复同步幂等。
- 镜像成功后，将自有 CDN URL 写入现有 `players.avatar`、`tournaments.cover_image` 及对应赛讯封面数据，不新增数据库字段或修改 API 响应结构。
- 镜像失败时保留已有自有 CDN URL；没有可用旧图时使用自有默认图，不再向小程序下发 `images.gc.wstservices.co.uk` URL。
- 通过一次完整 WST 同步回填现有球员和赛事图片，不新增独立的数据迁移链路。
- 替换前后端当前指向 WST 的默认赛事封面，并移除赛讯详情对客户端 WST 图片下载缓存的依赖。
- 将自有七牛 CDN 域名纳入微信小程序下载合法域名和发布验收清单。

## Capabilities

### New Capabilities
- `wst-image-mirroring`: 定义 WST 球员头像和赛事封面的受限下载、七牛镜像、幂等复用、失败回退及对外 URL 约束。

### Modified Capabilities

无。

## Impact

- 后端主要影响 `backend/internal/pkg/qiniu/`、`backend/internal/logic/wstsync/`、WST 默认封面常量和相关测试；继续复用现有七牛配置与数据库 URL 字段。
- 移动端主要影响 `app/utils/saixun.js`、`app/subPages/tournament/detail.vue`、`app/utils/image-cache.js` 及对应测试。
- API 契约和数据库结构保持不变，不需要修改 `backend/chasing_points.api` 或新增 migration。
- 部署侧需要确认七牛 Bucket/CDN 可用，并在微信公众平台配置对应下载合法域名。
- 图片镜像和分发必须以已确认的 WST 图片使用授权为前提，不建立任意第三方 URL 代理能力。
