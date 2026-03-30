/**
 * 海报生成工具
 * 使用 Canvas 绘制分享海报
 */

import { getGameTypeLabel } from './game-types.js'

/**
 * 绘制圆角矩形
 */
function drawRoundRect(ctx, x, y, w, h, r) {
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + w, y, x + w, y + h, r)
  ctx.arcTo(x + w, y + h, x, y + h, r)
  ctx.arcTo(x, y + h, x, y, r)
  ctx.arcTo(x, y, x + w, y, r)
  ctx.closePath()
}

/**
 * 绘制圆形头像
 */
function drawCircleImage(ctx, imgPath, x, y, r) {
  ctx.save()
  ctx.beginPath()
  ctx.arc(x + r, y + r, r, 0, Math.PI * 2)
  ctx.clip()
  ctx.drawImage(imgPath, x, y, r * 2, r * 2)
  ctx.restore()
}

/**
 * 绘制居中文字
 */
function drawCenterText(ctx, text, y, fontSize, color) {
  ctx.setFontSize(fontSize)
  ctx.setFillStyle(color)
  ctx.setTextAlign('center')
  ctx.fillText(text, 375, y)
  ctx.setTextAlign('left')
}

function getToneColors(tone) {
  switch (tone) {
    case 'gold':
      return { accent: '#f59e0b', soft: 'rgba(245, 158, 11, 0.18)' }
    case 'emerald':
    case 'green':
      return { accent: '#10b981', soft: 'rgba(16, 185, 129, 0.16)' }
    case 'red':
      return { accent: '#ef4444', soft: 'rgba(239, 68, 68, 0.16)' }
    case 'purple':
      return { accent: '#8b5cf6', soft: 'rgba(139, 92, 246, 0.16)' }
    case 'orange':
      return { accent: '#f97316', soft: 'rgba(249, 115, 22, 0.16)' }
    case 'cyan':
      return { accent: '#06b6d4', soft: 'rgba(6, 182, 212, 0.16)' }
    case 'slate':
      return { accent: '#94a3b8', soft: 'rgba(148, 163, 184, 0.16)' }
    case 'blue':
    default:
      return { accent: '#3b82f6', soft: 'rgba(59, 130, 246, 0.16)' }
  }
}

function truncateText(text, maxLength = 10) {
  if (!text) return ''
  return text.length > maxLength ? `${text.slice(0, maxLength - 1)}…` : text
}

function drawTextBlock(ctx, text, x, y, maxCharsPerLine, lineHeight, maxLines, color, fontSize) {
  const content = text || ''
  ctx.setFillStyle(color)
  ctx.setFontSize(fontSize)
  for (let index = 0; index < maxLines; index += 1) {
    const start = index * maxCharsPerLine
    if (start >= content.length) break
    let line = content.slice(start, start + maxCharsPerLine)
    if (index === maxLines - 1 && content.length > start + maxCharsPerLine) {
      line = `${line.slice(0, Math.max(maxCharsPerLine - 1, 1))}…`
    }
    ctx.fillText(line, x, y + lineHeight * index)
  }
}

function splitPosterLines(text, maxCharsPerLine, maxLines) {
  const content = text || ''
  const lines = []

  for (let index = 0; index < maxLines; index += 1) {
    const start = index * maxCharsPerLine
    if (start >= content.length) break
    let line = content.slice(start, start + maxCharsPerLine)
    if (index === maxLines - 1 && content.length > start + maxCharsPerLine) {
      line = `${line.slice(0, Math.max(maxCharsPerLine - 1, 1))}…`
    }
    lines.push(line)
  }

  return lines
}

export function resolvePkSummaryLayout(summaryText) {
  const cardY = 790
  const titleY = cardY + 50
  const firstLineY = cardY + 102
  const lineHeight = 44
  const bottomPadding = 34
  const summaryLines = splitPosterLines(
    summaryText || '真实对局数据越多，PK 报表越有说服力。',
    20,
    2
  )
  const lineCount = Math.max(summaryLines.length, 1)
  const cardHeight = firstLineY - cardY + (lineCount - 1) * lineHeight + bottomPadding

  return {
    cardY,
    cardHeight,
    titleY,
    summaryLines: summaryLines.map((line, index) => ({
      text: line,
      y: firstLineY + lineHeight * index
    }))
  }
}

function drawSummaryCard(ctx, item, x, y, w, h) {
  const { accent, soft } = getToneColors(item.tone)
  drawRoundRect(ctx, x, y, w, h, 24)
  ctx.setFillStyle(soft)
  ctx.fill()

  ctx.setFillStyle(accent)
  ctx.fillRect(x, y, 10, h)

  ctx.setFontSize(22)
  ctx.setFillStyle('#94a3b8')
  ctx.fillText(item.label || '--', x + 28, y + 42)

  ctx.setFontSize(34)
  ctx.setFillStyle('#f8fafc')
  ctx.fillText(item.value || '--', x + 28, y + 94)

  if (item.desc) {
    drawTextBlock(ctx, item.desc, x + 28, y + 132, 16, 28, 2, '#cbd5e1', 20)
  }
}

function drawSectionHeader(ctx, title, subtitle, y) {
  ctx.setFontSize(30)
  ctx.setFillStyle('#f8fafc')
  ctx.fillText(title, 60, y)
  if (subtitle) {
    ctx.setFontSize(20)
    ctx.setFillStyle('#94a3b8')
    ctx.fillText(subtitle, 60, y + 34)
  }
}

/**
 * 生成对局战绩海报
 * @param {string} canvasId Canvas ID
 * @param {Object} data 对局数据
 * @returns {Promise<string>} 海报图片临时路径
 */
export function generateMatchPoster(canvasId, data) {
  return new Promise((resolve, reject) => {
    const ctx = uni.createCanvasContext(canvasId)
    const W = 750
    const highlights = Array.isArray(data.summary_highlights) ? data.summary_highlights.slice(0, 4) : []
    const stats = Array.isArray(data.summary_stats) ? data.summary_stats.slice(0, 6) : []
    
    // 动态计算所需高度，防止超出固定高度被覆盖内容
    const hRows = highlights.length > 0 ? Math.ceil(highlights.length / 2) : 1
    const sRows = stats.length > 0 ? Math.ceil(stats.length / 2) : 0
    const statsTop = 572 + 56 + hRows * 170 + 20
    const statsEnd = statsTop + 56 + sRows * 170
    const H = Math.max(1320, statsEnd + 150)

    const gameTypeName = data.game_type_name || getGameTypeLabel(data.game_type)
    const rankChange = Number(data.rank_change || 0)
    const rankChangeLabel = rankChange > 0 ? `排位 +${rankChange}` : rankChange < 0 ? `排位 ${rankChange}` : '排位 ±0'
    const resultColor = data.result === '胜利' ? '#22c55e' : data.result === '失利' ? '#f97316' : '#cbd5e1'

    const grd = ctx.createLinearGradient(0, 0, W, H)
    grd.addColorStop(0, '#0f172a')
    grd.addColorStop(1, '#111827')
    ctx.setFillStyle(grd)
    ctx.fillRect(0, 0, W, H)

    ctx.setFillStyle('rgba(59,130,246,0.12)')
    ctx.beginPath()
    ctx.arc(640, 110, 180, 0, Math.PI * 2)
    ctx.fill()
    ctx.setFillStyle('rgba(16,185,129,0.10)')
    ctx.beginPath()
    ctx.arc(80, 220, 140, 0, Math.PI * 2)
    ctx.fill()

    drawCenterText(ctx, '追分 · 对局战报', 84, 30, '#cbd5e1')
    drawCenterText(ctx, gameTypeName, 126, 24, '#94a3b8')

    drawRoundRect(ctx, 40, 170, 670, 300, 32)
    ctx.setFillStyle('rgba(15,23,42,0.72)')
    ctx.fill()

    drawRoundRect(ctx, 60, 194, 132, 48, 24)
    ctx.setFillStyle('rgba(255,255,255,0.08)')
    ctx.fill()
    ctx.setFontSize(24)
    ctx.setFillStyle(resultColor)
    ctx.setTextAlign('center')
    ctx.fillText(data.result || '已完赛', 126, 228)

    drawRoundRect(ctx, 558, 194, 112, 48, 24)
    ctx.setFillStyle('rgba(59,130,246,0.14)')
    ctx.fill()
    ctx.setFillStyle('#93c5fd')
    ctx.fillText(rankChangeLabel, 614, 228)

    ctx.setFillStyle('#f8fafc')
    ctx.setFontSize(28)
    ctx.fillText(truncateText(data.my_name || '我', 7), 156, 306)
    ctx.fillText(truncateText(data.opponent_name || '对手', 7), 594, 306)

    ctx.setFontSize(86)
    ctx.setFillStyle('#f8fafc')
    ctx.fillText(String(data.my_score || 0), 172, 392)
    ctx.setFontSize(42)
    ctx.setFillStyle('#64748b')
    ctx.fillText(':', 375, 382)
    ctx.setFontSize(86)
    ctx.setFillStyle('#f8fafc')
    ctx.fillText(String(data.opponent_score || 0), 578, 392)
    ctx.setTextAlign('left')

    ctx.setFontSize(22)
    ctx.setFillStyle('#94a3b8')
    ctx.fillText('我的得分', 100, 438)
    ctx.fillText('对手得分', 514, 438)

    ctx.setStrokeStyle('rgba(255,255,255,0.08)')
    ctx.beginPath()
    ctx.moveTo(70, 438)
    ctx.lineTo(680, 438)
    ctx.stroke()

    ctx.setFontSize(22)
    ctx.setFillStyle('#cbd5e1')
    ctx.fillText(`对局时间 ${data.match_time || data.date || '--'}`, 80, 510)
    ctx.fillText(`总结口径与战报页一致`, 500, 510)

    drawSectionHeader(ctx, '本场亮点', `按${gameTypeName}模式提炼这场最值得记住的表现`, 572)
    if (highlights.length) {
      highlights.forEach((item, index) => {
        const col = index % 2
        const row = Math.floor(index / 2)
        drawSummaryCard(ctx, item, 60 + col * 315, 628 + row * 170, 275, 146)
      })
    } else {
      drawRoundRect(ctx, 60, 628, 630, 146, 24)
      ctx.setFillStyle('rgba(255,255,255,0.05)')
      ctx.fill()
      drawCenterText(ctx, '当前模式暂无可展示亮点，但战绩已完成记录', 710, 24, '#94a3b8')
    }

    // statsTop 已在顶部动态计算
    drawSectionHeader(ctx, '本场数据', '海报展示项与对局总结页保持同一统计口径', statsTop)
    stats.forEach((item, index) => {
      const col = index % 2
      const row = Math.floor(index / 2)
      drawSummaryCard(ctx, item, 60 + col * 315, statsTop + 56 + row * 170, 275, 146)
    })

    drawCenterText(ctx, '追分', H - 92, 30, '#60a5fa')
    drawCenterText(ctx, '真实对局数据生成，仅供复盘与分享', H - 54, 20, '#64748b')

    // 绘制完成
    ctx.draw(false, () => {
      setTimeout(() => {
        uni.canvasToTempFilePath({
          canvasId: canvasId,
          x: 0,
          y: 0,
          width: W,
          height: H,
          destWidth: W,
          destHeight: H,
          quality: 1,
          success: (res) => resolve(res.tempFilePath),
          fail: (err) => reject(err)
        })
      }, 300)
    })
  })
}

/**
 * 生成 PK 报表海报
 * @param {string} canvasId Canvas ID
 * @param {Object} data PK 报表数据
 * @returns {Promise<string>} 海报图片临时路径
 */
export function generatePkReportPoster(canvasId, data) {
  return new Promise((resolve, reject) => {
    const ctx = uni.createCanvasContext(canvasId)
    const W = 750
    const H = 1080

    const grd = ctx.createLinearGradient(0, 0, 0, H)
    grd.addColorStop(0, '#0f172a')
    grd.addColorStop(1, '#1e3a5f')
    ctx.setFillStyle(grd)
    ctx.fillRect(0, 0, W, H)

    drawCenterText(ctx, '追分 · PK 报表', 86, 28, '#94a3b8')
    drawCenterText(ctx, data.gameTypeLabel || '真实交锋数据', 132, 22, '#64748b')

    ctx.setTextAlign('center')
    ctx.setFontSize(28)
    ctx.setFillStyle('#ffffff')
    ctx.fillText(data.myName || '我', 190, 250)
    ctx.fillText(data.opponentName || '对手', 560, 250)

    ctx.setFontSize(80)
    ctx.setFillStyle('#22c55e')
    ctx.fillText(String(data.myWins || 0), 190, 380)
    ctx.setFontSize(42)
    ctx.setFillStyle('#64748b')
    ctx.fillText(':', 375, 368)
    ctx.setFontSize(80)
    ctx.setFillStyle('#f8fafc')
    ctx.fillText(String(data.opponentWins || 0), 560, 380)

    ctx.setFontSize(24)
    ctx.setFillStyle('#94a3b8')
    ctx.fillText('历史交锋胜场', 375, 430)
    ctx.setTextAlign('left')

    drawRoundRect(ctx, 60, 500, 630, 250, 24)
    ctx.setFillStyle('rgba(255,255,255,0.08)')
    ctx.fill()

    const metrics = [
      ['总场次', String(data.totalMatches || 0)],
      ['我的胜率', data.winRateLabel || '0.00%'],
      ['平均分差', data.avgScoreDiffLabel || '0.0'],
      ['最长连胜', String(data.maxWinStreak || 0)]
    ]

    metrics.forEach(([label, value], index) => {
      const col = index % 2
      const row = Math.floor(index / 2)
      const x = 110 + col * 280
      const y = 570 + row * 110
      ctx.setFontSize(22)
      ctx.setFillStyle('#94a3b8')
      ctx.fillText(label, x, y)
      ctx.setFontSize(36)
      ctx.setFillStyle('#ffffff')
      ctx.fillText(value, x, y + 48)
    })

    const summaryLayout = resolvePkSummaryLayout(data.summaryText)

    drawRoundRect(ctx, 60, summaryLayout.cardY, 630, summaryLayout.cardHeight, 24)
    ctx.setFillStyle('rgba(59,130,246,0.12)')
    ctx.fill()
    ctx.setFontSize(24)
    ctx.setFillStyle('#bfdbfe')
    ctx.fillText('系统结论', 100, summaryLayout.titleY)
    ctx.setFontSize(28)
    ctx.setFillStyle('#ffffff')
    summaryLayout.summaryLines.forEach((line) => {
      ctx.fillText(line.text, 100, line.y)
    })

    drawCenterText(ctx, '仅统计真实线下对局，不代表线上比赛结果', 1000, 22, '#94a3b8')

    ctx.draw(false, () => {
      setTimeout(() => {
        uni.canvasToTempFilePath({
          canvasId,
          quality: 1,
          success: (res) => resolve(res.tempFilePath),
          fail: (err) => reject(err)
        })
      }, 300)
    })
  })
}

/**
 * 保存海报到相册
 * @param {string} tempFilePath 临时文件路径
 */
export function savePosterToAlbum(tempFilePath) {
  return new Promise((resolve, reject) => {
    uni.saveImageToPhotosAlbum({
      filePath: tempFilePath,
      success: () => {
        uni.showToast({ title: '保存成功', icon: 'success' })
        resolve()
      },
      fail: (err) => {
        if (err.errMsg && err.errMsg.indexOf('auth deny') !== -1) {
          uni.showModal({
            title: '提示',
            content: '需要您授权保存图片到相册',
            success: (res) => {
              if (res.confirm) {
                uni.openSetting()
              }
            }
          })
        } else {
          uni.showToast({ title: '保存失败', icon: 'none' })
        }
        reject(err)
      }
    })
  })
}
