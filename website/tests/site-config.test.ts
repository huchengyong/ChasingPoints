import { describe, expect, it } from 'vitest'
import { siteConfig } from '../data/site'

describe('siteConfig', () => {
  it('contains the core marketing routes', () => {
    expect(siteConfig.navLinks.map((item) => item.to)).toEqual([
      '/',
      '/download',
      '/privacy',
      '/agreement',
      '/contact'
    ])
  })

  it('defaults the public site host to www.zhuifen.cn', () => {
    expect(siteConfig.siteUrl).toBe('https://www.zhuifen.cn')
  })

  it('defaults the ICP record displayed in the site footer', () => {
    expect(siteConfig.icpRecordNumber).toBe('沪ICP备2021037913号-11')
  })
})
