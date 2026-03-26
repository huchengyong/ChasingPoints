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
})
