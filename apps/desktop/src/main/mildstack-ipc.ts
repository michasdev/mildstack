import { ipcMain } from "electron"
import { getActiveInstancePort, setActiveInstancePort } from "./instance-state"
import {
    getConfiguredMildStackExecutablePath,
    getDefaultResolvedMildStackExecutablePath,
    resetConfiguredMildStackExecutablePath,
    setConfiguredMildStackExecutablePath,
    runMildStackCli
} from './mildstack-cli'

export type MildStackInstanceStatus = 'running' | 'not_started' | 'errored'

export interface MildStackInstance {
    instanceId: string
    port: number
    pid?: number
    status: MildStackInstanceStatus
    error?: string
}

export interface MildStackInstancesResponse {
    state: string
    services: Array<{
        name: string
        version: string
        tags: string[]
    }>
    instances: MildStackInstance[]
    ports: number[] | null
}

/**
 * Parses CLI error output. The CLI logs the error message on the first line
 * and exits with status 1. execAsync rejects on non-zero exit, so we catch
 * and extract.
 */
function parseCliError(err: unknown): string {
    if (err && typeof err === 'object' && 'stderr' in err) {
        const stderr = (err as { stderr: string }).stderr?.trim()
        if (stderr) {
            // First line is usually "Error: <message>"
            const firstLine = stderr.split('\n')[0]
            return firstLine.replace(/^Error:\s*/, '')
        }
    }
    if (err && typeof err === 'object' && 'message' in err) {
        return (err as Error).message
    }
    return 'Unknown CLI error'
}

export function registerMildStackIpcHandlers(): void {
    ipcMain.handle('mildstack:cliPath:get', async () => {
        return {
            cliPath: getConfiguredMildStackExecutablePath(),
            defaultCliPath: getDefaultResolvedMildStackExecutablePath()
        }
    })

    ipcMain.handle('mildstack:cliPath:set', async (_event, cliPath: string) => {
        const nextPath = setConfiguredMildStackExecutablePath(cliPath)
        return { cliPath: nextPath }
    })

    ipcMain.handle('mildstack:cliPath:reset', async () => {
        const nextPath = resetConfiguredMildStackExecutablePath()
        return { cliPath: nextPath }
    })

    ipcMain.handle('mildstack:cliPath:test', async () => {
        try {
            await runMildStackCli(['instances', '--json'])
            return { valid: true }
        } catch (err) {
            const error = parseCliError(err)
            return { valid: false, error }
        }
    })

    ipcMain.handle('instance:port', async () => {
        return getActiveInstancePort()
    })

    ipcMain.handle('instance:setSelected', (_event, port: number) => {
        setActiveInstancePort(port)
        return true
    })

    ipcMain.handle('mildstack:instances', async (_event): Promise<MildStackInstancesResponse> => {
        try {
            const { stdout } = await runMildStackCli(['instances', '--json'])
            return JSON.parse(stdout)
        } catch (err) {
            // If no instances exist or CLI is not available, return empty state
            console.error('[MildStack IPC] instances error:', err)
            return {
                state: 'not_started',
                services: [],
                instances: [],
                ports: null
            }
        }
    })

    ipcMain.handle('mildstack:start', async (_event, port: number): Promise<{ success: boolean; error?: string }> => {
        try {
            // --d flag to detach (run in background)
            await runMildStackCli(['start', String(port), '--d'])
            return { success: true }
        } catch (err) {
            const error = parseCliError(err)
            console.error('[MildStack IPC] start error:', error)
            return { success: false, error }
        }
    })

    ipcMain.handle('mildstack:stop', async (_event, args: { port?: number; all?: boolean }): Promise<{ success: boolean; error?: string }> => {
        try {
            const cliArgs = ['stop']
            if (args.all) {
                cliArgs.push('--all')
            } else if (args.port) {
                cliArgs.push(String(args.port))
            }
            cliArgs.push('--json')
            await runMildStackCli(cliArgs)
            return { success: true }
        } catch (err) {
            const error = parseCliError(err)
            console.error('[MildStack IPC] stop error:', error)
            return { success: false, error }
        }
    })

    ipcMain.handle('mildstack:delete', async (_event, args: { port?: number; all?: boolean }): Promise<{ success: boolean; error?: string }> => {
        try {
            const cliArgs = ['delete']
            if (args.all) {
                cliArgs.push('--all')
            } else if (args.port) {
                cliArgs.push(String(args.port))
            }
            cliArgs.push('--json')
            await runMildStackCli(cliArgs)
            return { success: true }
        } catch (err) {
            const error = parseCliError(err)
            console.error('[MildStack IPC] delete error:', error)
            return { success: false, error }
        }
    })

    // Validation handler — checks if the selected instance is running
    ipcMain.handle('mildstack:validateInstance', async (_event): Promise<{ valid: boolean; error?: string }> => {
        const port = getActiveInstancePort()
        try {
            const { stdout } = await runMildStackCli(['instances', '--json'])
            const response: MildStackInstancesResponse = JSON.parse(stdout)
            const instance = response.instances.find(i => i.port === port)

            if (!instance) {
                return { valid: false, error: `No instance found on port ${port}` }
            }
            if (instance.status !== 'running') {
                return { valid: false, error: `Instance on port ${port} is not running (status: ${instance.status})` }
            }
            return { valid: true }
        } catch {
            return { valid: false, error: `Unable to verify instance on port ${port}` }
        }
    })
}
