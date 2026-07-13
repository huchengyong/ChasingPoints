import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('seo assets', () => {
  it('includes the download route in sitemap.xml', () => {
    const sitemap = readFileSync(resolve(import.meta.dirname, '../public/sitemap.xml'), 'utf8')

    expect(sitemap).toContain('/download')
  })

  it('uses the production www.zhuifen.cn host in static seo assets', () => {
    const sitemap = readFileSync(resolve(import.meta.dirname, '../public/sitemap.xml'), 'utf8')
    const robots = readFileSync(resolve(import.meta.dirname, '../public/robots.txt'), 'utf8')

    expect(sitemap).toContain('https://www.zhuifen.cn/')
    expect(sitemap).not.toContain('https://example.com')
    expect(robots).toContain('Sitemap: https://www.zhuifen.cn/sitemap.xml')
    expect(robots).not.toContain('https://example.com')
  })
})
