import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('seo assets', () => {
  it('includes the download route in sitemap.xml', () => {
    const sitemap = readFileSync(resolve(import.meta.dirname, '../public/sitemap.xml'), 'utf8')

    expect(sitemap).toContain('/download')
  })
})
