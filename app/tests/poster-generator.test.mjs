import test from 'node:test'
import assert from 'node:assert/strict'

import { resolvePkSummaryLayout } from '../utils/posterGenerator.js'

test('resolvePkSummaryLayout leaves comfortable bottom padding for two-line summary', () => {
  const layout = resolvePkSummaryLayout('这是一段足够长的系统结论文案，用来验证第二行文本不会再贴着卡片底部显示')

  assert.equal(layout.summaryLines.length, 2)
  assert.ok(layout.cardHeight > 150)

  const lastLine = layout.summaryLines.at(-1)
  const bottomPadding = layout.cardY + layout.cardHeight - lastLine.y
  assert.ok(bottomPadding >= 28)
})
