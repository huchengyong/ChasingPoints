import test from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

const appRoot = fileURLToPath(new URL('..', import.meta.url))
const repoRoot = fileURLToPath(new URL('../../', import.meta.url))

// Legacy cool blue-black values that must not serve as page/card/input/modal surfaces.
const PROHIBITED_SURFACES = ['#101922', '#0f172a', '#0f1720', '#1e293b', '#0b1120', '#0c141d']

// Files that legitimately keep a cool dark canvas for posters/media overlays rather than page surfaces.
const ALLOWED_SURFACE_FILES = new Set([
  'subPages/match/shareResult.vue'
])

const listFiles = (dir) => {
  const results = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name === 'unpackage') continue
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      results.push(...listFiles(full))
    } else if (/\.(scss|vue)$/.test(entry.name)) {
      results.push(full)
    }
  }
  return results
}

test('root DESIGN.md is the canonical public UI standard', () => {
  const designPath = join(repoRoot, 'DESIGN.md')
  assert.equal(existsSync(designPath), true, 'DESIGN.md should exist at repo root')

  const source = readFileSync(designPath, 'utf8')
  for (const section of ['适用范围', '品牌原则', '颜色 Token', '亮暗主题', '组件规范', '响应式与可访问性', '页面验收清单', '视觉 QA 矩阵']) {
    assert.match(source, new RegExp(section), `DESIGN.md should contain the "${section}" section`)
  }
})

test('App exposes the canonical semantic token set', () => {
  const appSource = readFileSync(join(appRoot, 'App.vue'), 'utf8')
  const tokens = [
    '--ui-brand-primary',
    '--ui-brand-strong',
    '--ui-surface-page',
    '--ui-surface-card',
    '--ui-surface-subtle',
    '--ui-text-primary',
    '--ui-text-secondary',
    '--ui-border-default',
    '--ui-success',
    '--ui-warning',
    '--ui-danger',
    '--ui-info',
    '--ui-radius-lg',
    '--ui-shadow-primary'
  ]
  for (const token of tokens) {
    assert.match(appSource, new RegExp(`${token}\\s*:`), `App.vue should define ${token}`)
  }
})

test('legacy cool blue-black surfaces do not reappear as foundational backgrounds', () => {
  const offenders = []
  for (const file of listFiles(appRoot)) {
    const relative = file.replace(appRoot, '').replaceAll('\\', '/')
    if (ALLOWED_SURFACE_FILES.has(relative)) continue

    const source = readFileSync(file, 'utf8')
    const declarations = source.match(/background(?:-color)?\s*:[^;{}]+/g) || []
    for (const declaration of declarations) {
      const hit = PROHIBITED_SURFACES.find((color) => declaration.toUpperCase().includes(color.toUpperCase()))
      if (hit) {
        offenders.push(`${relative} -> ${declaration.trim()}`)
        break
      }
    }
  }

  assert.deepEqual(offenders, [], 'prohibited cool surfaces found:\n' + offenders.join('\n'))
})
