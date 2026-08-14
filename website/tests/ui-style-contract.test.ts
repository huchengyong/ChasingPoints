import { readFileSync, readdirSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const websiteRoot = resolve(import.meta.dirname, '..')
const stylesSource = readFileSync(join(websiteRoot, 'assets/styles/main.css'), 'utf8')

// Core brand and foundational neutral literals that must be centralized as tokens
// once a shared component has been migrated to the public UI standard.
const CORE_LITERALS = ['#d9a617', '#bb8400', '#9f6c00', '#1f1a10', '#fff7e1']

const SHARED_COMPONENTS = [
  'SiteHeader.vue',
  'SiteFooter.vue',
  'HeroSection.vue',
  'PageHero.vue',
  'FeatureGrid.vue',
  'ScenarioCards.vue',
  'FaqList.vue',
  'DownloadPanel.vue',
  'LegalArticle.vue'
]

const REQUIRED_TOKENS = [
  '--ui-brand-primary',
  '--ui-brand-strong',
  '--ui-brand-gradient-start',
  '--ui-brand-gradient-end',
  '--ui-surface-page',
  '--ui-surface-card',
  '--ui-surface-subtle',
  '--ui-surface-accent',
  '--ui-text-primary',
  '--ui-text-secondary',
  '--ui-text-muted',
  '--ui-text-on-accent',
  '--ui-border-default',
  '--ui-success',
  '--ui-warning',
  '--ui-danger',
  '--ui-info',
  '--ui-radius-lg',
  '--ui-radius-pill',
  '--ui-shadow-primary'
]

describe('website UI token contract', () => {
  it('defines the canonical semantic token set in main.css', () => {
    for (const token of REQUIRED_TOKENS) {
      expect(stylesSource).toContain(`${token}:`)
    }
  })

  it('keeps the body background and text on the semantic surface tokens', () => {
    expect(stylesSource).toMatch(/background:\s*var\(--ui-surface-page\)/)
    expect(stylesSource).toMatch(/color:\s*var\(--ui-text-primary\)/)
  })

  it('migrated shared components do not hardcode core brand or neutral literals', () => {
    const offenders: string[] = []
    for (const component of SHARED_COMPONENTS) {
      const source = readFileSync(join(websiteRoot, 'components', component), 'utf8')
      for (const literal of CORE_LITERALS) {
        if (source.toLowerCase().includes(literal.toLowerCase())) {
          offenders.push(`${component} -> ${literal}`)
        }
      }
    }
    expect(offenders).toEqual([])
  })

  it('exposes a reusable primary CTA primitive', () => {
    expect(stylesSource).toMatch(/\.ui-cta\s*\{/)
    expect(stylesSource).toMatch(/var\(--ui-brand-gradient-start\)/)
  })
})
