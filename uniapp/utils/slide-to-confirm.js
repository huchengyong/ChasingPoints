/**
 * 滑动结束纯逻辑：从左端滑块拖到终点并松手才算完成；点击轨道、中途松手、取消触摸均复位。
 */

export const SLIDE_COMPLETE_RATIO = 0.98

/**
 * @param {number} deltaX 距离起点拖动的像素（>=0）
 * @param {number} maxTrack 滑块可拖动的最大距离（轨道宽 - 滑块宽）
 */
export const slideProgress = (deltaX, maxTrack) => {
  if (!Number.isFinite(deltaX) || !Number.isFinite(maxTrack) || maxTrack <= 0) return 0
  const clamped = Math.min(Math.max(deltaX, 0), maxTrack)
  return clamped / maxTrack
}

/** 是否已滑到终点（允许 1-2px 误差）。 */
export const isSlideComplete = (deltaX, maxTrack) => slideProgress(deltaX, maxTrack) >= SLIDE_COMPLETE_RATIO

/**
 * 松手时是否提交：只有到达终点才完成。
 */
export const shouldCommitOnRelease = (deltaX, maxTrack) => isSlideComplete(deltaX, maxTrack)

/** 键盘完成（End 键或右方向键到达终点）。 */
export const isSlideFinishKey = (key) => key === 'End'
/** 键盘取消（Escape / Home 回起点）。 */
export const isSlideResetKey = (key) => key === 'Escape' || key === 'Home'
