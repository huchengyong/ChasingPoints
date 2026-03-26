import { existsSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { comingSoonContent } from '../data/coming-soon'

describe('comingSoonContent', () => {
  it('uses the approved title and short waiting copy', () => {
    expect(comingSoonContent.title).toBe('敬请期待')
    expect(comingSoonContent.description).toContain('当前平台下载暂未开放')
  })

  it('offers clear actions back to home and download pages', () => {
    expect(comingSoonContent.primaryAction.to).toBe('/')
    expect(comingSoonContent.secondaryAction.to).toBe('/download')
  })

  it('has a dedicated route file for the coming-soon page', () => {
    expect(existsSync(resolve(import.meta.dirname, '../pages/coming-soon.vue'))).toBe(true)
  })
})
