export interface DownloadEntry {
  platform: 'ios' | 'android'
  label: string
  href: string
  qrImage: string
  note: string
}

export const downloadEntries: DownloadEntry[] = [
  {
    platform: 'ios',
    label: 'iPhone 下载',
    href: 'https://example.com/download/ios',
    qrImage: '/qr-ios.png',
    note: '请使用 iPhone 打开，或在桌面端扫码下载。'
  },
  {
    platform: 'android',
    label: 'Android 下载',
    href: 'https://example.com/download/android',
    qrImage: '/qr-android.png',
    note: '请使用 Android 手机打开，或在桌面端扫码下载。'
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
