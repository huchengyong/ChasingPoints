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

  it('defaults the public security record displayed in the site footer', () => {
    expect(siteConfig.policeRecordNumber).toBe('沪公网安备31011502406701号')
    expect(siteConfig.policeRecordUrl).toBe(
      'https://beian.mps.gov.cn/#/query/webSearch?code=31011502406701'
    )
    expect(siteConfig.policeRecordIconUrl).toBe(
      'https://beian.mps.gov.cn/web/assets/logo01.6189a29f.png'
    )
  })
})
