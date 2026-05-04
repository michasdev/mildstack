import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@renderer/components/ui/card'
import { Input } from '@renderer/components/ui/input'
import { Button } from '@renderer/components/ui/button'
import { toast } from 'sonner'

type AppUpdateState =
  | 'idle'
  | 'checking'
  | 'available'
  | 'not-available'
  | 'downloading'
  | 'downloaded'
  | 'unsupported'
  | 'error'

interface AppUpdateStatus {
  currentVersion: string
  state: AppUpdateState
  availableVersion?: string
  lastCheckedAt?: string
  error?: string
}

const updateStatusLabels: Record<AppUpdateState, string> = {
  idle: 'Ready to check for updates.',
  checking: 'Checking for updates...',
  available: 'A new version is available.',
  'not-available': 'You are on the latest version.',
  downloading: 'Downloading update...',
  downloaded: 'Update downloaded. The app will restart to finish installation.',
  unsupported: 'Update checks are not available in this environment.',
  error: 'Could not verify updates.'
}

export default function SettingsPage() {
  const [cliPath, setCliPath] = useState('')
  const [defaultCliPath, setDefaultCliPath] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [checkingUpdates, setCheckingUpdates] = useState(false)
  const [installingUpdate, setInstallingUpdate] = useState(false)
  const [updateStatus, setUpdateStatus] = useState<AppUpdateStatus | null>(null)

  useEffect(() => {
    let mounted = true

    const loadSettings = async () => {
      const [cliResult, updateResult] = await Promise.allSettled([
        window.api.mildstack.getCliPath(),
        window.api.mildstack.getAppUpdateStatus()
      ])

      if (!mounted) return

      if (cliResult.status === 'fulfilled') {
        setCliPath(cliResult.value.cliPath)
        setDefaultCliPath(cliResult.value.defaultCliPath)
      } else {
        toast.error('Failed to load CLI settings', {
          description: cliResult.reason instanceof Error ? cliResult.reason.message : String(cliResult.reason)
        })
      }

      if (updateResult.status === 'fulfilled') {
        setUpdateStatus(updateResult.value)
      } else {
        toast.error('Failed to load app update status', {
          description: updateResult.reason instanceof Error ? updateResult.reason.message : String(updateResult.reason)
        })
      }

      setLoading(false)
    }

    void loadSettings()

    return () => {
      mounted = false
    }
  }, [])

  const onSave = async () => {
    setSaving(true)
    try {
      const result = await window.api.mildstack.setCliPath(cliPath)
      setCliPath(result.cliPath)
      toast.success('CLI path updated')
    } catch (error) {
      toast.error('Failed to save CLI path', {
        description: error instanceof Error ? error.message : String(error)
      })
    } finally {
      setSaving(false)
    }
  }

  const onReset = async () => {
    setSaving(true)
    try {
      const result = await window.api.mildstack.resetCliPath()
      setCliPath(result.cliPath)
      toast.success('CLI path reset to default')
    } catch (error) {
      toast.error('Failed to reset CLI path', {
        description: error instanceof Error ? error.message : String(error)
      })
    } finally {
      setSaving(false)
    }
  }

  const onTest = async () => {
    setTesting(true)
    try {
      const result = await window.api.mildstack.testCliPath()
      if (!result.valid) {
        throw new Error(result.error || 'Invalid CLI path')
      }
      toast.success('CLI path works')
    } catch (error) {
      toast.error('CLI path test failed', {
        description: error instanceof Error ? error.message : String(error)
      })
    } finally {
      setTesting(false)
    }
  }

  const onCheckUpdates = async () => {
    setCheckingUpdates(true)
    try {
      const status = await window.api.mildstack.checkAppUpdates()
      setUpdateStatus(status)

      if (status.state === 'available') {
        toast.success('Update available', {
          description: `Version ${status.availableVersion || 'new'} is ready to install.`
        })
        return
      }

      if (status.state === 'not-available') {
        toast.success('You are on the latest version')
        return
      }

      if (status.state === 'unsupported') {
        toast.info('Update checks unavailable', {
          description: status.error
        })
        return
      }

      if (status.state === 'error') {
        toast.error('Failed to check updates', {
          description: status.error
        })
      }
    } catch (error) {
      toast.error('Failed to check updates', {
        description: error instanceof Error ? error.message : String(error)
      })
    } finally {
      setCheckingUpdates(false)
    }
  }

  const onInstallUpdate = async () => {
    setInstallingUpdate(true)
    try {
      const status = await window.api.mildstack.installAppUpdate()
      setUpdateStatus(status)

      if (status.state === 'error' || status.state === 'unsupported') {
        toast.error('Failed to install update', {
          description: status.error || 'No update is ready to install.'
        })
        return
      }

      toast.success('Installing update...', {
        description: 'The app will restart automatically when the update is ready.'
      })
    } catch (error) {
      toast.error('Failed to install update', {
        description: error instanceof Error ? error.message : String(error)
      })
    } finally {
      setInstallingUpdate(false)
    }
  }

  const canInstallUpdate = updateStatus?.state === 'available' || updateStatus?.state === 'downloaded'
  const updateStatusLabel = updateStatus ? updateStatusLabels[updateStatus.state] : 'Loading update status...'

  return (
    <div className="mx-auto max-w-3xl">
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>CLI Settings</CardTitle>
          <CardDescription>
            Configure the MildStack CLI executable path used by all desktop IPC commands.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <p className="text-sm font-medium">Executable path</p>
            <Input
              value={cliPath}
              onChange={(e) => setCliPath(e.target.value)}
              placeholder={defaultCliPath || 'mildstack'}
              disabled={loading || saving}
            />
            <p className="text-xs text-muted-foreground">
              Default resolved path: <span className="font-mono">{defaultCliPath || 'mildstack'}</span>
            </p>
          </div>

          <div className="flex items-center gap-2">
            <Button onClick={onSave} disabled={loading || saving || testing}>
              {saving ? 'Saving...' : 'Save'}
            </Button>
            <Button variant="outline" onClick={onReset} disabled={loading || saving || testing}>
              Reset to default
            </Button>
            <Button variant="secondary" onClick={onTest} disabled={loading || saving || testing}>
              {testing ? 'Testing...' : 'Test path'}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Application Updates</CardTitle>
          <CardDescription>Check for new releases and install them manually.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <p className="text-sm font-medium">
              Current version: <span className="font-mono">{updateStatus?.currentVersion || '...'}</span>
            </p>
            <p className="text-sm text-muted-foreground">{updateStatusLabel}</p>
            {updateStatus?.availableVersion ? (
              <p className="text-xs text-muted-foreground">
                Available version: <span className="font-mono">{updateStatus.availableVersion}</span>
              </p>
            ) : null}
            {updateStatus?.error ? (
              <p className="text-xs text-destructive">{updateStatus.error}</p>
            ) : null}
          </div>

          <div className="flex items-center gap-2">
            <Button onClick={onCheckUpdates} disabled={checkingUpdates || installingUpdate || loading}>
              {checkingUpdates ? 'Checking...' : 'Check for updates'}
            </Button>
            <Button
              variant="secondary"
              onClick={onInstallUpdate}
              disabled={!canInstallUpdate || checkingUpdates || installingUpdate || loading}
            >
              {installingUpdate ? 'Updating...' : 'Update now'}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
