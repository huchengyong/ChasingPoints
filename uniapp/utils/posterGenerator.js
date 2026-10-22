/**
 * 海报生成工具
 * 使用 Canvas 绘制分享海报
 */

import { getGameTypeLabel } from './game-types.js'
import { resolveMemberIdentity } from './member-identity.js'
import { DEFAULT_USER_AVATAR } from './user-profile.js'

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

// 会员模板按分享者的外观档位配色（暖黑金，与 DESIGN.md「高价值表面」一致）。
const MEMBER_POSTER_THEMES = {
  standard: {
    background: ['#1E180D', '#141109'],
    card: 'rgba(30,24,13,0.78)',
    glowPrimary: 'rgba(224,174,18,0.14)',
    glowSecondary: 'rgba(224,174,18,0.08)',
    title: '#F0C542',
    subtitle: '#D7C89B',
    chipBackground: 'rgba(224,174,18,0.16)',
    chipText: '#F0C542',
    brand: '#F0C542',
    footnote: '#9F926E'
  },
  advanced: {
    background: ['#2E2209', '#141109'],
    card: 'rgba(36,26,8,0.78)',
    glowPrimary: 'rgba(240,197,66,0.20)',
    glowSecondary: 'rgba(224,174,18,0.10)',
    title: '#F5CF5A',
    subtitle: '#E2D3A5',
    chipBackground: 'rgba(240,197,66,0.20)',
    chipText: '#F5CF5A',
    brand: '#F5CF5A',
    footnote: '#A89A74'
  },
  premium: {
    background: ['#3C2B08', '#8F6507'],
    card: 'rgba(20,14,4,0.72)',
    glowPrimary: 'rgba(253,233,166,0.18)',
    glowSecondary: 'rgba(253,233,166,0.10)',
    title: '#FDE9A6',
    subtitle: '#F0DEB0',
    chipBackground: 'rgba(253,233,166,0.20)',
    chipText: '#FDE9A6',
    brand: '#FDE9A6',
    footnote: '#D9C48A'
  }
}

// VIP 标识在海报上的样式，对应客户端组件的三档（标准浅底、进阶金底、尊享暖黑金）。
const POSTER_BADGE_STYLES = {
  standard: { fill: 'rgba(224,174,18,0.22)', gradient: null, stroke: '#E0AE12', text: '#F0C542' },
  advanced: { fill: '#D9A617', gradient: ['#D9A617', '#BB8400'], stroke: 'rgba(187,132,0,0.6)', text: '#FFFFFF' },
  premium: { fill: '#3C2B08', gradient: ['#3C2B08', '#6B4A07'], stroke: '#C69200', text: '#F0C542' }
}

// 会员模板在标准模板的头部卡片里多出的高度，卡片下方所有内容整体下移。
const MEMBER_HEADER_EXTRA = 124
const POSTER_AVATAR_RADIUS = 44
const POSTER_AVATAR_CENTER_Y = 322
const POSTER_FRAME_RATIO = 156 / 96

/**
 * 战绩海报用哪套模板，由服务端在请求时刻解析的会员字段决定：
 * 分享者是有效会员 → 会员模板（档位配色、头像框、VIP 标识）；否则保持标准模板。
 * 对手是会员时两套模板都要在对手一侧展示 VIP 标识。字段缺失按非会员处理。
 */
export function resolveMatchPosterTemplate(data = {}) {
  const mine = resolveMemberIdentity({ vipLevel: data?.my_vip_level, rankLevel: data?.my_rank_level })
  const opponent = resolveMemberIdentity({ vipLevel: data?.opponent_vip_level })
  const member = mine.showBadge
  return {
    member,
    tier: member ? mine.tier : '',
    theme: member ? MEMBER_POSTER_THEMES[mine.tier] : null,
    headerExtra: member ? MEMBER_HEADER_EXTRA : 0,
    myBadge: member ? { text: mine.badgeText, tier: mine.tier } : null,
    frameUrl: member ? mine.frameUrl : '',
    opponentBadge: opponent.showBadge ? { text: opponent.badgeText, tier: opponent.tier } : null
  }
}

/**
 * 读取海报要画的图片；失败只返回空串，调用方跳过这一项，不让一张图拖垮整张海报。
 */
export function loadPosterImage(src) {
  if (!src) return Promise.resolve('')
  return new Promise((resolve) => {
    uni.getImageInfo({
      src,
      success: (res) => resolve(res?.path || ''),
      fail: () => resolve('')
    })
  })
}

function drawVipBadge(ctx, centerX, centerY, badge, scale = 1) {
  const style = POSTER_BADGE_STYLES[badge.tier] || POSTER_BADGE_STYLES.standard
  const width = Math.round(88 * scale)
  const height = Math.round(34 * scale)
  const x = centerX - width / 2
  const y = centerY - height / 2
  drawRoundRect(ctx, x, y, width, height, height / 2)
  if (style.gradient) {
    const grd = ctx.createLinearGradient(x, y, x + width, y + height)
    grd.addColorStop(0, style.gradient[0])
    grd.addColorStop(1, style.gradient[1])
    ctx.setFillStyle(grd)
  } else {
    ctx.setFillStyle(style.fill)
  }
  ctx.fill()
  drawRoundRect(ctx, x, y, width, height, height / 2)
  ctx.setStrokeStyle(style.stroke)
  ctx.stroke()
  ctx.setFontSize(Math.round(22 * scale))
  ctx.setFillStyle(style.text)
  ctx.setTextAlign('center')
  ctx.fillText(badge.text, centerX, y + Math.round(25 * scale))
  ctx.setTextAlign('left')
}

// 头像读不到时画名字首字的占位圆，海报仍然完整。
function drawAvatarOrPlaceholder(ctx, imagePath, name, centerX, centerY, radius) {
  if (imagePath) {
    drawCircleImage(ctx, imagePath, centerX - radius, centerY - radius, radius)
    return
  }
  ctx.beginPath()
  ctx.arc(centerX, centerY, radius, 0, Math.PI * 2)
  ctx.setFillStyle('rgba(255,255,255,0.14)')
  ctx.fill()
  ctx.setFontSize(radius)
  ctx.setFillStyle('#f8fafc')
  ctx.setTextAlign('center')
  ctx.fillText(String(name || '我').slice(0, 1), centerX, centerY + radius * 0.35)
  ctx.setTextAlign('left')
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
 * 分享者是有效会员时使用会员模板，否则使用标准模板；对手是会员时两套模板都展示其 VIP 标识。
 * @param {string} canvasId Canvas ID
 * @param {Object} data 对局数据
 * @returns {Promise<string>} 海报图片临时路径
 */
export async function generateMatchPoster(canvasId, data) {
  const template = resolveMatchPosterTemplate(data)
  // 图片读取失败只会缺这一项（框、头像用占位），不会让会员模板整体失败。
  const images = template.member
    ? {
      avatar: await loadPosterImage(data.my_avatar || DEFAULT_USER_AVATAR),
      frame: await loadPosterImage(template.frameUrl),
      opponentAvatar: await loadPosterImage(data.opponent_avatar || DEFAULT_USER_AVATAR)
    }
    : {}
  return drawMatchPoster(canvasId, data, template, images)
}

function drawMatchPoster(canvasId, data, template, images) {
  return new Promise((resolve, reject) => {
    const ctx = uni.createCanvasContext(canvasId)
    const W = 750
    const theme = template.theme
    const dy = template.headerExtra
    const highlights = Array.isArray(data.summary_highlights) ? data.summary_highlights.slice(0, 4) : []
    const stats = Array.isArray(data.summary_stats) ? data.summary_stats.slice(0, 6) : []
    
    // 动态计算所需高度，防止超出固定高度被覆盖内容
    const hRows = highlights.length > 0 ? Math.ceil(highlights.length / 2) : 1
    const sRows = stats.length > 0 ? Math.ceil(stats.length / 2) : 0
    const statsTop = 572 + dy + 56 + hRows * 170 + 20
    const statsEnd = statsTop + 56 + sRows * 170
    const H = Math.max(1320 + dy, statsEnd + 150)

    const gameTypeName = data.game_type_name || getGameTypeLabel(data.game_type)
    const rankChange = Number(data.rank_change || 0)
    const rankChangeLabel = rankChange > 0 ? `排位 +${rankChange}` : rankChange < 0 ? `排位 ${rankChange}` : '排位 ±0'
    const resultColor = data.result === '胜利' ? '#22c55e' : data.result === '失利' ? '#f97316' : '#cbd5e1'

    const grd = ctx.createLinearGradient(0, 0, W, H)
    grd.addColorStop(0, theme ? theme.background[0] : '#0f172a')
    grd.addColorStop(1, theme ? theme.background[1] : '#111827')
    ctx.setFillStyle(grd)
    ctx.fillRect(0, 0, W, H)

    ctx.setFillStyle(theme ? theme.glowPrimary : 'rgba(59,130,246,0.12)')
    ctx.beginPath()
    ctx.arc(640, 110, 180, 0, Math.PI * 2)
    ctx.fill()
    ctx.setFillStyle(theme ? theme.glowSecondary : 'rgba(16,185,129,0.10)')
    ctx.beginPath()
    ctx.arc(80, 220, 140, 0, Math.PI * 2)
    ctx.fill()

    drawCenterText(ctx, '追分 · 对局战报', 84, 30, theme ? theme.title : '#cbd5e1')
    drawCenterText(ctx, gameTypeName, 126, 24, theme ? theme.subtitle : '#94a3b8')

    drawRoundRect(ctx, 40, 170, 670, 300 + dy, 32)
    ctx.setFillStyle(theme ? theme.card : 'rgba(15,23,42,0.72)')
    ctx.fill()

    drawRoundRect(ctx, 60, 194, 132, 48, 24)
    ctx.setFillStyle('rgba(255,255,255,0.08)')
    ctx.fill()
    ctx.setFontSize(24)
    ctx.setFillStyle(resultColor)
    ctx.setTextAlign('center')
    ctx.fillText(data.result || '已完赛', 126, 228)

    drawRoundRect(ctx, 558, 194, 112, 48, 24)
    ctx.setFillStyle(theme ? theme.chipBackground : 'rgba(59,130,246,0.14)')
    ctx.fill()
    ctx.setFillStyle(theme ? theme.chipText : '#93c5fd')
    ctx.fillText(rankChangeLabel, 614, 228)

    if (template.member) {
      // 会员模板：分享者头像 + 段位头像框 + VIP 标识；对手是会员时同样标出 VIP 标识。
      drawAvatarOrPlaceholder(ctx, images.avatar, data.my_name, 156, POSTER_AVATAR_CENTER_Y, POSTER_AVATAR_RADIUS)
      if (images.frame) {
        const frameSize = Math.round(POSTER_AVATAR_RADIUS * 2 * POSTER_FRAME_RATIO)
        ctx.drawImage(images.frame, 156 - frameSize / 2, POSTER_AVATAR_CENTER_Y - frameSize / 2, frameSize, frameSize)
      }
      drawVipBadge(ctx, 156, POSTER_AVATAR_CENTER_Y + POSTER_AVATAR_RADIUS + 6, template.myBadge)
      drawAvatarOrPlaceholder(ctx, images.opponentAvatar, data.opponent_name, 594, POSTER_AVATAR_CENTER_Y, POSTER_AVATAR_RADIUS)
      if (template.opponentBadge) {
        drawVipBadge(ctx, 594, POSTER_AVATAR_CENTER_Y + POSTER_AVATAR_RADIUS + 6, template.opponentBadge)
      }
      ctx.setTextAlign('center')
    } else if (template.opponentBadge) {
      // 标准模板：只在对手名字上方标出 VIP 标识（缩小一档，夹在排位标签与名字之间）。
      drawVipBadge(ctx, 594, 264, template.opponentBadge, 0.85)
      ctx.setTextAlign('center')
    }

    ctx.setFillStyle('#f8fafc')
    ctx.setFontSize(28)
    ctx.fillText(truncateText(data.my_name || '我', 7), 156, 306 + dy)
    ctx.fillText(truncateText(data.opponent_name || '对手', 7), 594, 306 + dy)

    ctx.setFontSize(86)
    ctx.setFillStyle('#f8fafc')
    ctx.fillText(String(data.my_score || 0), 172, 392 + dy)
    ctx.setFontSize(42)
    ctx.setFillStyle('#64748b')
    ctx.fillText(':', 375, 382 + dy)
    ctx.setFontSize(86)
    ctx.setFillStyle('#f8fafc')
    ctx.fillText(String(data.opponent_score || 0), 578, 392 + dy)
    ctx.setTextAlign('left')

    ctx.setFontSize(22)
    ctx.setFillStyle('#94a3b8')
    ctx.fillText('我的得分', 100, 438 + dy)
    ctx.fillText('对手得分', 514, 438 + dy)

    ctx.setStrokeStyle('rgba(255,255,255,0.08)')
    ctx.beginPath()
    ctx.moveTo(70, 438 + dy)
    ctx.lineTo(680, 438 + dy)
    ctx.stroke()

    ctx.setFontSize(22)
    ctx.setFillStyle('#cbd5e1')
    ctx.fillText(`对局时间 ${data.match_time || data.date || '--'}`, 80, 510 + dy)
    ctx.fillText(`总结口径与战报页一致`, 500, 510 + dy)

    drawSectionHeader(ctx, '本场亮点', `按${gameTypeName}模式提炼这场最值得记住的表现`, 572 + dy)
    if (highlights.length) {
      highlights.forEach((item, index) => {
        const col = index % 2
        const row = Math.floor(index / 2)
        drawSummaryCard(ctx, item, 60 + col * 315, 628 + dy + row * 170, 275, 146)
      })
    } else {
      drawRoundRect(ctx, 60, 628 + dy, 630, 146, 24)
      ctx.setFillStyle('rgba(255,255,255,0.05)')
      ctx.fill()
      drawCenterText(ctx, '当前模式暂无可展示亮点，但战绩已完成记录', 710 + dy, 24, '#94a3b8')
    }

    // statsTop 已在顶部动态计算
    drawSectionHeader(ctx, '本场数据', '海报展示项与对局总结页保持同一统计口径', statsTop)
    stats.forEach((item, index) => {
      const col = index % 2
      const row = Math.floor(index / 2)
      drawSummaryCard(ctx, item, 60 + col * 315, statsTop + 56 + row * 170, 275, 146)
    })

    drawCenterText(ctx, '追分', H - 92, 30, theme ? theme.brand : '#60a5fa')
    drawCenterText(ctx, '真实对局数据生成，仅供复盘与分享', H - 54, 20, theme ? theme.footnote : '#64748b')

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
