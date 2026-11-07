export const SOCIAL_TABS = [
  { label: '推荐', value: 'recommend' },
  { label: '好友', value: 'friends' },
  { label: '战报', value: 'reports' }
]

export const resolveFeedTab = (tab) => {
  if (tab === 'friends') return 'following'
  if (tab === 'reports') return 'reports'
  return 'public'
}

export const buildFeedUrl = (tab) => {
  const safeTab = SOCIAL_TABS.some((item) => item.value === tab) ? tab : 'recommend'
  return `/subPages/social/feed?tab=${safeTab}`
}

export const resolveSocialEmptyState = ({ tab, isLoggedIn }) => {
  if (tab === 'friends') {
    return {
      icon: '👥',
      title: isLoggedIn ? '好友还没有新动态' : '登录后查看好友动态',
      desc: isLoggedIn ? '去发一条近况，带动球友互动。' : '登录后可查看关注与好友的真实战绩分享。',
      ctaText: isLoggedIn ? '' : '立即登录'
    }
  }

  if (tab === 'reports') {
    return {
      icon: '🏆',
      title: '暂时还没有战报',
      desc: '真实对局结束后分享的战绩，会优先出现在这里。',
      ctaText: ''
    }
  }

  return {
    icon: '📝',
    title: '暂时还没有推荐内容',
    desc: '去发布第一条动态，或者稍后再来看看。',
    ctaText: ''
  }
}

export const filterReportPosts = (list = []) => {
  return list.filter((item) => item?.post_type === 1)
}
