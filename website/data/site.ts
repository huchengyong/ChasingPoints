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
  siteUrl: resolvePublicValue('NUXT_PUBLIC_SITE_URL', 'https://example.com'),
  navLinks: legalLinks,
  legalLinks
}
