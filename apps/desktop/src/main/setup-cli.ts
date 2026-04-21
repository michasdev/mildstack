import { app } from 'electron'
import { join } from 'path'
import fs from 'fs'

export function setupCliInstaller() {
  if (!app.isPackaged) return

  try {
    const binName = process.platform === 'win32' ? 'mildstack.exe' : 'mildstack'
    const targetCliPath = join(process.resourcesPath, 'bin', binName)

    if (!fs.existsSync(targetCliPath)) {
      console.warn('CLI binary not found at', targetCliPath)
      return
    }

    if (process.platform === 'darwin' || process.platform === 'linux') {
      const linkPath = '/usr/local/bin/mildstack'
      
      const createSymlink = (target: string, link: string) => {
        if (fs.existsSync(link) || fs.lstatSync(link, { throwIfNoEntry: false })) {
          try {
            const existingTarget = fs.readlinkSync(link)
            if (existingTarget === target) return true // Already correctly linked
          } catch (e) {
            // Not a symlink or can't read it
          }
          fs.unlinkSync(link)
        }
        fs.symlinkSync(target, link)
        return true
      }

      try {
        if (createSymlink(targetCliPath, linkPath)) {
          console.log(`Created symlink for CLI at ${linkPath}`)
        }
      } catch (e) {
        console.error(`Failed to create symlink at ${linkPath}`, e)
        // Fallback to ~/.local/bin
        try {
          const homeDir = app.getPath('home')
          const localBinDir = join(homeDir, '.local', 'bin')
          if (!fs.existsSync(localBinDir)) {
            fs.mkdirSync(localBinDir, { recursive: true })
          }
          const localLinkPath = join(localBinDir, 'mildstack')
          if (createSymlink(targetCliPath, localLinkPath)) {
            console.log(`Created symlink for CLI at ${localLinkPath}`)
          }
        } catch (fallbackError) {
          console.error('Failed to setup CLI fallback symlink', fallbackError)
        }
      }
    } else if (process.platform === 'win32') {
      console.log(`CLI binary is available at: ${targetCliPath}`)
      // Windows PATH setup usually requires external scripts or installers.
    }
  } catch (error) {
    console.error('Unexpected error setting up CLI', error)
  }
}
