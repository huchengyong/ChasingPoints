import { describe, expect, it } from 'vitest'
import { homePageContent } from '../data/home'

describe('homePageContent', () => {
  it('keeps the primary CTA pointed at the download page', () => {
    expect(homePageContent.hero.primaryAction.to).toBe('/download')
  })

  it('includes exactly four product highlight items', () => {
    expect(homePageContent.highlights).toHaveLength(4)
  })

  it('provides three preview cards for the homepage hero montage', () => {
    expect(homePageContent.hero.previewCards).toHaveLength(3)
  })
})
