import { ipcMain } from 'electron'
import { getActiveInstancePort } from './instance-state'
import { runMildStackCli } from './mildstack-cli'

/**
 * Validates that the currently selected instance is running before
 * allowing S3/DynamoDB commands to execute. This is called as a
 * pre-check within IPC handlers.
 * 
 * Returns true if the instance is running, throws otherwise.
 */
export async function assertInstanceRunning(): Promise<void> {
    const port = getActiveInstancePort()
    try {
        const { stdout } = await runMildStackCli(['instances', '--json'])
        const response = JSON.parse(stdout)
        const instance = response.instances?.find((i: { port: number }) => i.port === port)

        if (!instance) {
            throw new Error(`No MildStack instance found on port ${port}. Please start an instance first.`)
        }
        if (instance.status !== 'running') {
            throw new Error(
                `MildStack instance on port ${port} is not running (status: ${instance.status}). Please start it first.`
            )
        }
    } catch (err) {
        if (err instanceof Error && err.message.includes('MildStack instance')) {
            throw err
        }
        throw new Error(`Unable to verify MildStack instance on port ${port}. Is the CLI installed?`)
    }
}

/**
 * Wraps an ipcMain.handle callback with instance validation.
 * Before the handler executes, it checks that the selected instance is running.
 */
export function withInstanceValidation<T extends unknown[], R>(
    handler: (event: Electron.IpcMainInvokeEvent, ...args: T) => Promise<R>
): (event: Electron.IpcMainInvokeEvent, ...args: T) => Promise<R> {
    return async (event, ...args) => {
        await assertInstanceRunning()
        return handler(event, ...args)
    }
}

/**
 * Registers an IPC handler with instance validation middleware.
 * Usage: registerValidatedHandler('channel', async (event, args) => { ... })
 */
export function registerValidatedHandler<T extends unknown[], R>(
    channel: string,
    handler: (event: Electron.IpcMainInvokeEvent, ...args: T) => Promise<R>
): void {
    ipcMain.handle(channel, withInstanceValidation(handler))
}
