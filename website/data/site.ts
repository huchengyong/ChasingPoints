import { resolvePublicValue } from './runtime'

export interface SiteNavLink {
  label: string
  to: string
}

export interface SiteConfig {
  brandName: string
  defaultTitle: string
  titleTemplate: string
  siteUrl: string
  icpRecordNumber: string
  icpRecordUrl: string
  policeRecordNumber: string
  policeRecordUrl: string
  policeRecordIconUrl: string
  navLinks: SiteNavLink[]
  legalLinks: SiteNavLink[]
}

const legalLinks: SiteNavLink[] = [
  { label: '首页', to: '/' },
  { label: '下载', to: '/download' },
  { label: '隐私政策', to: '/privacy' },
  { label: '用户协议', to: '/agreement' },
  { label: '联系我们', to: '/contact' }
]

export const siteConfig: SiteConfig = {
  brandName: '追分',
  defaultTitle: '追分官网',
  titleTemplate: '%s | 追分官网',
  siteUrl: resolvePublicValue('NUXT_PUBLIC_SITE_URL', 'https://www.zhuifen.cn'),
  icpRecordNumber: resolvePublicValue('NUXT_PUBLIC_ICP_RECORD_NUMBER', '沪ICP备2021037913号-11'),
  icpRecordUrl: 'https://beian.miit.gov.cn/',
  policeRecordNumber: '沪公网安备31011502406701号',
  policeRecordUrl: 'https://beian.mps.gov.cn/#/query/webSearch?code=31011502406701',
  policeRecordIconUrl: 'https://beian.mps.gov.cn/web/assets/logo01.6189a29f.png',
  navLinks: legalLinks,
  legalLinks
}
