import { resolvePublicValue } from './runtime'

export const contactInfo = {
  title: '联系我们与投诉举报',
  description: '如果你想反馈问题、提交投诉举报、洽谈合作或了解更多信息，可以通过以下方式联系追分团队。',
  email: resolvePublicValue('NUXT_PUBLIC_CONTACT_EMAIL', 'service@dianzaozao.com'),
  phone: resolvePublicValue('NUXT_PUBLIC_CONTACT_PHONE', '021-50583611'),
  feedbackApiBaseUrl: resolvePublicValue('NUXT_PUBLIC_API_BASE_URL', 'https://api-bm.dianzaozao.com'),
  businessNote: '商务合作、内容联动与品牌沟通可优先通过邮箱联系。',
  complaintNote: '投诉举报请尽量写清对象、时间、联系方式和证据线索。'
}
