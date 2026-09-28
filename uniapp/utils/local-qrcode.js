import UQRCode from 'uqrcodejs'

const MIN_QR_SIZE = 120
const MAX_QR_SIZE = 1024

export const buildLocalQRCodePlan = ({
  canvasId = '',
  content = '',
  size = 240
} = {}) => {
  const normalizedCanvasId = String(canvasId || '').trim()
  const normalizedContent = String(content || '').trim()
  const normalizedSize = Math.min(
    MAX_QR_SIZE,
    Math.max(MIN_QR_SIZE, Number(size) || 240)
  )

  if (!normalizedCanvasId) {
    throw new Error('二维码画布不可用')
  }
  if (!normalizedContent) {
    throw new Error('二维码内容无效')
  }

  return {
    canvasId: normalizedCanvasId,
    content: normalizedContent,
    size: normalizedSize
  }
}

export const renderLocalQRCode = ({
  canvasId,
  content,
  size,
  component,
  uniApi = globalThis.uni,
  QRCode = UQRCode
} = {}) => {
  const plan = buildLocalQRCodePlan({ canvasId, content, size })
  if (!uniApi || typeof uniApi.createCanvasContext !== 'function') {
    throw new Error('当前端暂不支持本地二维码')
  }

  const qr = new QRCode()
  qr.data = plan.content
  qr.size = plan.size
  qr.make()
  qr.canvasContext = component === undefined
    ? uniApi.createCanvasContext(plan.canvasId)
    : uniApi.createCanvasContext(plan.canvasId, component)
  qr.drawCanvas()
  return plan
}
