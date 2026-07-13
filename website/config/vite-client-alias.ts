import { createRequire } from 'node:module'
import type { Alias, AliasOptions, UserConfig } from 'vite'

const require = createRequire(import.meta.url)
const APP_MANIFEST_ALIAS = '#app-manifest'
const appManifestFallback = require.resolve('mocked-exports/empty')

type ViteConfigWithResolve = Pick<UserConfig, 'resolve'>

function hasAppManifestAlias(alias: AliasOptions | undefined): boolean {
  if (!alias) {
    return false
  }

  if (Array.isArray(alias)) {
    return alias.some((entry: Alias) => entry.find === APP_MANIFEST_ALIAS)
  }

  return Object.hasOwn(alias, APP_MANIFEST_ALIAS)
}

export function addClientAppManifestFallback(config: ViteConfigWithResolve): void {
  config.resolve ||= {}

  const alias = config.resolve.alias
  if (Array.isArray(alias)) {
    if (!hasAppManifestAlias(alias)) {
      alias.unshift({
        find: APP_MANIFEST_ALIAS,
        replacement: appManifestFallback
      })
    }
    return
  }

  config.resolve.alias = {
    [APP_MANIFEST_ALIAS]: appManifestFallback,
    ...(alias || {})
  }
}
