const CAMERA_SCOPE = 'scope.camera'
const LOCATION_SCOPE = 'scope.userLocation'

const hasPermissionDenied = (error) => {
  const message = String(error?.errMsg || error?.message || '')
  return /denied|deny|refuse|forbidden|auth/i.test(message)
}

export const classifyScanFailure = (error) => {
  const message = String(error?.errMsg || error?.message || '').toLowerCase()
  if (/cancel|取消/.test(message)) return 'cancelled'
  if (hasPermissionDenied(error)) return 'permission-denied'
  return 'scan-failed'
}

export const showPermissionGuide = ({
  permissionLabel = '',
  action = '',
  uniApi = globalThis.uni
} = {}) => {
  return new Promise((resolve) => {
    if (!uniApi || typeof uniApi.showModal !== 'function') {
      return resolve(false)
    }

    uniApi.showModal({
      title: '需要权限',
      content: `${permissionLabel}未开启，无法${action}，请到设置中打开后重试。`,
      confirmText: '去设置',
      cancelText: '知道了',
      success: ({ confirm }) => {
        if (!confirm || typeof uniApi.openSetting !== 'function') {
          return resolve(false)
        }

        uniApi.openSetting({
          success: () => resolve(true),
          fail: () => resolve(false)
        })
      },
      fail: () => resolve(false)
    })
  })
}

const parsePermissionError = (error) => {
  if (hasPermissionDenied(error)) {
    return {
      granted: false,
      reason: 'denied',
      message: '未授予对应权限'
    }
  }

  return {
    granted: false,
    reason: 'error',
    message: String(error?.errMsg || error?.message || '权限校验失败')
  }
}

export const requestPermission = async ({
  scope,
  permissionLabel,
  action,
  uniApi = globalThis.uni
} = {}) => {
  if (!uniApi) {
    return {
      granted: true,
      reason: 'unsupported-platform'
    }
  }

  if (typeof uniApi.authorize !== 'function') {
    return {
      granted: true,
      reason: 'unsupported-platform'
    }
  }

  const checkedFromSetting = await new Promise((resolve) => {
    if (typeof uniApi.getSetting !== 'function') {
      return resolve({ unknown: true })
    }

    uniApi.getSetting({
      success: ({ authSetting = {} }) => {
        if (authSetting?.[scope] === true) {
          resolve({ granted: true })
          return
        }

        if (authSetting?.[scope] === false) {
          resolve({ granted: false, reason: 'denied', message: `${permissionLabel}已被拒绝` })
          return
        }

        resolve({ unknown: true })
      },
      fail: () => resolve({ unknown: true })
    })
  })

  if (checkedFromSetting.granted || checkedFromSetting.reason === 'denied') {
    return checkedFromSetting
  }

  return new Promise((resolve) => {
    uniApi.authorize({
      scope,
      success: () => resolve({ granted: true }),
      fail: (error) => resolve(parsePermissionError(error))
    })
  }).then((result) => {
    if (result.granted) return result

    if (result.reason === 'denied') {
      return showPermissionGuide({
        permissionLabel: permissionLabel || '权限',
        action: action || '继续',
        uniApi
      }).then(() => result)
    }

    if (typeof uniApi.showToast === 'function') {
      uniApi.showToast({
        title: result.message,
        icon: 'none'
      })
    }

    return result
  })
}

export const requestCameraPermission = ({ uniApi = globalThis.uni } = {}) => requestPermission({
  scope: CAMERA_SCOPE,
  permissionLabel: '相机权限',
  action: '扫码发起对局',
  uniApi
})

export const requestLocationPermission = ({ uniApi = globalThis.uni } = {}) => requestPermission({
  scope: LOCATION_SCOPE,
  permissionLabel: '位置信息权限',
  action: '查看附近球房',
  uniApi
})

export const scanQrCode = async ({
  onSuccess,
  onFailure,
  uniApi = globalThis.uni
} = {}) => {
  const permission = await requestCameraPermission({ uniApi })
  if (!permission.granted) {
    return {
      success: false,
      reason: permission.reason === 'denied' ? 'permission-denied' : 'permission-error',
      permission
    }
  }

  if (!uniApi || typeof uniApi.scanCode !== 'function') {
    if (typeof uniApi?.showToast === 'function') {
      uniApi.showToast({
        title: '当前端暂不支持扫码',
        icon: 'none'
      })
    }
    return {
      success: false,
      reason: 'unsupported'
    }
  }

  return new Promise((resolve) => {
    uniApi.scanCode({
      scanType: ['qrCode'],
      success: (res) => {
        onSuccess?.(res)
        resolve({ success: true, result: res })
      },
      fail: (error) => {
        onFailure?.(error)
        resolve({
          success: false,
          reason: classifyScanFailure(error),
          message: error?.errMsg || '扫码失败'
        })
      }
    })
  })
}

export const getCurrentLocation = async ({
  onPermissionDenied,
  uniApi = globalThis.uni,
  action = '查看附近球房'
} = {}) => {
  const permission = await requestPermission({
    scope: LOCATION_SCOPE,
    permissionLabel: '位置信息权限',
    action,
    uniApi
  })

  if (!permission.granted) {
    onPermissionDenied?.(permission)
    return { success: false, permission }
  }

  return new Promise((resolve) => {
    uniApi.getLocation({
      type: 'gcj02',
      success: (res) => resolve({
        success: true,
        location: { latitude: res.latitude, longitude: res.longitude }
      }),
      fail: (error) => resolve({
        success: false,
        message: error?.errMsg || '定位获取失败',
        permission
      })
    })
  })
}
