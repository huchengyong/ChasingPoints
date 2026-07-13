import { describe, expect, it } from 'vitest'
import { addClientAppManifestFallback } from '../config/vite-client-alias'

describe('addClientAppManifestFallback', () => {
  it('adds the Nuxt app manifest fallback to object aliases', () => {
    const config = {
      resolve: {
        alias: {
          '#imports': '/tmp/imports'
        }
      }
    }

    addClientAppManifestFallback(config)

    expect(config.resolve.alias['#app-manifest']).toContain('mocked-exports')
    expect(config.resolve.alias['#imports']).toBe('/tmp/imports')
  })

  it('keeps an existing app manifest alias', () => {
    const config = {
      resolve: {
        alias: {
          '#app-manifest': '/tmp/custom-manifest'
        }
      }
    }

    addClientAppManifestFallback(config)

    expect(config.resolve.alias['#app-manifest']).toBe('/tmp/custom-manifest')
  })

  it('prepends the fallback to array aliases when missing', () => {
    const config = {
      resolve: {
        alias: [
          {
            find: '#imports',
            replacement: '/tmp/imports'
          }
        ]
      }
    }

    addClientAppManifestFallback(config)

    expect(config.resolve.alias[0]).toMatchObject({
      find: '#app-manifest'
    })
  })
})
