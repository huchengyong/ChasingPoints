import test from 'node:test'
import assert from 'node:assert/strict'

import { readFileSync } from 'node:fs'

import {
  drawPosterQrCode,
  generateMatchPoster,
  resolveMatchPosterTemplate,
  resolvePkSummaryLayout,
  resolvePosterQrTarget
} from '../utils/posterGenerator.js'

test('resolvePkSummaryLayout leaves comfortable bottom padding for two-line summary', () => {
  const layout = resolvePkSummaryLayout('这是一段足够长的系统结论文案，用来验证第二行文本不会再贴着卡片底部显示')

  assert.equal(layout.summaryLines.length, 2)
  assert.ok(layout.cardHeight > 150)

  const lastLine = layout.summaryLines.at(-1)
  const bottomPadding = layout.cardY + layout.cardHeight - lastLine.y
  assert.ok(bottomPadding >= 28)
})

// ---------- 战绩海报：标准模板 / 会员模板 ----------


const shareData = {
  game_type: 3,
  game_type_name: '中式八球',
  my_name: '张三',
  opponent_name: '李四',
  my_avatar: 'https://img.example/me.png',
  opponent_avatar: 'https://img.example/op.png',
  my_score: 5,
  opponent_score: 3,
  result: '胜利',
  rank_change: 12,
  match_time: '2026-10-22 20:00',
  summary_highlights: [{ label: '炸清', value: '2次', tone: 'gold' }],
  summary_stats: [{ label: '总局数', value: '8', tone: 'blue' }]
}

// 记录全部画布调用；imageResult 控制 getImageInfo 成功或失败，exportFails 控制导出失败。
async function drawPoster(data, { failImages = false, exportFails = false } = {}) {
  const calls = []
  const ctx = new Proxy({}, {
    get: (_, name) => {
      if (name === 'createLinearGradient') {
        return (...args) => {
          calls.push([name, ...args])
          return { addColorStop: (...stop) => calls.push(['addColorStop', ...stop]) }
        }
      }
      if (name === 'draw') return (_reserve, done) => done()
      return (...args) => { calls.push([name, ...args]) }
    }
  })
  const exported = []
  const previous = globalThis.uni
  globalThis.uni = {
    createCanvasContext: () => ctx,
    getImageInfo: ({ src, success, fail }) => (failImages ? fail({ errMsg: 'fail' }) : success({ path: `local:${src}` })),
    canvasToTempFilePath: (options) => {
      exported.push(options)
      return exportFails ? options.fail({ errMsg: 'export failed' }) : options.success({ tempFilePath: '/tmp/poster.png' })
    }
  }
  try {
    const path = await generateMatchPoster('poster', data)
    return { calls, exported, path }
  } finally {
    globalThis.uni = previous
  }
}

const texts = (calls) => calls.filter(([name]) => name === 'fillText').map(([, text]) => text)
const images = (calls) => calls.filter(([name]) => name === 'drawImage').map(([, src]) => src)

test('resolveMatchPosterTemplate picks the member template only for an active sharer', () => {
  const tiers = [1, 2, 3, 4, 5].map((level) => resolveMatchPosterTemplate({ my_vip_level: level, my_rank_level: 4 }))
  assert.deepEqual(tiers.map((item) => item.tier), ['standard', 'standard', 'advanced', 'advanced', 'premium'])
  assert.deepEqual(tiers.map((item) => item.myBadge.text), ['VIP1', 'VIP2', 'VIP3', 'VIP4', 'VIP5'])
  assert.ok(tiers.every((item) => item.member && item.headerExtra > 0))
  assert.equal(tiers[4].frameUrl, 'https://cdn.zhuifen.cn/static/images/avatar-frames/frame_platinum.png')

  for (const data of [undefined, null, {}, { my_vip_level: 0 }, { my_vip_level: 'x' }, { my_vip_level: -1 }]) {
    const template = resolveMatchPosterTemplate(data)
    assert.equal(template.member, false, JSON.stringify(data))
    assert.equal(template.theme, null)
    assert.equal(template.headerExtra, 0)
    assert.equal(template.myBadge, null)
    assert.equal(template.frameUrl, '')
  }
})

test('resolveMatchPosterTemplate marks the opponent badge independently of the sharer', () => {
  const freeVsMember = resolveMatchPosterTemplate({ my_vip_level: 0, opponent_vip_level: 5 })
  assert.equal(freeVsMember.member, false)
  assert.deepEqual(freeVsMember.opponentBadge, { text: 'VIP5', tier: 'premium' })

  const memberVsFree = resolveMatchPosterTemplate({ my_vip_level: 3, opponent_vip_level: 0 })
  assert.equal(memberVsFree.member, true)
  assert.equal(memberVsFree.opponentBadge, null)

  // 没有头像框的会员（例如没有段位）依然使用会员模板，只是不画框。
  const unranked = resolveMatchPosterTemplate({ my_vip_level: 2, my_rank_level: 0 })
  assert.equal(unranked.member, true)
  assert.equal(unranked.frameUrl, '')
})

test('free sharer keeps the standard poster: no avatar, no frame, no VIP text, slate background', async () => {
  const { calls, exported } = await drawPoster({ ...shareData, my_vip_level: 0, opponent_vip_level: 0 })
  assert.deepEqual(images(calls), [])
  assert.ok(!texts(calls).some((text) => /^VIP/.test(text)))
  const gradient = calls.filter(([name]) => name === 'addColorStop').slice(0, 2).map(([, , color]) => color)
  assert.deepEqual(gradient, ['#0f172a', '#111827'])
  assert.equal(exported[0].height, 1320)
})

test('free sharer against a member shows only the opponent VIP badge on the standard template', async () => {
  const standard = await drawPoster({ ...shareData })
  const { calls, exported } = await drawPoster({ ...shareData, my_vip_level: 0, opponent_vip_level: 3 })
  assert.deepEqual(texts(calls).filter((text) => /^VIP/.test(text)), ['VIP3'])
  assert.deepEqual(images(calls), [], 'the standard template never draws avatars or frames')
  assert.equal(exported[0].height, standard.exported[0].height, 'the standard layout must not move')
  const gradient = calls.filter(([name]) => name === 'addColorStop').slice(0, 2).map(([, , color]) => color)
  assert.deepEqual(gradient, ['#0f172a', '#111827'])
})

test('Lv5 member gets the premium template with own frame and VIP badge', async () => {
  const { calls, exported, path } = await drawPoster({ ...shareData, my_vip_level: 5, my_rank_level: 6 })
  assert.equal(path, '/tmp/poster.png')
  const gradient = calls.filter(([name]) => name === 'addColorStop').slice(0, 2).map(([, , color]) => color)
  assert.deepEqual(gradient, ['#3C2B08', '#8F6507'])
  assert.ok(images(calls).includes('local:https://cdn.zhuifen.cn/static/images/avatar-frames/frame_king.png'))
  assert.ok(images(calls).includes('local:https://img.example/me.png'), 'own avatar is drawn inside the frame')
  assert.ok(images(calls).includes('local:https://img.example/op.png'), 'opponent avatar is drawn without a frame')
  assert.equal(images(calls).filter((src) => src.includes('avatar-frames')).length, 1, 'only the sharer gets a frame')
  assert.deepEqual(texts(calls).filter((text) => /^VIP/.test(text)), ['VIP5'])
  assert.equal(exported[0].height, 1320 + resolveMatchPosterTemplate({ my_vip_level: 5 }).headerExtra, 'the header grows to fit the avatar row and everything below moves down')
})

test('member template moves every section below the header by the same offset', async () => {
  const standard = await drawPoster({ ...shareData })
  const member = await drawPoster({ ...shareData, my_vip_level: 2, my_rank_level: 3 })
  const extra = resolveMatchPosterTemplate({ my_vip_level: 2 }).headerExtra
  const placed = (calls) => calls.filter(([name]) => name === 'fillText').map(([, text, x, y]) => `${text}@${x},${y}`)
  const memberPlaced = new Set(placed(member.calls))
  const headerBottom = 306 // 名字基线及以下的内容都属于要下移的部分；标题与结果标签在它上方
  let checked = 0
  for (const [name, text, x, y] of standard.calls) {
    if (name !== 'fillText' || y < headerBottom || ['追分', '真实对局数据生成，仅供复盘与分享'].includes(text)) continue
    checked += 1
    assert.ok(memberPlaced.has(`${text}@${x},${y + extra}`), `${text} must move down by ${extra}`)
  }
  assert.ok(checked >= 10, 'the comparison must cover the whole body, not a few labels')
  // 页脚跟着总高度走，同样下移。
  const footer = (calls) => calls.find(([name, text]) => name === 'fillText' && text === '追分')[3]
  assert.equal(footer(member.calls) - footer(standard.calls), extra)
})

test('member against a member shows both VIP badges, member against free shows only their own', async () => {
  const both = await drawPoster({ ...shareData, my_vip_level: 3, my_rank_level: 4, opponent_vip_level: 5 })
  assert.deepEqual(texts(both.calls).filter((text) => /^VIP/.test(text)), ['VIP3', 'VIP5'])
  const mineOnly = await drawPoster({ ...shareData, my_vip_level: 3, my_rank_level: 4, opponent_vip_level: 0 })
  assert.deepEqual(texts(mineOnly.calls).filter((text) => /^VIP/.test(text)), ['VIP3'])
})

test('a failed image read only drops that item: placeholder avatar, no frame, badge still drawn', async () => {
  const { calls, path } = await drawPoster({ ...shareData, my_vip_level: 4, my_rank_level: 5 }, { failImages: true })
  assert.equal(path, '/tmp/poster.png')
  assert.deepEqual(images(calls), [])
  assert.deepEqual(texts(calls).filter((text) => /^VIP/.test(text)), ['VIP4'])
  // 占位头像用名字首字。
  assert.ok(texts(calls).includes('张'))
  assert.ok(texts(calls).includes('李'))
})

test('a member template that cannot be exported rejects so the page falls back to the existing poster', async () => {
  await assert.rejects(() => drawPoster({ ...shareData, my_vip_level: 5, my_rank_level: 6 }, { exportFails: true }), { errMsg: 'export failed' })
})

test('the share page still degrades to the existing fallback poster when generation fails', () => {
  const source = readFileSync(new URL('../subPages/match/shareResult.vue', import.meta.url), 'utf8')
  assert.match(source, /posterPath\.value = await generateMatchPoster\('matchPoster', shareData\)/)
  assert.match(source, /catch \(e\) \{[\s\S]*posterPath\.value = await generateFallbackPoster\(\)/)
  // 降级海报不带会员字段，始终是标准模板。
  assert.doesNotMatch(source.slice(source.indexOf('const generateFallbackPoster')), /vip_level/)
})

// ---------- 海报二维码 ----------

test('poster QR points at the match for public matches and at the home page otherwise', () => {
  assert.deepEqual(resolvePosterQrTarget({ visibility: 'public', match_id: 501 }), {
    url: 'https://www.zhuifen.cn/go/?m=501',
    caption: '微信扫码查看这场比赛'
  })

  const home = { url: 'https://www.zhuifen.cn/go/', caption: '微信扫码打开追分' }
  assert.deepEqual(resolvePosterQrTarget({ visibility: 'private', match_id: 501 }), home)
  assert.deepEqual(resolvePosterQrTarget({ match_id: 501 }), home)
  assert.deepEqual(resolvePosterQrTarget({ visibility: 'public' }), home)
  assert.deepEqual(resolvePosterQrTarget(), home)
})

test('a QR that cannot be encoded is skipped without throwing', () => {
  const fillRects = []
  const ctx = { setFillStyle: () => {}, fillRect: (...args) => fillRects.push(args) }

  assert.equal(drawPosterQrCode(ctx, `https://www.zhuifen.cn/go/?j=${'a'.repeat(6000)}`, { x: 0, y: 0, size: 100 }), false)
  assert.equal(fillRects.length, 0)
})

test('the match poster still generates with the QR caption of its own visibility', async () => {
  const publicPoster = await drawPoster({ ...shareData, visibility: 'public', match_id: 501 })
  assert.equal(publicPoster.path, '/tmp/poster.png')
  assert.ok(texts(publicPoster.calls).includes('微信扫码查看这场比赛'))

  const privatePoster = await drawPoster({ ...shareData, visibility: 'private', match_id: 501 })
  assert.equal(privatePoster.path, '/tmp/poster.png')
  assert.ok(texts(privatePoster.calls).includes('微信扫码打开追分'))
})

// 按画布调用还原文字的大致占位（中文按每字一个字号宽），检查二维码区和其他文字互不压叠。
const textBoxes = (calls) => {
  let fontSize = 16
  let align = 'left'
  const boxes = []
  for (const [name, ...args] of calls) {
    if (name === 'setFontSize') fontSize = args[0]
    if (name === 'setTextAlign') align = args[0]
    if (name === 'fillText') {
      const [text, x, y] = args
      const width = Array.from(String(text)).length * fontSize
      const left = align === 'center' ? x - width / 2 : align === 'right' ? x - width : x
      boxes.push({ text, left, right: left + width, top: y - fontSize, bottom: y })
    }
  }
  return boxes
}

const overlaps = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom

test('the poster QR block does not overlap the brand line, footnote or data cards', async () => {
  for (const data of [
    { ...shareData, visibility: 'public', match_id: 501 },
    { ...shareData, visibility: 'public', match_id: 501, summary_stats: Array.from({ length: 6 }, (_, i) => ({ label: `项${i}`, value: '1', tone: 'blue' })) }
  ]) {
    const { calls } = await drawPoster(data)
    const qrRect = calls.find(([name, , , w, h]) => name === 'fillRect' && w === 120 && h === 120)
    assert.ok(qrRect, '应绘制 120 尺寸的二维码底块')
    const [, x, y, size] = qrRect
    const qrBox = { left: x, right: x + size, top: y, bottom: y + size }
    const captions = new Set(['微信扫码查看这场比赛', '微信扫码打开追分', '微信扫一扫进入小程序'])
    const boxes = textBoxes(calls)
    const captionBoxes = boxes.filter((box) => captions.has(box.text))
    assert.equal(captionBoxes.length, 2)
    for (const box of boxes.filter((item) => !captions.has(item.text))) {
      assert.equal(overlaps(qrBox, box), false, `二维码压住了「${box.text}」`)
      for (const caption of captionBoxes) {
        assert.equal(overlaps(caption, box), false, `「${caption.text}」压住了「${box.text}」`)
      }
    }
  }
})
