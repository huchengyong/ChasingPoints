const unsupportedExportError = () => {
  const error = new Error('当前设备暂不支持保存个人数据文件')
  error.code = 'PERSONAL_DATA_EXPORT_UNSUPPORTED'
  return error
}

const runtimeUni = (runtime = globalThis) => runtime?.uni || {}

export const getPersonalDataExportCapability = (runtime = globalThis) => {
  const uniApi = runtimeUni(runtime)
  const wxApi = runtime?.wx
  const fileSystem = wxApi?.getFileSystemManager?.() || uniApi.getFileSystemManager?.()
  const userDataPath = wxApi?.env?.USER_DATA_PATH || uniApi.env?.USER_DATA_PATH
  if (fileSystem && userDataPath && typeof fileSystem.writeFileSync === 'function' && typeof fileSystem.appendFileSync === 'function') {
    return { supported: true, platform: 'mini-program', fileSystem, userDataPath }
  }

  const plusApi = runtime?.plus
  if (plusApi?.io?.resolveLocalFileSystemURL) {
    return { supported: true, platform: 'app', plusApi }
  }
  return { supported: false, message: '当前设备暂不支持保存个人数据文件' }
}

const createMiniProgramWriter = (capability, fileName) => {
  const filePath = `${capability.userDataPath}/${fileName}`
  let initialized = false
  return {
    async write(chunk) {
      if (!initialized) {
        capability.fileSystem.writeFileSync(filePath, String(chunk), 'utf8')
        initialized = true
        return
      }
      capability.fileSystem.appendFileSync(filePath, String(chunk), 'utf8')
    },
    async complete() {
      return { filePath, platform: capability.platform }
    },
    async cleanup() {
      if (initialized && typeof capability.fileSystem.unlinkSync === 'function') {
        capability.fileSystem.unlinkSync(filePath)
      }
    }
  }
}

const resolvePlusFileWriter = (plusApi, fileName) => new Promise((resolve, reject) => {
  plusApi.io.resolveLocalFileSystemURL(plusApi.io.PRIVATE_DOC, (root) => {
    root.getFile(fileName, { create: true }, (entry) => {
      entry.createWriter((writer) => {
        writer.onwriteend = () => resolve({ entry, writer })
        writer.onerror = reject
        writer.truncate(0)
      }, reject)
    }, reject)
  }, reject)
})

const createAppWriter = async (capability, fileName) => {
  const { entry, writer } = await resolvePlusFileWriter(capability.plusApi, fileName)
  const write = (chunk) => new Promise((resolve, reject) => {
    writer.onwriteend = resolve
    writer.onerror = reject
    writer.seek(writer.length)
    writer.write(String(chunk))
  })
  return {
    write,
    async complete() {
      return {
        filePath: typeof entry.toURL === 'function' ? entry.toURL() : entry.fullPath,
        platform: capability.platform
      }
    },
    async cleanup() {
      await new Promise((resolve, reject) => entry.remove(resolve, reject))
    }
  }
}

export const createPersonalDataExportFileWriter = async ({ fileName, runtime = globalThis } = {}) => {
  const capability = getPersonalDataExportCapability(runtime)
  if (!capability.supported || !fileName) throw unsupportedExportError()
  if (capability.platform === 'mini-program') return createMiniProgramWriter(capability, fileName)
  return createAppWriter(capability, fileName)
}

export const openPersonalDataExportFile = ({ filePath, runtime = globalThis } = {}) => new Promise((resolve, reject) => {
  const uniApi = runtimeUni(runtime)
  if (!filePath || typeof uniApi.openDocument !== 'function') {
    reject(unsupportedExportError())
    return
  }
  uniApi.openDocument({
    filePath,
    showMenu: true,
    success: resolve,
    fail: reject
  })
})
