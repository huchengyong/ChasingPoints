import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const pageSource = readFileSync(
  new URL('../pages/social/index.vue', import.meta.url),
  'utf8'
)

const styleSource = readFileSync(
  new URL('../pages/social/index.scss', import.meta.url),
  'utf8'
)

test('social page reapplies theme on show for the native navigation bar', () => {
  assert.match(pageSource, /usePageTheme/)
  assert.match(pageSource, /const \{ isDarkMode \} = usePageTheme\(\)/)
  assert.doesNotMatch(pageSource, /useThemeStore|THEME_CHANGE_EVENT|applyNavigationBarTheme/)
})

test('social page dark hero keeps light text on a dark card', () => {
  assert.match(
    styleSource,
    /&\.dark-mode\s*\{[\s\S]*\.hero-card\s*\{[\s\S]*background:\s*linear-gradient\(135deg,\s*#241d10 0%,\s*#1a1408 100%\);[\s\S]*border-color:\s*#3a2e16;/
  )
  assert.match(
    styleSource,
    /&\.dark-mode\s*\{[\s\S]*\.hero-desc,[\s\S]*color:\s*#d7c89b;/
  )
  assert.match(
    styleSource,
    /&\.dark-mode\s*\{[\s\S]*\.hero-title,[\s\S]*color:\s*#fff7e1;/
  )
})
