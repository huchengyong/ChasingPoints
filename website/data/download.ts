import { resolvePublicValue } from './runtime'

export interface DownloadEntry {
  platform: 'ios' | 'android'
  label: string
  href: string
  qrImage: string
  qrTargetUrl: string
  note: string
}

const comingSoonQrTarget = 'https://www.zhuifen.cn/coming-soon'

export const downloadEntries: DownloadEntry[] = [
  {
    platform: 'ios',
    label: 'iPhone 下载',
    href: comingSoonQrTarget,
    qrImage: '/qr-ios.svg',
    qrTargetUrl: comingSoonQrTarget,
    note: '请使用 iPhone 打开下载入口，或在桌面端扫码查看当前平台开放状态。'
  },
  {
    platform: 'android',
    label: 'Android 下载',
    href: comingSoonQrTarget,
    qrImage: '/qr-android.svg',
    qrTargetUrl: comingSoonQrTarget,
    note: '请使用 Android 手机打开下载入口，或在桌面端扫码查看当前平台开放状态。'
  }
]

export const downloadPageContent = {
  title: '下载追分',
  description: '选择你的设备，获取 iOS 与 Android 的正式下载入口。',
  notes: [
    '追分面向台球爱好者，帮助你记录对局、生成战报并持续积累战绩。',
    '如果你在电脑上访问官网，可以直接扫码下载到手机。',
    '首期下载页只承接官方下载，不混入其他复杂功能流程。'
  ]
}
