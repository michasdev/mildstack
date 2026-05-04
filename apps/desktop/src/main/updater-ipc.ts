import { app, ipcMain } from 'electron'
import { is } from '@electron-toolkit/utils'
import { autoUpdater } from 'electron-updater'

type AppUpdateState =
  | 'idle'
  | 'checking'
  | 'available'
  | 'not-available'
  | 'downloading'
  | 'downloaded'
  | 'unsupported'
  | 'error'

export interface AppUpdateStatus {
  currentVersion: string
  state: AppUpdateState
  availableVersion?: string
  lastCheckedAt?: string
  error?: string
}

let handlersRegistered = false
let listenersRegistered = false

let updateStatus: AppUpdateStatus = {
  currentVersion: app.getVersion(),
  state: is.dev ? 'unsupported' : 'idle'
}

function getStatus(): AppUpdateStatus {
  return {
    ...updateStatus,
    currentVersion: app.getVersion()
  }
}

function setStatus(next: Partial<AppUpdateStatus>): void {
  updateStatus = {
    ...updateStatus,
    ...next,
    currentVersion: app.getVersion()
  }
}

function parseUpdaterError(error: unknown): string {
  if (error instanceof Error) return error.message
  return String(error)
}

function updatesSupported(): boolean {
  return app.isPackaged && !is.dev
}

function ensureUpdaterListeners(): void {
  if (listenersRegistered) return
  listenersRegistered = true

  autoUpdater.autoDownload = false
  autoUpdater.autoInstallOnAppQuit = false

  autoUpdater.on('checking-for-update', () => {
    setStatus({
      state: 'checking',
      error: undefined
    })
  })

  autoUpdater.on('update-available', (info) => {
    setStatus({
      state: 'available',
      availableVersion: info.version,
      error: undefined
    })
  })

  autoUpdater.on('update-not-available', () => {
    setStatus({
      state: 'not-available',
      availableVersion: undefined,
      error: undefined
    })
  })

  autoUpdater.on('download-progress', () => {
    setStatus({
      state: 'downloading',
      error: undefined
    })
  })

  autoUpdater.on('update-downloaded', (info) => {
    setStatus({
      state: 'downloaded',
      availableVersion: info.version,
      error: undefined
    })
  })

  autoUpdater.on('error', (error) => {
    setStatus({
      state: 'error',
      error: parseUpdaterError(error)
    })
    console.error('Auto updater error:', error)
  })
}

async function handleCheckForUpdates(): Promise<AppUpdateStatus> {
  if (!updatesSupported()) {
    setStatus({
      state: 'unsupported',
      availableVersion: undefined,
      error: 'Update checks are only available in packaged builds.',
      lastCheckedAt: new Date().toISOString()
    })
    return getStatus()
  }

  ensureUpdaterListeners()

  try {
    setStatus({
      state: 'checking',
      availableVersion: undefined,
      error: undefined,
      lastCheckedAt: new Date().toISOString()
    })

    const result = await autoUpdater.checkForUpdates()
    setStatus({ lastCheckedAt: new Date().toISOString() })

    if (!result) {
      setStatus({
        state: 'unsupported',
        error: 'Updater is not active in this build.'
      })
      return getStatus()
    }

    if (result.isUpdateAvailable) {
      setStatus({
        state: 'available',
        availableVersion: result.updateInfo.version,
        error: undefined
      })
      return getStatus()
    }

    setStatus({
      state: 'not-available',
      availableVersion: undefined,
      error: undefined
    })
    return getStatus()
  } catch (error) {
    setStatus({
      state: 'error',
      error: parseUpdaterError(error)
    })
    return getStatus()
  }
}

async function handleInstallUpdate(): Promise<AppUpdateStatus> {
  if (!updatesSupported()) {
    setStatus({
      state: 'unsupported',
      availableVersion: undefined,
      error: 'Updates are only available in packaged builds.'
    })
    return getStatus()
  }

  ensureUpdaterListeners()

  if (updateStatus.state === 'downloaded') {
    autoUpdater.quitAndInstall()
    return getStatus()
  }

  if (updateStatus.state !== 'available') {
    setStatus({
      state: 'error',
      error: 'No update ready to install. Check for updates first.'
    })
    return getStatus()
  }

  try {
    setStatus({
      state: 'downloading',
      error: undefined
    })

    await autoUpdater.downloadUpdate()

    setStatus({
      state: 'downloaded',
      error: undefined
    })

    autoUpdater.quitAndInstall()
    return getStatus()
  } catch (error) {
    setStatus({
      state: 'error',
      error: parseUpdaterError(error)
    })
    return getStatus()
  }
}

export function registerUpdaterIpcHandlers(): void {
  if (handlersRegistered) return
  handlersRegistered = true

  ensureUpdaterListeners()

  ipcMain.handle('app:update:status', async () => getStatus())
  ipcMain.handle('app:update:check', async () => handleCheckForUpdates())
  ipcMain.handle('app:update:install', async () => handleInstallUpdate())
}

export function checkForUpdatesInBackground(): void {
  if (!updatesSupported()) return

  ensureUpdaterListeners()

  autoUpdater.checkForUpdates().catch((error) => {
    setStatus({
      state: 'error',
      error: parseUpdaterError(error)
    })
    console.error('Auto updater check failed:', error)
  })
}
