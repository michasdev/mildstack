import { app, shell, BrowserWindow, ipcMain } from 'electron'
import { join } from 'path'
import { electronApp, optimizer, is } from '@electron-toolkit/utils'
import { autoUpdater } from 'electron-updater'
import icon from '../../build/icon.png?asset'
import { registerS3IpcHandlers } from './s3-ipc'
import { registerDynamoDBIpcHandlers } from './dynamodb-ipc'
import { registerSQSIpcHandlers } from './sqs-ipc'
import { registerMildStackIpcHandlers } from './mildstack-ipc'
import { setupCliInstaller } from './setup-cli'

// Set app name for macOS Dock and Menu Bar as early as possible
if (process.platform === 'darwin') {
  app.name = 'MildStack Desktop'
  app.setName('MildStack Desktop')
}

function checkForUpdates(): void {
  if (is.dev) return

  autoUpdater.on('error', (error) => {
    console.error('Auto updater error:', error)
  })

  autoUpdater.checkForUpdatesAndNotify().catch((error) => {
    console.error('Auto updater check failed:', error)
  })
}

function createWindow(): void {
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 900,
    height: 670,
    show: false,
    autoHideMenuBar: true,
    icon,
    webPreferences: {
      preload: join(__dirname, '../preload/index.js'),
      sandbox: false,
      webSecurity: true
    }
  })

  mainWindow.on('ready-to-show', () => {
    mainWindow.show()
  })

  mainWindow.webContents.setWindowOpenHandler((details) => {
    shell.openExternal(details.url)
    return { action: 'deny' }
  })

  // HMR for renderer base on electron-vite cli.
  // Load the remote URL for development or the local html file for production.
  if (is.dev && process.env['ELECTRON_RENDERER_URL']) {
    mainWindow.loadURL(process.env['ELECTRON_RENDERER_URL'])
  } else {
    mainWindow.loadFile(join(__dirname, '../renderer/index.html'))
  }
}

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.

app.whenReady().then(() => {
  // Set app user model id for windows
  electronApp.setAppUserModelId('com.michasdev.mildstack-desktop')

  // Set Dock icon for macOS in development
  if (is.dev && process.platform === 'darwin') {
    app.dock?.setIcon(icon)
  }

  // Default open or close DevTools by F12 in development
  // and ignore CommandOrControl + R in production.
  // see https://github.com/alex8088/electron-toolkit/tree/master/packages/utils
  app.on('browser-window-created', (_, window) => {
    optimizer.watchWindowShortcuts(window)
  })

  // IPC test
  ipcMain.on('ping', () => console.log('pong'))
  registerS3IpcHandlers()
  registerDynamoDBIpcHandlers()
  registerSQSIpcHandlers()
  registerMildStackIpcHandlers()

  createWindow()
  checkForUpdates()
  setupCliInstaller()

  app.on('activate', function () {
    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (BrowserWindow.getAllWindows().length === 0) createWindow()
  })
})

// Quit when all windows are closed, except on macOS. There, it's common
// for applications and their menu bar to stay active until the user quits
// explicitly with Cmd + Q.
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and require them here.
