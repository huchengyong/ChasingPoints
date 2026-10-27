import { describe, expect, it } from 'vitest'
import { downloadEntries } from '../data/download'

describe('downloadEntries', () => {
  it('offers the WeChat mini program with its scan code', () => {
    const miniProgram = downloadEntries.find((entry) => entry.platform === 'wechat-miniprogram')

    expect(miniProgram?.href).toBe('/go/')
    expect(miniProgram?.qrImage).toBe('/qr-miniprogram.jpg')
    expect(miniProgram?.qrTargetUrl).toBe('/go/')
  })

  it('leaves Harmony OS without a download link until it is published', () => {
    const harmony = downloadEntries.find((entry) => entry.platform === 'harmony')

    expect(harmony?.href).toBeUndefined()
    expect(harmony?.qrImage).toBeUndefined()
    expect(harmony?.note).toContain('上架')
  })

  it('no longer offers iOS or Android entries', () => {
    expect(downloadEntries.map((entry) => entry.platform)).toEqual(['wechat-miniprogram', 'harmony'])
  })
})
