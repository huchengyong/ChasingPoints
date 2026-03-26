import { siteConfig } from '../data/site'

export interface BuildPageSeoInput {
  title: string
  description: string
  path: string
  image?: string
}

export interface BuildPageSeoResult {
  title: string
  description: string
  canonicalUrl: string
  ogTitle: string
  ogDescription: string
  ogUrl: string
  ogImage: string
}

export function buildPageSeo({
  title,
  description,
  path,
  image = '/og-cover.jpg'
}: BuildPageSeoInput): BuildPageSeoResult {
  const canonicalUrl = `${siteConfig.siteUrl}${path}`
  const ogImage = image.startsWith('http')
    ? image
    : `${siteConfig.siteUrl}${image}`

  return {
    title: `${title} | ${siteConfig.brandName}官网`,
    description,
    canonicalUrl,
    ogTitle: `${title} | ${siteConfig.brandName}官网`,
    ogDescription: description,
    ogUrl: canonicalUrl,
    ogImage
  }
}

export function usePageSeo(input: BuildPageSeoInput) {
  const seo = buildPageSeo(input)

  useSeoMeta({
    title: seo.title,
    description: seo.description,
    ogTitle: seo.ogTitle,
    ogDescription: seo.ogDescription,
    ogUrl: seo.ogUrl,
    ogImage: seo.ogImage,
    twitterCard: 'summary_large_image',
    twitterTitle: seo.ogTitle,
    twitterDescription: seo.ogDescription,
    twitterImage: seo.ogImage
  })

  useHead({
    link: [
      {
        rel: 'canonical',
        href: seo.canonicalUrl
      }
    ]
  })

  return seo
}
