import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8')

const detailSource = read('subPages/match/matchDetail.vue')
const resultSource = read('subPages/match/matchResult.vue')
const cardSource = read('components/matchGearCard.vue')

test('both match pages mount the card through the same enabled rule and never call the gear API themselves', () => {
  for (const [name, source] of [['matchDetail', detailSource], ['matchResult', resultSource]]) {
    assert.match(source, /import matchGearCard from '@\/components\/matchGearCard\.vue'/, name)
    assert.match(source, /<matchGearCard :match-id="matchId" :enabled="showMatchGear" \/>/, name)
    assert.match(source, /shouldShowMatchGear\(\{/, name)
    assert.doesNotMatch(source, /@\/api\/gear\.js/, `${name} 的装备读取只能放在卡片里`)
  }
})

test('the page rule hides the card for spectators and referees and only opens it for completed matches', () => {
  assert.match(detailSource, /isSpectate:\s*isSpectateMode\.value/)
  assert.match(detailSource, /viewerRole:\s*matchData\.value\.viewer_role/)
  assert.match(detailSource, /status:\s*matchData\.value\.status/)
  assert.match(resultSource, /viewerRole:\s*matchData\.value\.viewer_role/)
  assert.match(resultSource, /status:\s*matchData\.value\.status/)
})

test('the card adds no websocket subscription and no extra page lifecycle reads', () => {
  assert.doesNotMatch(cardSource, /websocket|matchWS|WS_MESSAGE_TYPES/)
  assert.doesNotMatch(cardSource, /onShow|onLoad/)
  // 对局页自己的生命周期里不得出现装备读取
  assert.doesNotMatch(detailSource, /getGearMatch|useGearStore/)
  assert.doesNotMatch(resultSource, /getGearMatch|useGearStore/)
})

test('the card keeps reads quiet so a failure stays inside the card', () => {
  assert.match(cardSource, /getGearMatch\(params, \{ silent: true \}\)/)
})

test('gear pages are registered in their own subpackage and never in the main package', () => {
  const pages = read('pages.json')
  const mainPages = pages.slice(0, pages.indexOf('"subPackages"'))
  assert.doesNotMatch(mainPages, /subPages\/gear/)
  assert.match(pages, /"root": "subPages\/gear"/)
})
