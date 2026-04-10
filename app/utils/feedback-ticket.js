export const FEEDBACK_CATEGORY_OPTIONS = [
  {
    label: '意见反馈',
    value: 'feedback'
  },
  {
    label: '投诉',
    value: 'complaint'
  },
  {
    label: '举报',
    value: 'report'
  }
]

const FEEDBACK_CATEGORY_VALUES = new Set(FEEDBACK_CATEGORY_OPTIONS.map((item) => item.value))

export const normalizeFeedbackCategory = (category = '') => {
  const value = String(category).trim()
  return FEEDBACK_CATEGORY_VALUES.has(value) ? value : 'feedback'
}

export const buildFeedbackTicketPayload = ({ category = 'feedback', content = '', contact = '' } = {}) => {
  return {
    source: 'app',
    category: normalizeFeedbackCategory(category),
    content: String(content).trim(),
    contact: String(contact).trim()
  }
}
