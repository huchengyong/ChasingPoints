const DEFAULT_QINIU_UPLOAD_URL = 'https://up-z2.qiniup.com'
export const AVATAR_IMAGE_MAX_EDGE = 512
export const AVATAR_IMAGE_QUALITY = 80

const padNumber = (value) => String(value).padStart(2, '0')

export const formatSettingsPhone = (phone = '') => {
  const value = String(phone || '').trim()
  if (!value) return ''

  if (/^\d{11}$/.test(value)) {
    return `+86 ${value.slice(0, 3)}****${value.slice(7)}`
  }

  if (/^\d{3}\*{4}\d{4}$/.test(value)) {
    return `+86 ${value}`
  }

  if (value.startsWith('+86 ')) return value
  if (value.startsWith('+')) return value
  return `+86 ${value}`
}

export const getFileExtensionFromPath = (filePath = '') => {
  const value = String(filePath || '').split('?')[0]
  const lastDotIndex = value.lastIndexOf('.')
  if (lastDotIndex < 0) return '.jpg'

  const ext = value.slice(lastDotIndex).toLowerCase()
  if (!/^\.[a-z0-9]+$/.test(ext)) return '.jpg'
  return ext
}

export const buildQiniuAvatarObjectKey = ({
  userId = 0,
  filePath = '',
  timestamp = Date.now()
} = {}) => {
  const date = new Date(timestamp)
  const dateSegment = `${date.getFullYear()}${padNumber(date.getMonth() + 1)}${padNumber(date.getDate())}`
  const ext = getFileExtensionFromPath(filePath)
  return `avatars/${Number(userId) || 0}/${dateSegment}/${Number(timestamp) || Date.now()}${ext}`
}

export const normalizeQiniuUploadResult = ({ domain = '', key = '' } = {}) => {
  const normalizedDomain = String(domain || '').trim().replace(/\/+$/, '')
  const normalizedKey = String(key || '').trim().replace(/^\/+/, '')
  if (!normalizedDomain || !normalizedKey) return ''
  return `${normalizedDomain}/${normalizedKey}`
}

export const normalizeQiniuUploadTokenResponse = (payload = {}) => ({
  uploadUrl: payload.upload_url || payload.uploadUrl || DEFAULT_QINIU_UPLOAD_URL,
  uploadToken: payload.upload_token || payload.uploadToken || '',
  key: payload.key || '',
  domain: payload.domain || ''
})

export const getAvatarCompressionDimensions = ({
  width = 0,
  height = 0,
  maxEdge = AVATAR_IMAGE_MAX_EDGE
} = {}) => {
  const normalizedWidth = Number(width) || 0
  const normalizedHeight = Number(height) || 0
  const normalizedMaxEdge = Number(maxEdge) || AVATAR_IMAGE_MAX_EDGE

  if (normalizedWidth <= 0 || normalizedHeight <= 0) {
    return {
      width: normalizedMaxEdge,
      height: normalizedMaxEdge
    }
  }

  const longestEdge = Math.max(normalizedWidth, normalizedHeight)
  if (longestEdge <= normalizedMaxEdge) {
    return {
      width: normalizedWidth,
      height: normalizedHeight
    }
  }

  const scale = normalizedMaxEdge / longestEdge
  return {
    width: Math.max(1, Math.round(normalizedWidth * scale)),
    height: Math.max(1, Math.round(normalizedHeight * scale))
  }
}

const getImageInfo = (uniApi, src) => new Promise((resolve, reject) => {
  uniApi.getImageInfo({
    src,
    success: resolve,
    fail: reject
  })
})

const compressImage = (uniApi, options) => new Promise((resolve, reject) => {
  uniApi.compressImage({
    ...options,
    success: resolve,
    fail: reject
  })
})

export const prepareAvatarForUpload = async (filePath, uniApi = typeof uni !== 'undefined' ? uni : null) => {
  const normalizedFilePath = String(filePath || '').trim()
  if (!normalizedFilePath) return ''

  if (
    !uniApi ||
    typeof uniApi.getImageInfo !== 'function' ||
    typeof uniApi.compressImage !== 'function'
  ) {
    return normalizedFilePath
  }

  try {
    const imageInfo = await getImageInfo(uniApi, normalizedFilePath)
    const targetSize = getAvatarCompressionDimensions({
      width: imageInfo?.width,
      height: imageInfo?.height
    })

    if (
      Number(targetSize.width || 0) === Number(imageInfo?.width || 0) &&
      Number(targetSize.height || 0) === Number(imageInfo?.height || 0)
    ) {
      return normalizedFilePath
    }

    const compressed = await compressImage(uniApi, {
      src: normalizedFilePath,
      quality: AVATAR_IMAGE_QUALITY,
      compressedWidth: targetSize.width,
      compressedHeight: targetSize.height
    })

    return compressed?.tempFilePath || normalizedFilePath
  } catch (error) {
    return normalizedFilePath
  }
}

export const uploadAvatarToQiniu = ({
  filePath,
  uploadUrl,
  uploadToken,
  key
}) => new Promise((resolve, reject) => {
  uni.uploadFile({
    url: uploadUrl || DEFAULT_QINIU_UPLOAD_URL,
    filePath,
    name: 'file',
    formData: {
      token: uploadToken,
      key
    },
    success: (res) => {
      if (res.statusCode < 200 || res.statusCode >= 300) {
        reject(new Error('上传失败'))
        return
      }

      try {
        const data = typeof res.data === 'string' ? JSON.parse(res.data || '{}') : (res.data || {})
        resolve({
          key: data.key || key,
          hash: data.hash || ''
        })
      } catch (error) {
        reject(new Error('上传结果解析失败'))
      }
    },
    fail: () => {
      reject(new Error('上传失败'))
    }
  })
})
