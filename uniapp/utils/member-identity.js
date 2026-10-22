import { resolveCdnAssetUrl } from './cdn-asset.js'

// 会员身份的展示规则：VIP 成长等级决定标识档位，该球种的当前段位决定头像框。
// 身份只是外观，与排位分、对局权限无关。
export const MAX_VIP_LEVEL = 5

export const MEMBER_IDENTITY_TIERS = [
  { tier: 'standard', label: '标准', minLevel: 1, maxLevel: 2, levelText: 'Lv1–Lv2' },
  { tier: 'advanced', label: '进阶', minLevel: 3, maxLevel: 4, levelText: 'Lv3–Lv4' },
  { tier: 'premium', label: '尊享', minLevel: 5, maxLevel: 5, levelText: 'Lv5' }
]

const RANK_AVATAR_FRAME_PATHS = {
  1: '/static/images/avatar-frames/frame_bronze.png',
  2: '/static/images/avatar-frames/frame_silver.png',
  3: '/static/images/avatar-frames/frame_gold.png',
  4: '/static/images/avatar-frames/frame_platinum.png',
  5: '/static/images/avatar-frames/frame_diamond.png',
  6: '/static/images/avatar-frames/frame_king.png'
}

const toSafeInteger = (value) => {
  const parsed = Math.trunc(Number(value))
  return Number.isFinite(parsed) ? parsed : 0
}

export const resolveRankAvatarFramePath = (level) => {
  return resolveCdnAssetUrl(RANK_AVATAR_FRAME_PATHS[toSafeInteger(level)] || '')
}

// 接口只返回有效会员的 1–5；缺失、0、非法值都按非会员处理，超出上限按最高档。
export const normalizeVipLevel = (vipLevel) => {
  const level = toSafeInteger(vipLevel)
  return level < 1 ? 0 : Math.min(level, MAX_VIP_LEVEL)
}

export const resolveMemberTier = (vipLevel) => {
  const level = normalizeVipLevel(vipLevel)
  return MEMBER_IDENTITY_TIERS.find(item => level >= item.minLevel && level <= item.maxLevel) || null
}

const NO_IDENTITY = Object.freeze({
  showBadge: false,
  vipLevel: 0,
  tier: '',
  tierLabel: '',
  badgeText: '',
  frameUrl: ''
})

/**
 * @param {{ vipLevel?: number, rankLevel?: number }} input
 * vipLevel 为 0 或缺失时没有任何会员外观；没有段位时只有标识、没有头像框。
 */
export const resolveMemberIdentity = ({ vipLevel = 0, rankLevel = 0 } = {}) => {
  const tier = resolveMemberTier(vipLevel)
  if (!tier) return NO_IDENTITY

  const level = normalizeVipLevel(vipLevel)
  return {
    showBadge: true,
    vipLevel: level,
    tier: tier.tier,
    tierLabel: tier.label,
    badgeText: `VIP${level}`,
    frameUrl: resolveRankAvatarFramePath(rankLevel)
  }
}

// 头像框素材（256px 方图）的翼饰横向几乎铺满整图，本人页用 96rpx 头像配 156rpx 框、124rpx 布局盒。
const FRAME_SIZE_RATIO = 156 / 96
const FRAME_BOX_RATIO = 124 / 96
const MIN_BADGE_FONT_RPX = 14
const MAX_BADGE_FONT_RPX = 22

/**
 * 头像尺寸（rpx）换算成头像框、VIP 标识与布局盒的尺寸。
 * - 默认：布局盒就是头像本身，框图向外溢出（用于四周有空白的位置）。
 * - reserveFrame：布局盒按头像框预留，会员与非会员占位一致（用于列表行）。
 * - fitFrame：size 是含头像框的总占位。有头像框时头像缩小到框内，框图不再向外溢出（用于领奖台这类空间固定的位置）。
 */
export const resolveMemberAvatarMetrics = (size, { reserveFrame = false, fitFrame = false, hasFrame = false } = {}) => {
  const requested = Math.max(0, toSafeInteger(size))
  const fitted = fitFrame && hasFrame
  const avatarSize = fitted ? Math.round(requested / FRAME_SIZE_RATIO) : requested
  const badgeFontSize = Math.min(MAX_BADGE_FONT_RPX, Math.max(MIN_BADGE_FONT_RPX, Math.round(avatarSize * 0.22)))
  return {
    avatarSize,
    boxSize: fitted ? requested : (reserveFrame ? Math.round(avatarSize * FRAME_BOX_RATIO) : avatarSize),
    frameSize: fitted ? requested : Math.round(avatarSize * FRAME_SIZE_RATIO),
    badgeFontSize,
    badgeHeight: Math.round(badgeFontSize * 1.5),
    badgePaddingX: Math.round(badgeFontSize * 0.45)
  }
}
