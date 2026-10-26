import test from 'node:test'
import assert from 'node:assert/strict'
import { readdirSync, statSync } from 'node:fs'
import { join, relative, sep } from 'node:path'

import { isCdnAssetPath, resolveCdnAssetUrl } from '../utils/cdn-asset.js'
import { STATIC_ASSET_BASE_URL } from '../utils/runtime-config.js'

const listFiles = (dir) => readdirSync(dir).flatMap((name) => {
  const full = join(dir, name)
  return statSync(full).isDirectory() ? listFiles(full) : [full]
})

const toAssetPath = (root, file) => `/${relative(root, file).split(sep).join('/')}`

test('resolveCdnAssetUrl maps rank, achievement, avatar frame, gear and default cover paths to the CDN', () => {
  assert.equal(STATIC_ASSET_BASE_URL, 'https://cdn.zhuifen.cn')
  assert.equal(resolveCdnAssetUrl('/static/images/ranks/rank_gold.png'), 'https://cdn.zhuifen.cn/static/images/ranks/rank_gold.png')
  assert.equal(resolveCdnAssetUrl('/static/images/achievements/wins_1.png'), 'https://cdn.zhuifen.cn/static/images/achievements/wins_1.png')
  assert.equal(resolveCdnAssetUrl('/static/images/avatar-frames/frame_king.png'), 'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_king.png')
  assert.equal(resolveCdnAssetUrl('/static/images/default-event-cover.png'), 'https://cdn.zhuifen.cn/static/images/default-event-cover.png')
  assert.equal(resolveCdnAssetUrl('/static/images/gear/gear-cue-pool-default.png'), 'https://cdn.zhuifen.cn/static/images/gear/gear-cue-pool-default.png')
})

test('resolveCdnAssetUrl leaves other values untouched and is idempotent', () => {
  assert.equal(resolveCdnAssetUrl('/static/logo.png'), '/static/logo.png')
  assert.equal(resolveCdnAssetUrl('/static/images/default-avatars/ball-01.png'), '/static/images/default-avatars/ball-01.png')
  assert.equal(resolveCdnAssetUrl('https://cdn.zhuifen.cn/avatar/1.png'), 'https://cdn.zhuifen.cn/avatar/1.png')
  assert.equal(resolveCdnAssetUrl('/static/images/ranks/../logo.png'), '/static/images/ranks/../logo.png')
  assert.equal(resolveCdnAssetUrl('/static/images/gear/../logo.png'), '/static/images/gear/../logo.png')
  assert.equal(resolveCdnAssetUrl('/static/images/gear/sub/a.png'), '/static/images/gear/sub/a.png')
  assert.equal(resolveCdnAssetUrl(''), '')
  assert.equal(resolveCdnAssetUrl(undefined), undefined)
  assert.equal(resolveCdnAssetUrl(null), null)

  const once = resolveCdnAssetUrl('/static/images/ranks/rank_gold.png')
  assert.equal(resolveCdnAssetUrl(once), once)
})

test('every file in cdn-assets is resolved to the CDN and none of them stays in the mini-program static directory', () => {
  const cdnRoot = new URL('../cdn-assets', import.meta.url).pathname
  const staticRoot = new URL('../static', import.meta.url).pathname
  const cdnPaths = listFiles(cdnRoot).map((file) => toAssetPath(cdnRoot, file))

  assert.ok(cdnPaths.length > 0)
  for (const assetPath of cdnPaths) {
    assert.equal(isCdnAssetPath(assetPath), true, `${assetPath} 不会被解析到 CDN`)
  }

  const bundledPaths = listFiles(staticRoot).map((file) => `/static${toAssetPath(staticRoot, file)}`)
  for (const assetPath of bundledPaths) {
    assert.equal(isCdnAssetPath(assetPath), false, `${assetPath} 应只存在于 cdn-assets，不能留在 static`)
  }
})
