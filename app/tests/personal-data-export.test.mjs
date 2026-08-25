import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

import {
  buildPersonalDataExportFilename,
  exportPersonalDataToWriter
} from '../utils/personal-data-export.js'
import {
  createPersonalDataExportFileWriter,
  getPersonalDataExportCapability
} from '../utils/personal-data-export-file.js'

const createWriter = () => {
  const chunks = []
  let cleaned = false
  return {
    chunks,
    get cleaned() {
      return cleaned
    },
    async write(chunk) {
      chunks.push(chunk)
    },
    async complete() {
      return { filePath: '/tmp/personal-data.json' }
    },
    async cleanup() {
      cleaned = true
    }
  }
}

test('personal data export writes all cursor pages as one complete JSON document', async () => {
  const writer = createWriter()
  const requestedCursors = []
  const progress = []
  const pages = {
    '': {
      success: true,
      format_version: 'personal-data-export/v1',
      snapshot_at: '2026-08-20T12:00:00Z',
      item_count: 1,
      complete: false,
      next_cursor: 'next-page',
      items: [{ category: 'profile', id: '1', data_json: '{"nickname":"用户"}' }]
    },
    'next-page': {
      success: true,
      format_version: 'personal-data-export/v1',
      snapshot_at: '2026-08-20T12:00:00Z',
      item_count: 1,
      complete: true,
      items: [{ category: 'matches', id: '9', data_json: '{"status":"completed"}' }]
    }
  }

  const result = await exportPersonalDataToWriter({
    userId: 88,
    writer,
    requestPage: async (cursor) => {
      requestedCursors.push(cursor)
      return pages[cursor]
    },
    onProgress: (state) => progress.push(state)
  })

  assert.deepEqual(requestedCursors, ['', 'next-page'])
  assert.equal(result.itemCount, 2)
  assert.equal(result.pageCount, 2)
  assert.deepEqual(progress, [
    { pageCount: 1, itemCount: 1, complete: false },
    { pageCount: 2, itemCount: 2, complete: true }
  ])
  assert.deepEqual(JSON.parse(writer.chunks.join('')), {
    format_version: 'personal-data-export/v1',
    generated_at: '2026-08-20T12:00:00Z',
    account_id: '88',
    items: [
      { category: 'profile', id: '1', data: { nickname: '用户' } },
      { category: 'matches', id: '9', data: { status: 'completed' } }
    ],
    item_count: 2,
    complete: true
  })
})

test('personal data export cleans incomplete files on invalid cursor state or secret fields', async () => {
  const brokenCursorWriter = createWriter()
  await assert.rejects(
    exportPersonalDataToWriter({
      userId: 88,
      writer: brokenCursorWriter,
      requestPage: async () => ({
        success: true,
        format_version: 'personal-data-export/v1',
        snapshot_at: '2026-08-20T12:00:00Z',
        item_count: 0,
        complete: false,
        items: []
      })
    }),
    /游标状态无效/
  )
  assert.equal(brokenCursorWriter.cleaned, true)

  const secretWriter = createWriter()
  await assert.rejects(
    exportPersonalDataToWriter({
      userId: 88,
      writer: secretWriter,
      requestPage: async () => ({
        success: true,
        format_version: 'personal-data-export/v1',
        snapshot_at: '2026-08-20T12:00:00Z',
        item_count: 1,
        complete: true,
        items: [{ category: 'profile', id: '1', data_json: '{"access_token":"must-not-export"}' }]
      })
    }),
    /不允许的数据/
  )
  assert.equal(secretWriter.cleaned, true)
})

test('personal data export uses a date and account scoped filename', () => {
  assert.equal(
    buildPersonalDataExportFilename(88, new Date(2026, 7, 20, 9, 8, 7)),
    'chasing-points-personal-data-88-20260820-090807.json'
  )
})

test('mini-program file writer appends UTF-8 chunks and supports cleanup', async () => {
  const files = new Map()
  const fileSystem = {
    writeFileSync(path, chunk) {
      files.set(path, chunk)
    },
    appendFileSync(path, chunk) {
      files.set(path, (files.get(path) || '') + chunk)
    },
    unlinkSync(path) {
      files.delete(path)
    }
  }
  const runtime = {
    wx: {
      env: { USER_DATA_PATH: '/user-data' },
      getFileSystemManager: () => fileSystem
    }
  }
  assert.equal(getPersonalDataExportCapability(runtime).supported, true)
  const writer = await createPersonalDataExportFileWriter({ fileName: 'export.json', runtime })
  await writer.write('{"items":[')
  await writer.write('1]}')
  const saved = await writer.complete()
  assert.equal(saved.filePath, '/user-data/export.json')
  assert.equal(files.get(saved.filePath), '{"items":[1]}')
  await writer.cleanup()
  assert.equal(files.has(saved.filePath), false)
})

test('App/Harmony file writer replaces old content, appends chunks, and removes failures', async () => {
  let content = 'old-content'
  let removed = false
  let position = 0
  const writer = {
    length: content.length,
    onwriteend: null,
    onerror: null,
    truncate(size) {
      content = content.slice(0, size)
      this.length = content.length
      this.onwriteend?.()
    },
    seek(nextPosition) {
      position = nextPosition
    },
    write(chunk) {
      content = content.slice(0, position) + chunk + content.slice(position + chunk.length)
      this.length = content.length
      this.onwriteend?.()
    }
  }
  const entry = {
    toURL: () => 'file:///private/export.json',
    createWriter: (success) => success(writer),
    remove: (success) => {
      removed = true
      success()
    }
  }
  const runtime = {
    plus: {
      io: {
        PRIVATE_DOC: '_doc',
        resolveLocalFileSystemURL: (_root, success) => success({
          getFile: (_name, _options, resolve) => resolve(entry)
        })
      }
    }
  }
  const fileWriter = await createPersonalDataExportFileWriter({ fileName: 'export.json', runtime })
  await fileWriter.write('{')
  await fileWriter.write('}')
  assert.equal(content, '{}')
  assert.deepEqual(await fileWriter.complete(), { filePath: 'file:///private/export.json', platform: 'app' })
  await fileWriter.cleanup()
  assert.equal(removed, true)
})

test('unsupported runtimes fail before beginning a personal data export', () => {
  assert.deepEqual(getPersonalDataExportCapability({}), {
    supported: false,
    message: '当前设备暂不支持保存个人数据文件'
  })
})

test('file adapter contains explicit App/Harmony and mini-program paths', () => {
  const source = readFileSync(new URL('../utils/personal-data-export-file.js', import.meta.url), 'utf8')
  assert.match(source, /wxApi\?\.getFileSystemManager/)
  assert.match(source, /plusApi\?\.io\?\.resolveLocalFileSystemURL/)
  assert.match(source, /PRIVATE_DOC/)
  assert.match(source, /openDocument/)
})
