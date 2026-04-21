import { getActiveInstancePort } from './instance-state'

export function resolveLocalEndpoint(serviceName: string): string {
  const normalizedServiceName = serviceName.trim().toUpperCase()
  const envKey = `MILDSTACK_${normalizedServiceName}_ENDPOINT`
  const override = process.env[envKey]?.trim()
  if (override) {
    return override
  }

  const port = getActiveInstancePort()
  return `http://localhost:${port}`
}
