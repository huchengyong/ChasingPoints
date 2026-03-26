import { describe, expect, it } from 'vitest'
import { resolvePublicValue } from '../data/runtime'

describe('resolvePublicValue', () => {
  it('falls back when the environment variable is missing', () => {
    expect(resolvePublicValue('NUXT_PUBLIC_TEST_ONLY', 'fallback')).toBe('fallback')
  })

  it('returns the environment override when present', () => {
    process.env.NUXT_PUBLIC_TEST_ONLY = 'live-value'

    expect(resolvePublicValue('NUXT_PUBLIC_TEST_ONLY', 'fallback')).toBe('live-value')

    delete process.env.NUXT_PUBLIC_TEST_ONLY
  })
})
