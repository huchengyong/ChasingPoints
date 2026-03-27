const formatDuration = (durationSeconds = 0) => {
  if (!durationSeconds || durationSeconds < 0) return '00:00'

  const minutes = Math.floor(durationSeconds / 60)
  const seconds = durationSeconds % 60
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

const getMatchScore = (match = {}) => {
  const leftScore = match.my_score ?? match.player1_score ?? 0
  const rightScore = match.opponent_score ?? match.player2_score ?? 0
  return `${leftScore} : ${rightScore}`
}

export const resolveGuestHeroCopy = () => ({
  eyebrow: '个人竞技主页',
  title: '登录后，解锁你的个人竞技主页',
  description: '记录比分、查看段位变化、沉淀每一场对局，挑战和消息也会集中在这里。',
  primaryActionText: '登录/注册',
  secondaryActionText: '发起PK'
})

export const resolveSectionTitles = () => ({
  stats: '竞技概览',
  quickActions: '竞技工具',
  secondaryServices: '更多竞技服务',
  settings: '设置与支持'
})

export const resolveUserHomepageMode = ({
  isLoggedIn,
  hasCurrentMatch,
  hasRecentMatch
}) => {
  if (!isLoggedIn) return 'guest'
  if (hasCurrentMatch) return 'ongoing'
  if (hasRecentMatch) return 'active'
  return 'idle'
}

export const resolvePrimaryAction = (mode) => {
  if (mode === 'ongoing') return 'continue'
  if (mode === 'guest') return 'login'
  return 'start'
}

export const resolveStatusCardContent = ({ mode, currentMatch = null, recentMatch = null } = {}) => {
  if (mode === 'ongoing' && currentMatch) {
    return {
      eyebrow: '当前对局',
      title: '继续这场比赛',
      description: `${currentMatch.opponent_name || '对手'} · ${getMatchScore(currentMatch)} · ${formatDuration(currentMatch.duration_seconds)}`,
      action: 'continue',
      actionText: '继续对局',
      secondaryAction: 'start',
      secondaryActionText: '发起 PK'
    }
  }

  if (mode === 'active' && recentMatch) {
    return {
      eyebrow: '最近一场',
      title: recentMatch.title || '刚完成一场比赛',
      description: recentMatch.description || '查看刚结束的对局结果，继续保持状态。',
      action: 'recent',
      actionText: '查看最近战绩',
      secondaryAction: 'start',
      secondaryActionText: '再来一场'
    }
  }

  if (mode === 'guest') {
    return {
      eyebrow: '登录后解锁',
      title: '登录后，解锁你的个人竞技主页',
      description: '登录后查看进行中的对局、个人战绩、段位变化和待处理事项。',
      action: 'login',
      actionText: '登录/注册',
      secondaryAction: 'start_pk',
      secondaryActionText: '发起PK'
    }
  }

  return {
    eyebrow: '准备开杆',
    title: '今天还没开始比赛',
    description: '这周还没开杆，来打一场找回节奏，继续记录你的竞技状态。',
    action: 'start',
    actionText: '发起 PK',
    secondaryAction: 'history',
    secondaryActionText: '查看比赛记录'
  }
}
