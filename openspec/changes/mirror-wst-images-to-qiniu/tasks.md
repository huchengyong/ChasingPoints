## 1. 七牛服务端图片镜像能力

- [x] 1.1 在 `backend/internal/pkg/qiniu/` 增加按图片类别和规范化来源 URL 生成 SHA-256 确定性对象键的 helper，并补公开 URL 与当前七牛域名识别测试
- [x] 1.2 实现受限远程图片下载：仅允许 HTTPS 与 `images.gc.wstservices.co.uk`，设置超时、状态码校验、PNG/JPEG/WebP 类型校验和固定大小上限
- [x] 1.3 扩展现有 `UploadService`，先检查确定性对象是否存在，存在时直接复用，不存在时才上传并返回公开 URL
- [x] 1.4 为非法协议、非法主机、重定向越界、错误状态、非法类型、超限响应、首次上传和已有对象复用补齐七牛包单元测试
- [x] 1.5 运行七牛包测试并确认现有用户头像上传凭证签发行为无回归

## 2. WST 同步接入

- [x] 2.1 为 WST `Service` 增加最小图片镜像依赖并提供测试 fake，生产环境复用 `ServiceContext.QiniuUploadService`
- [x] 2.2 在非 dry-run 同步进入数据库事务前镜像球员头像，并将成功结果写入 `PlayerUpsertRecord`
- [x] 2.3 在非 dry-run 同步进入数据库事务前镜像真实赛事封面，将 WST 通用默认封面视为缺失值而不是可持久化图片
- [x] 2.4 调整球员和赛事 upsert 回退：本次镜像失败时只保留当前七牛公开域名 URL，没有可保留图片时写入空值，不回退到 WST URL
- [x] 2.5 确保单张图片下载或上传失败只记录不含凭证的资源错误并继续同步其他业务数据
- [x] 2.6 增加同步测试，覆盖镜像成功、对象复用、来源变化、镜像失败保留旧七牛 URL、无旧图回空以及未配置七牛的降级行为
- [x] 2.7 增加 dry-run 和事务边界测试，确认 dry-run 不调用镜像服务且外部图片操作发生在数据库事务之前

## 3. 公开赛讯与移动端收口

- [x] 3.1 移除 WST 默认封面作为后端公开默认值，并在赛讯列表、详情和球员头像映射中将残留的 WST 图片 URL 过滤为空值
- [x] 3.2 更新 event-news 和 WST 同步相关测试，确认 WST 官方赛讯图片只返回当前七牛公开域名 URL 或空值，且绝不返回 `images.gc.wstservices.co.uk`
- [x] 3.3 在 `app/static/` 增加应用自有默认赛事封面，并将 `app/utils/saixun.js` 的默认封面改为该包内资源
- [x] 3.4 从赛讯详情移除 `warmCachedPlayerAvatars` 和 `cacheSaiXunMatchAvatars` 调用，直接渲染 API 返回的七牛 URL
- [x] 3.5 删除仅为 WST 绕过服务的 `app/utils/image-cache.js` 与 `app/tests/image-cache.test.mjs`，并清理由此产生的未使用引用
- [x] 3.6 更新移动端赛讯测试，覆盖七牛图片直出、缺失封面使用本地默认图以及代码中不再存在 WST 默认图片常量

## 4. 自动化验证

- [x] 4.1 在 `backend/` 运行 WST 同步、event-news 和七牛相关定向测试并修复本变更引入的失败
- [x] 4.2 在 `backend/` 运行 `go test ./...`，确认其他同步、用户头像上传和赛讯链路无回归
- [x] 4.3 在 `app/` 运行 `node --test tests/*.test.mjs`，确认移动端纯逻辑测试全部通过
- [x] 4.4 执行仓库搜索，确认生产代码和默认值中不再向移动端暴露 `images.gc.wstservices.co.uk`，仅允许受限镜像源校验代码保留该主机
- [x] 4.5 运行 `openspec validate mirror-wst-images-to-qiniu --type change --strict` 并修复所有校验问题

## 5. 回填与发布验收

- [x] 5.1 在启用镜像前确认 WST 图片使用授权、七牛 Bucket/公开域名配置和自有默认赛事封面版权
- [x] 5.2 先发布后端，并使用现有 `backend/cmd/wst_sync` 对需要保留的 WST 赛季执行完整非 dry-run 同步
- [x] 5.3 回填后核查官方球员、赛事和赛讯 API，确认图片 URL 为当前七牛公开域名或空值，不存在 WST 图片域名残留
- [x] 5.4 在微信公众平台将七牛公开域名配置为下载合法域名，并通过体验版真机确认球员头像、赛事封面和本地默认图均正常展示
- [x] 5.5 发布移动端版本并记录回滚顺序：回滚后端前先关闭 WST 自动同步，保留已镜像对象和数据库七牛 URL
