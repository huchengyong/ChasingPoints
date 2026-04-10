import { describe, expect, it } from 'vitest'
import { contactInfo } from '../data/contact'

describe('contactInfo', () => {
  it('defaults to the official service email and landline', () => {
    expect(contactInfo.email).toBe('service@dianzaozao.com')
    expect(contactInfo.phone).toBe('021-50583611')
  })

  it('covers complaint and report submissions in contact copy', () => {
    expect(contactInfo.title).toContain('投诉举报')
    expect(contactInfo.description).toContain('投诉举报')
    expect(contactInfo.feedbackApiBaseUrl).toBe('https://api-bm.dianzaozao.com')
  })
})
