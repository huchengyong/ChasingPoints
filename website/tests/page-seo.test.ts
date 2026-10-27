import { describe, expect, it } from 'vitest'
import { buildPageSeo } from '../composables/usePageSeo'

describe('buildPageSeo', () => {
  it('returns download page metadata with canonical path', () => {
    const result = buildPageSeo({
      title: '下载追分',
      description: '微信小程序与鸿蒙下载入口',
      path: '/download'
    })

    expect(result.title).toContain('下载追分')
    expect(result.ogUrl).toContain('/download')
  })
})
