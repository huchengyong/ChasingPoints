import { resolvePublicValue } from './runtime'

export const contactInfo = {
  title: '联系我们',
  description: '如果你想反馈问题、洽谈合作或了解更多信息，可以通过以下方式联系追分团队。',
  email: resolvePublicValue('NUXT_PUBLIC_CONTACT_EMAIL', 'hello@example.com'),
  wechat: resolvePublicValue('NUXT_PUBLIC_CONTACT_WECHAT', 'chasingpoints'),
  businessNote: '商务合作、内容联动与品牌沟通可优先通过邮箱联系。'
}
