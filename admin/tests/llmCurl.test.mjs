import test from 'node:test'
import assert from 'node:assert/strict'

import { buildLlmCurl } from '../src/utils/llmCurl.ts'

const extractBody = (curl) => {
  const marker = "-d '"
  const start = curl.indexOf(marker) + marker.length
  return JSON.parse(curl.slice(start, curl.lastIndexOf("'")))
}

test('buildLlmCurl renders the configured endpoint, model and max_tokens', () => {
  const curl = buildLlmCurl({
    endpoint_url: ' https://opencode.ai/inference/anthropic/v1/messages ',
    model: 'claude-haiku-4-5',
    max_tokens: 1024
  })

  assert.ok(curl.startsWith("curl -X POST 'https://opencode.ai/inference/anthropic/v1/messages' \\\n"))
  assert.ok(curl.includes("-H 'Authorization: Bearer <API_KEY>'"))
  assert.ok(curl.includes("-H 'Content-Type: application/json'"))
  assert.deepEqual(extractBody(curl), {
    model: 'claude-haiku-4-5',
    max_tokens: 1024,
    messages: [{ role: 'user', content: 'Say hi' }]
  })
})

test('buildLlmCurl uses placeholders for unfinished fields', () => {
  const curl = buildLlmCurl({ endpoint_url: '', model: '  ', max_tokens: null })

  assert.ok(curl.includes("curl -X POST '<endpoint_url>'"))
  assert.deepEqual(extractBody(curl), {
    model: '<model>',
    max_tokens: '<max_tokens>',
    messages: [{ role: 'user', content: 'Say hi' }]
  })
})

test('buildLlmCurl escapes single quotes so the command stays one shell argument', () => {
  const curl = buildLlmCurl({
    endpoint_url: "https://example.com/it's",
    model: "m'1",
    max_tokens: 8
  })

  assert.ok(curl.includes("'https://example.com/it'\\''s'"))
  assert.ok(curl.includes("\"model\": \"m'\\''1\""))
})

test('buildLlmCurl never emits a real api key', () => {
  const curl = buildLlmCurl({ endpoint_url: 'https://a.com/v1', model: 'm', max_tokens: 1 })

  assert.equal(curl.match(/Bearer (\S+?)'/)?.[1], '<API_KEY>')
})
