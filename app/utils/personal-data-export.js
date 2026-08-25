const sensitiveExportKeys = new Set([
  'access_token',
  'refresh_token',
  'credential',
  'authorization',
  'open_id',
  'union_id',
  'push_token',
  'ticket'
])

const exportError = (message, code = 'PERSONAL_DATA_EXPORT_FAILED') => {
  const error = new Error(message)
  error.code = code
  return error
}

export const buildPersonalDataExportFilename = (userId, date = new Date()) => {
  const normalizedUserId = String(userId || '').trim()
  if (!normalizedUserId) throw exportError('缺少导出账号信息')
  const pad = (value) => String(value).padStart(2, '0')
  const timestamp = [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate())
  ].join('') + '-' + [pad(date.getHours()), pad(date.getMinutes()), pad(date.getSeconds())].join('')
  return `chasing-points-personal-data-${normalizedUserId}-${timestamp}.json`
}

const containsSensitiveExportValue = (value) => {
  if (Array.isArray(value)) return value.some(containsSensitiveExportValue)
  if (!value || typeof value !== 'object') return false
  return Object.entries(value).some(([key, nestedValue]) => (
    sensitiveExportKeys.has(String(key).toLowerCase()) || containsSensitiveExportValue(nestedValue)
  ))
}

const normalizeExportItem = (item) => {
  const category = String(item?.category || '').trim()
  const id = String(item?.id || '').trim()
  if (!category || !id || typeof item?.data_json !== 'string') {
    throw exportError('导出分段格式无效')
  }
  let data
  try {
    data = JSON.parse(item.data_json)
  } catch (_) {
    throw exportError('导出分段数据无效')
  }
  if (!data || typeof data !== 'object' || containsSensitiveExportValue(data)) {
    throw exportError('导出内容包含不允许的数据')
  }
  return { category, id, data }
}

const normalizeExportPage = (response, metadata) => {
  if (!response?.success || !Array.isArray(response.items)) {
    throw exportError(response?.message || '导出分段失败')
  }
  const formatVersion = String(response.format_version || '').trim()
  const snapshotAt = String(response.snapshot_at || '').trim()
  if (!formatVersion || !snapshotAt || (metadata && (metadata.formatVersion !== formatVersion || metadata.snapshotAt !== snapshotAt))) {
    throw exportError('导出数据快照不一致')
  }
  if (Number(response.item_count) !== response.items.length) {
    throw exportError('导出分段计数不一致')
  }
  const complete = Boolean(response.complete)
  const nextCursor = String(response.next_cursor || '').trim()
  if ((complete && nextCursor) || (!complete && !nextCursor)) {
    throw exportError('导出游标状态无效')
  }
  return {
    metadata: { formatVersion, snapshotAt },
    items: response.items.map(normalizeExportItem),
    complete,
    nextCursor
  }
}

const assertWriter = (writer) => {
  if (typeof writer?.write !== 'function' || typeof writer?.complete !== 'function' || typeof writer?.cleanup !== 'function') {
    throw exportError('导出文件服务不可用')
  }
}

/**
 * Pulls server-bounded pages and writes a single UTF-8 JSON document without
 * retaining the full export in memory. The writer is removed on every failure.
 */
export const exportPersonalDataToWriter = async ({
  requestPage,
  writer,
  userId,
  fileName = buildPersonalDataExportFilename(userId),
  onProgress = () => {}
} = {}) => {
  if (typeof requestPage !== 'function') throw exportError('导出接口不可用')
  assertWriter(writer)

  let cursor = ''
  let metadata = null
  let itemCount = 0
  let pageCount = 0
  let firstItem = true
  const seenCursors = new Set()

  try {
    while (true) {
      if (cursor && seenCursors.has(cursor)) throw exportError('导出游标重复，已停止生成文件')
      if (cursor) seenCursors.add(cursor)
      const page = normalizeExportPage(await requestPage(cursor), metadata)
      metadata = metadata || page.metadata

      if (pageCount === 0) {
        await writer.write(
          `{"format_version":${JSON.stringify(metadata.formatVersion)},` +
          `"generated_at":${JSON.stringify(metadata.snapshotAt)},` +
          `"account_id":${JSON.stringify(String(userId))},"items":[`
        )
      }
      for (const item of page.items) {
        await writer.write((firstItem ? '' : ',') + JSON.stringify(item))
        firstItem = false
        itemCount += 1
      }
      pageCount += 1
      onProgress({ pageCount, itemCount, complete: page.complete })

      if (page.complete) break
      cursor = page.nextCursor
    }

    await writer.write(`],"item_count":${itemCount},"complete":true}`)
    const saved = await writer.complete()
    return {
      ...metadata,
      fileName,
      itemCount,
      pageCount,
      saved
    }
  } catch (error) {
    try {
      await writer.cleanup()
    } catch (_) {
      // The original export error is more useful to the caller.
    }
    throw error
  }
}
