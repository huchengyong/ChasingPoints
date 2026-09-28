export const THEME_MODE_SYSTEM = 'system'
export const THEME_MODE_LIGHT = 'light'
export const THEME_MODE_DARK = 'dark'

export const THEME_MODE_STORAGE_KEY = 'theme_mode'
export const LEGACY_THEME_STORAGE_KEY = 'user_theme_dark'

const THEME_MODES = [THEME_MODE_SYSTEM, THEME_MODE_LIGHT, THEME_MODE_DARK]

export const normalizeThemeMode = (mode) => (
  THEME_MODES.includes(mode) ? mode : THEME_MODE_SYSTEM
)

export const readStoredThemeMode = (storage) => {
  const storedMode = storage.getStorageSync(THEME_MODE_STORAGE_KEY)
  if (THEME_MODES.includes(storedMode)) {
    return storedMode
  }

  const legacyPreference = storage.getStorageSync(LEGACY_THEME_STORAGE_KEY)
  if (typeof legacyPreference === 'boolean') {
    return legacyPreference ? THEME_MODE_DARK : THEME_MODE_LIGHT
  }

  return THEME_MODE_SYSTEM
}

export const resolveSystemDarkMode = (themeInfo = {}, fallback = false) => {
  const value = [themeInfo.hostTheme, themeInfo.theme, themeInfo.osTheme]
    .find((theme) => theme === THEME_MODE_LIGHT || theme === THEME_MODE_DARK)

  return value ? value === THEME_MODE_DARK : fallback
}

export const resolveDarkMode = (themeMode, systemIsDark) => {
  const normalizedMode = normalizeThemeMode(themeMode)
  if (normalizedMode === THEME_MODE_DARK) return true
  if (normalizedMode === THEME_MODE_LIGHT) return false
  return Boolean(systemIsDark)
}
