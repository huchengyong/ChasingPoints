import { STATIC_ASSET_BASE_URL } from './runtime-config.js'

// 这些资源存放在 cdn-assets/ 并上传到 CDN，不放进 static/，避免撑大小程序主包。
// 后端返回的 icon 仍是 /static/images/... 形式的路径，统一在这里转成 CDN 地址。
const CDN_ASSET_PATH_PATTERN = /^\/static\/images\/(?:(?:achievements|ranks|avatar-frames|gear)\/[^/]+|default-event-cover\.png)$/

export const isCdnAssetPath = (path) => typeof path === 'string' && CDN_ASSET_PATH_PATTERN.test(path)

export const resolveCdnAssetUrl = (path) => (isCdnAssetPath(path) ? `${STATIC_ASSET_BASE_URL}${path}` : path)
