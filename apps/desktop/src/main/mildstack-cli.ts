import { app } from 'electron'
import { join } from 'path'
import fs from 'fs'
import { exec, execFile } from 'node:child_process'
import { promisify } from 'node:util'

/**
 * Resolves the CLI executable path used by IPC handlers.
 * In packaged builds we prefer the bundled binary under resources/bin.
 * In development we fallback to PATH or MAIN_VITE_MILDSTACK_EXECUTABLE.
 */
const execFileAsync = promisify(execFile)
const execAsync = promisify(exec)

function getDefaultMildStackExecutablePath(): string {
  const defaultExecutable = import.meta.env.MAIN_VITE_MILDSTACK_EXECUTABLE || 'mildstack'

  if (!app.isPackaged) {
    return defaultExecutable
  }

  const binName = process.platform === 'win32' ? 'mildstack.exe' : 'mildstack'
  return join(process.resourcesPath, 'bin', binName)
}

type CliSettings = {
  cliPath: string
}

const settingsFileName = 'mildstack-desktop-settings.json'
let cachedCliPath: string | null = null

function getSettingsFilePath(): string {
  return join(app.getPath('userData'), settingsFileName)
}

function readSettings(): CliSettings | null {
  try {
    const filePath = getSettingsFilePath()
    if (!fs.existsSync(filePath)) return null
    const raw = fs.readFileSync(filePath, 'utf-8')
    const parsed = JSON.parse(raw) as Partial<CliSettings>
    if (typeof parsed.cliPath === 'string') {
      return { cliPath: parsed.cliPath.trim() }
    }
    return null
  } catch {
    return null
  }
}

function writeSettings(settings: CliSettings): void {
  const filePath = getSettingsFilePath()
  fs.writeFileSync(filePath, JSON.stringify(settings, null, 2), 'utf-8')
}

export function getConfiguredMildStackExecutablePath(): string {
  if (cachedCliPath) return cachedCliPath

  const saved = readSettings()
  if (saved?.cliPath) {
    cachedCliPath = saved.cliPath
    return cachedCliPath
  }

  cachedCliPath = getDefaultMildStackExecutablePath()
  return cachedCliPath
}

function isDevShellCommand(value: string): boolean {
  if (app.isPackaged) return false

  // Only enable arbitrary shell commands in local development mode.
  return (
    value.includes('&&') ||
    value.includes('||') ||
    value.includes(';') ||
    value.includes('|') ||
    value.includes('>') ||
    value.includes('<') ||
    value.includes('$(') ||
    value.startsWith('cd ') ||
    value.startsWith('go ')
  )
}

function shellEscapeArg(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`
}

export function setConfiguredMildStackExecutablePath(path: string): string {
  const normalized = path.trim()
  if (!normalized) {
    throw new Error('CLI path cannot be empty.')
  }

  if (isDevShellCommand(normalized)) {
    cachedCliPath = normalized
    writeSettings({ cliPath: normalized })
    return cachedCliPath
  }

  // If user provides an explicit filesystem path, validate it early.
  const looksLikePath =
    normalized.includes('/') ||
    normalized.startsWith('.') ||
    normalized.startsWith('~')
  if (looksLikePath) {
    const expanded = normalized.startsWith('~')
      ? join(app.getPath('home'), normalized.slice(1))
      : normalized
    if (!fs.existsSync(expanded)) {
      throw new Error(`CLI path does not exist: ${expanded}`)
    }
    const stat = fs.statSync(expanded)
    if (!stat.isFile()) {
      throw new Error(`CLI path is not a file: ${expanded}`)
    }
  }

  cachedCliPath = normalized
  writeSettings({ cliPath: normalized })
  return cachedCliPath
}

export function resetConfiguredMildStackExecutablePath(): string {
  const next = getDefaultMildStackExecutablePath()
  cachedCliPath = next
  writeSettings({ cliPath: next })
  return next
}

export function getDefaultResolvedMildStackExecutablePath(): string {
  return getDefaultMildStackExecutablePath()
}

export function resolveMildStackExecutablePath(): string {
  return getConfiguredMildStackExecutablePath()
}

export async function runMildStackCli(args: string[]): Promise<{ stdout: string; stderr: string }> {
  const executable = resolveMildStackExecutablePath()
  if (isDevShellCommand(executable)) {
    const argv = args.map(shellEscapeArg).join(' ')
    const command = `${executable} ${argv}`.trim()
    return execAsync(command, { shell: process.env.SHELL || '/bin/zsh' })
  }
  return execFileAsync(executable, args, { shell: false })
}
