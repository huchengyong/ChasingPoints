import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const readSource = (path) => readFileSync(new URL(path, import.meta.url), 'utf8')

// 深色规则与 .checkbox.checked 优先级相同且写在后面，必须在深色规则内重新声明勾选态
const darkCheckboxSources = [
  ['welcome', readSource('../pages/welcome/index.scss'), '\n.dark-mode {'],
  ['login', readSource('../pages/login/login.vue'), '.dark-mode .checkbox {'],
  ['bind phone', readSource('../components/bindPhone.vue'), '&.dark-mode {']
]

for (const [name, source, darkMarker] of darkCheckboxSources) {
  test(`${name} agreement checkbox keeps the checked fill in dark mode`, () => {
    const darkSource = source.slice(source.indexOf(darkMarker))
    assert.match(darkSource, /\.checkbox \{[^}]*&\.checked \{\s*background: #f7d86a;\s*border-color: #f7d86a;/)
  })
}
