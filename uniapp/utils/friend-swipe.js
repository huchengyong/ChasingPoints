export const clampFriendSwipeOffset = (offset = 0, actionWidth = 0) => {
  const maxSwipe = Math.max(Number(actionWidth) || 0, 0)
  const parsedOffset = Number(offset) || 0

  if (maxSwipe <= 0) {
    return 0
  }

  if (parsedOffset > 0) {
    return 0
  }

  if (parsedOffset < -maxSwipe) {
    return -maxSwipe
  }

  return parsedOffset
}

export const resolveFriendSwipeEndOffset = (offset = 0, actionWidth = 0, openRatio = 0.5) => {
  const maxSwipe = Math.max(Number(actionWidth) || 0, 0)
  const clampedOffset = clampFriendSwipeOffset(offset, maxSwipe)

  if (maxSwipe <= 0) {
    return 0
  }

  return Math.abs(clampedOffset) >= maxSwipe * openRatio ? -maxSwipe : 0
}
