import { existsSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { privacyPolicy, userAgreement } from '../data/legal'

describe('legal content', () => {
  it('keeps updated dates for both legal documents', () => {
    expect(privacyPolicy.updatedAt).toBeTruthy()
    expect(userAgreement.updatedAt).toBeTruthy()
  })

  it('contains at least one section in each document', () => {
    expect(privacyPolicy.sections.length).toBeGreaterThan(0)
    expect(userAgreement.sections.length).toBeGreaterThan(0)
  })

  it('has route files for download, privacy, agreement, and contact pages', () => {
    const pageDir = resolve(import.meta.dirname, '../pages')

    expect(existsSync(resolve(pageDir, 'download.vue'))).toBe(true)
    expect(existsSync(resolve(pageDir, 'privacy.vue'))).toBe(true)
    expect(existsSync(resolve(pageDir, 'agreement.vue'))).toBe(true)
    expect(existsSync(resolve(pageDir, 'contact.vue'))).toBe(true)
  })
})
