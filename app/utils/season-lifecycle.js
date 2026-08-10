const DAY_MS = 24 * 60 * 60 * 1000
const SHANGHAI_OFFSET_MS = 8 * 60 * 60 * 1000

const parseInstant = (value) => {
  const parsed = Date.parse(String(value || ''))
  return Number.isFinite(parsed) ? parsed : NaN
}

const parseBusinessDate = (value, addDays = 0) => {
  const matched = String(value || '').match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (!matched) return NaN

  const year = Number(matched[1])
  const month = Number(matched[2])
  const day = Number(matched[3])
  const date = new Date(Date.UTC(year, month - 1, day))
  if (date.getUTCFullYear() !== year || date.getUTCMonth() !== month - 1 || date.getUTCDate() !== day) return NaN

  return date.getTime() + addDays * DAY_MS - SHANGHAI_OFFSET_MS
}

export const resolveSeasonTimeline = (season = {}, now = Date.now()) => {
  const wireStartAt = parseInstant(season?.start_at)
  const wireEndExclusive = parseInstant(season?.end_at_exclusive)
  const startAt = Number.isFinite(wireStartAt) ? wireStartAt : parseBusinessDate(season?.start_date)
  const endExclusive = Number.isFinite(wireEndExclusive) ? wireEndExclusive : parseBusinessDate(season?.end_date, 1)
  const currentAt = now instanceof Date ? now.getTime() : Number(now)
  if (!Number.isFinite(startAt) || !Number.isFinite(endExclusive) || !Number.isFinite(currentAt) || endExclusive <= startAt) {
    return { remainDays: 0, progressPercent: 0 }
  }

  const remainDays = Math.max(0, Math.ceil((endExclusive - currentAt) / DAY_MS))
  const progressPercent = Math.min(100, Math.max(0, ((currentAt - startAt) / (endExclusive - startAt)) * 100))
  return { remainDays, progressPercent }
}
