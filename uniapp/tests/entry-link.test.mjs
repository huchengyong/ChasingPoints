import test from 'node:test'
import assert from 'node:assert/strict'

import { buildEntryLink, parseEntryLink, decodeLaunchQuery } from '../utils/entry-link.js'

const code = 'Ab3dEf6hIj9k'

test('entry link builds and parses all three types', () => {
  const invite = buildEntryLink({ type: 'invite', value: code })
  const match = buildEntryLink({ type: 'match', value: 123 })
  const home = buildEntryLink()

  assert.equal(invite, `https://www.zhuifen.cn/go/?j=${code}`)
  assert.equal(match, 'https://www.zhuifen.cn/go/?m=123')
  assert.equal(home, 'https://www.zhuifen.cn/go/')

  assert.deepEqual(parseEntryLink(invite), { type: 'invite', inviteCode: code })
  assert.deepEqual(parseEntryLink(match), { type: 'match', matchId: 123 })
  assert.deepEqual(parseEntryLink(home), { type: 'home' })
})

test('entry link accepts path with or without trailing slash', () => {
  assert.deepEqual(parseEntryLink(`https://www.zhuifen.cn/go?j=${code}`), { type: 'invite', inviteCode: code })
  assert.deepEqual(parseEntryLink(`https://www.zhuifen.cn/go/?j=${code}#frag`), { type: 'invite', inviteCode: code })
  assert.deepEqual(parseEntryLink('https://www.zhuifen.cn/go'), { type: 'home' })
})

test('entry link rejects other hosts, protocols and lookalike paths', () => {
  const rejected = [
    `http://www.zhuifen.cn/go/?j=${code}`,
    `https://evil.example.com/go/?j=${code}`,
    `https://zhuifen.cn/go/?j=${code}`,
    `https://sub.www.zhuifen.cn/go/?j=${code}`,
    `https://www.zhuifen.cn/gox?j=${code}`,
    `https://www.zhuifen.cn/go/extra?j=${code}`,
    `https://www.zhuifen.cn.evil.com/go/?j=${code}`,
    'not a link',
    ''
  ]
  rejected.forEach((raw) => {
    assert.deepEqual(parseEntryLink(raw), { type: 'other' }, raw)
  })
})

test('entry link falls back to home when params are invalid or ambiguous', () => {
  const invalid = [
    `https://www.zhuifen.cn/go/?j=${code}&m=1`,
    'https://www.zhuifen.cn/go/?j=Ab3dEf6hIj9',
    'https://www.zhuifen.cn/go/?j=Ab3dEf6hIj9kk',
    'https://www.zhuifen.cn/go/?j=Ab3d-Ef6hIj9',
    'https://www.zhuifen.cn/go/?m=abc',
    'https://www.zhuifen.cn/go/?m=0',
    'https://www.zhuifen.cn/go/?m=-1',
    'https://www.zhuifen.cn/go/?m=1.5',
    'https://www.zhuifen.cn/go/?m=',
    'https://www.zhuifen.cn/go/?unknown=1'
  ]
  invalid.forEach((raw) => {
    assert.deepEqual(parseEntryLink(raw), { type: 'home' }, raw)
  })
})

test('launch query decodes once or twice and survives bad input', () => {
  const invite = `https://www.zhuifen.cn/go/?j=${code}`
  assert.equal(decodeLaunchQuery(encodeURIComponent(invite)), invite)
  assert.equal(decodeLaunchQuery(encodeURIComponent(encodeURIComponent(invite))), invite)
  assert.equal(decodeLaunchQuery(invite), invite)
  assert.equal(decodeLaunchQuery(''), '')
  assert.equal(decodeLaunchQuery(undefined), '')
  assert.equal(decodeLaunchQuery('%E0%A4%A'), '%E0%A4%A')
})
