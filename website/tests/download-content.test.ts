import { describe, expect, it } from 'vitest'
import { downloadEntries } from '../data/download'

describe('downloadEntries', () => {
  it('points both direct download actions to the coming-soon page', () => {
    expect(downloadEntries.map((entry) => entry.href)).toEqual([
      'https://www.zhuifen.cn/coming-soon',
      'https://www.zhuifen.cn/coming-soon'
    ])
  })

  it('points both QR targets to the coming-soon page', () => {
    expect(downloadEntries.map((entry) => entry.qrTargetUrl)).toEqual([
      'https://www.zhuifen.cn/coming-soon',
      'https://www.zhuifen.cn/coming-soon'
    ])
  })
})
