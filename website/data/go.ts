// /go/ 兜底页：用系统相机或其他 App 扫了入口二维码的访问者。
// 匹配码和对局码都要回到微信里重扫原来的码才能保留邀请/对局，不能展示首页小程序码代替。
export type EntryVariant = 'home' | 'invite' | 'match'

export interface EntryCopy {
  eyebrow: string
  title: string
  description: string
  hint: string
  note: string
  showMiniProgramCode: boolean
}

export const entryCopy: Record<EntryVariant, EntryCopy> = {
  home: {
    eyebrow: '微信入口',
    title: '在微信里打开追分',
    description: '记录对局、生成战报，把每一场球留下来。',
    hint: '打开微信扫一扫，或长按识别小程序码',
    note: '小程序打开即用，不需要额外安装。鸿蒙版上架前，先用小程序记录你的对局。',
    showMiniProgramCode: true
  },
  invite: {
    eyebrow: '球友邀请',
    title: '球友邀请你打一局',
    description: '这是追分的匹配码，需要用微信扫码才能接受邀请。',
    hint: '请打开微信「扫一扫」，重新扫描对方手机上的匹配码',
    note: '扫码后登录即可直接进入对局确认。已安装鸿蒙版追分的，也可以在 App 内扫码。匹配码几分钟内有效，过期请让对方刷新。',
    showMiniProgramCode: false
  },
  match: {
    eyebrow: '对局分享',
    title: '这场对局值得一看',
    description: '用微信扫码，就能查看这场比赛的比分与局记录。',
    hint: '请打开微信「扫一扫」，重新扫描刚才的二维码',
    note: '直接扫原来的二维码会进入这场比赛；下面的小程序码只能打开追分首页。',
    showMiniProgramCode: true
  }
}

export const resolveEntryVariant = (query: Record<string, unknown>): EntryVariant => {
  const { j, m } = query
  if (typeof j === 'string' && j) return 'invite'
  if (typeof m === 'string' && m) return 'match'
  return 'home'
}
