// 统一扫码流程的运行时接线：绑定真实的用户会话与预览接口。核心逻辑和测试见 quick-scan.js。
import { previewMatchInvite, resolveMatchInviteCode } from '@/api/match.js'
import { useUserStore } from '@/store/user.js'
import { createQuickScan } from '@/utils/quick-scan.js'

const quickScan = createQuickScan({
  uniApi: uni,
  previewMatchInvite,
  resolveMatchInviteCode,
  getSession: () => {
    const userStore = useUserStore()
    return {
      isLoggedIn: userStore.isLoggedIn,
      userId: userStore.userId,
      authGeneration: userStore.authGeneration
    }
  }
})

export const startQuickScan = (options) => quickScan.start(options)

export const takeScanHandoff = (identity) => quickScan.takeHandoff(identity)

export const handoffResolvedScan = (payload) => quickScan.handoffResolved(payload)
