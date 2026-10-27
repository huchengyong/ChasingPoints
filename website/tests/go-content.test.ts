import { describe, expect, it } from 'vitest'
import { entryCopy, resolveEntryVariant } from '../data/go'

describe('/go/ fallback page', () => {
  it('picks the variant from the entry link query', () => {
    expect(resolveEntryVariant({ j: 'Ab3dEf6hIj9k' })).toBe('invite')
    expect(resolveEntryVariant({ m: '123' })).toBe('match')
    expect(resolveEntryVariant({})).toBe('home')
    expect(resolveEntryVariant({ j: '' })).toBe('home')
  })

  it('sends invite visitors back to the original match code instead of the home mini program code', () => {
    // 首页小程序码会打开小程序首页，邀请就丢了。
    expect(entryCopy.invite.showMiniProgramCode).toBe(false)
    expect(entryCopy.invite.hint).toContain('微信')
    expect(entryCopy.invite.hint).toContain('重新扫描对方手机上的匹配码')
    expect(`${entryCopy.invite.description}${entryCopy.invite.note}`).not.toContain('不用再扫码')
  })

  it('tells match visitors to rescan the original code', () => {
    expect(entryCopy.match.hint).toContain('重新扫描')
    expect(entryCopy.match.note).toContain('首页')
  })
})
