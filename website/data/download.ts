export interface DownloadEntry {
  platform: 'wechat-miniprogram' | 'harmony'
  label: string
  heading: string
  note: string
  href?: string
  qrImage?: string
  qrTargetUrl?: string
}

// 小程序码图片由用户从微信公众平台下载后放在 public/，缺图时页面只保留文字入口。
const miniProgramEntryPoint = '/go/'

export const downloadEntries: DownloadEntry[] = [
  {
    platform: 'wechat-miniprogram',
    label: '微信小程序',
    heading: '微信扫一扫直接用',
    note: '无需下载安装。用微信扫小程序码，或长按识别进入追分。',
    href: miniProgramEntryPoint,
    qrImage: '/qr-miniprogram.jpg',
    qrTargetUrl: miniProgramEntryPoint
  },
  {
    platform: 'harmony',
    label: '鸿蒙',
    heading: 'HarmonyOS 手机',
    note: '鸿蒙版正在准备上架，开放后这里会换成正式下载入口。'
  }
]

export const downloadPageContent = {
  title: '获取追分',
  description: '微信小程序打开即用，鸿蒙版即将上线。',
  notes: [
    '追分面向台球爱好者，帮助你记录对局、生成战报并持续积累战绩。',
    '小程序扫码后即可开局，不需要额外安装。',
    '鸿蒙版上架前，这里只开放小程序入口。'
  ]
}
