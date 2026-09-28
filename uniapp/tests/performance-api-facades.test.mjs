import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const readApi = (file) => readFileSync(new URL(`../api/${file}`, import.meta.url), 'utf8')

test('performance read endpoints stay behind app API facades', () => {
  const expectedRoutes = {
    'user.js': ['/api/user/bootstrap', '/api/user/overview'],
    'match.js': ['/api/match/h2h/overview'],
    'stats.js': ['/api/stats/overview'],
    'season.js': ['/api/season/overview'],
    'rank.js': ['/api/public/rank/leaderboard-summary', '/api/public/rank/configs'],
    'achievement.js': ['/api/achievement/detail'],
    'opponent.js': ['/api/opponent/candidates']
  }

  for (const [file, routes] of Object.entries(expectedRoutes)) {
    const source = readApi(file)
    assert.doesNotMatch(source, /uni\.request/)
    for (const route of routes) {
      assert.match(source, new RegExp(route.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
    }
  }
})
