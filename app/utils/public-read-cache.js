const LOCATION_BUCKET_PRECISION = 1000

const sortValue = (value) => {
  if (Array.isArray(value)) return value.map(sortValue)
  if (!value || typeof value !== 'object') return value
  return Object.keys(value).sort().reduce((result, key) => {
    result[key] = sortValue(value[key])
    return result
  }, {})
}

export const buildPublicReadKey = (prefix, params = {}) => (
  `${prefix}:${JSON.stringify(sortValue(params || {}))}`
)

const bucketCoordinate = (value) => {
  const coordinate = Number(value)
  if (!Number.isFinite(coordinate)) return 0
  return Math.round(coordinate * LOCATION_BUCKET_PRECISION) / LOCATION_BUCKET_PRECISION
}

export const buildNearbyVenueCacheKey = ({ latitude, longitude, radius, limit } = {}) => buildPublicReadKey('nearby', {
  latitude: bucketCoordinate(latitude),
  longitude: bucketCoordinate(longitude),
  radius: Number(radius) || 0,
  limit: Number(limit) || 0
})
