import { useEffect, useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@renderer/components/ui/card"
import { Input } from "@renderer/components/ui/input"
import { Button } from "@renderer/components/ui/button"
import { toast } from "sonner"

export default function SettingsPage() {
  const [cliPath, setCliPath] = useState("")
  const [defaultCliPath, setDefaultCliPath] = useState("")
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)

  useEffect(() => {
    let mounted = true
    window.api.mildstack.getCliPath()
      .then((result) => {
        if (!mounted) return
        setCliPath(result.cliPath)
        setDefaultCliPath(result.defaultCliPath)
      })
      .catch((error) => {
        toast.error("Failed to load CLI settings", {
          description: error instanceof Error ? error.message : String(error)
        })
      })
      .finally(() => {
        if (mounted) setLoading(false)
      })

    return () => {
      mounted = false
    }
  }, [])

  const onSave = async () => {
    setSaving(true)
    try {
      const result = await window.api.mildstack.setCliPath(cliPath)
      setCliPath(result.cliPath)
      toast.success("CLI path updated")
    } catch (error) {
      toast.error("Failed to save CLI path", {
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
      toast.success("CLI path reset to default")
    } catch (error) {
      toast.error("Failed to reset CLI path", {
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
        throw new Error(result.error || "Invalid CLI path")
      }
      toast.success("CLI path works")
    } catch (error) {
      toast.error("CLI path test failed", {
        description: error instanceof Error ? error.message : String(error)
      })
    } finally {
      setTesting(false)
    }
  }

  return (
    <div className="mx-auto max-w-3xl">
      <Card>
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
              placeholder={defaultCliPath || "mildstack"}
              disabled={loading || saving}
            />
            <p className="text-xs text-muted-foreground">
              Default resolved path: <span className="font-mono">{defaultCliPath || "mildstack"}</span>
            </p>
          </div>

          <div className="flex items-center gap-2">
            <Button onClick={onSave} disabled={loading || saving || testing}>
              {saving ? "Saving..." : "Save"}
            </Button>
            <Button variant="outline" onClick={onReset} disabled={loading || saving || testing}>
              Reset to default
            </Button>
            <Button variant="secondary" onClick={onTest} disabled={loading || saving || testing}>
              {testing ? "Testing..." : "Test path"}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
