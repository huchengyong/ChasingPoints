import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const footerSource = readFileSync(resolve(import.meta.dirname, '../components/SiteFooter.vue'), 'utf8')

describe('SiteFooter compliance records', () => {
  it('renders the public security record from site configuration', () => {
    expect(footerSource).toContain('siteConfig.policeRecordUrl')
    expect(footerSource).toContain('siteConfig.policeRecordNumber')
    expect(footerSource).toContain('siteConfig.policeRecordIconUrl')
  })

  it('keeps compliance links secure and responsive', () => {
    expect(footerSource).toContain('rel="noopener noreferrer"')
    expect(footerSource).toContain('site-footer__records')
    expect(footerSource).toContain('flex-wrap: wrap')
  })
})
