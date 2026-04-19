/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useCallback, useMemo, useState } from 'react'
import Editor from '@monaco-editor/react'
import { AlertCircle } from 'lucide-react'

import { Button } from '@renderer/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@renderer/components/ui/dialog'
import { Spinner } from '@renderer/components/ui/spinner'
import type { DynamoDBItem, DynamoDBTableSummary } from '../types'
import { marshallToFriendlyJson, unmarshallFromFriendlyJson } from '../types'

interface ItemEditorProps {
  tableInfo: DynamoDBTableSummary
  item: DynamoDBItem | null
  isOpen: boolean
  onClose: () => void
  onSave: (item: DynamoDBItem) => Promise<void>
}

export function ItemEditor({ tableInfo, item, isOpen, onClose, onSave }: ItemEditorProps) {
  const isEditMode = item !== null

  const initialJson = useMemo(() => {
    if (item) {
      return JSON.stringify(marshallToFriendlyJson(item), null, 2)
    }
    // Build a template from the table schema
    const template: Record<string, unknown> = {}
    for (const ks of tableInfo.KeySchema) {
      const attr = tableInfo.AttributeDefinitions.find(
        (a) => a.AttributeName === ks.AttributeName
      )
      if (attr?.AttributeType === 'N') {
        template[ks.AttributeName] = 0
      } else {
        template[ks.AttributeName] = ''
      }
    }
    return JSON.stringify(template, null, 2)
  }, [item, tableInfo])

  const [editorValue, setEditorValue] = useState(initialJson)
  const [error, setError] = useState<string | null>(null)
  const [isSaving, setIsSaving] = useState(false)

  const handleSave = useCallback(async () => {
    setError(null)
    try {
      const parsed = JSON.parse(editorValue)
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        setError('Item must be a JSON object.')
        return
      }

      // Validate that all key attributes are present
      for (const ks of tableInfo.KeySchema) {
        if (!(ks.AttributeName in parsed) || parsed[ks.AttributeName] === '' || parsed[ks.AttributeName] === null) {
          setError(`Missing required key attribute: "${ks.AttributeName}"`)
          return
        }
      }

      const dynamoItem = unmarshallFromFriendlyJson(parsed)
      setIsSaving(true)
      await onSave(dynamoItem)
    } catch (err) {
      if (err instanceof SyntaxError) {
        setError(`Invalid JSON: ${err.message}`)
      } else {
        setError(err instanceof Error ? err.message : 'Failed to save item.')
      }
    } finally {
      setIsSaving(false)
    }
  }, [editorValue, onSave, tableInfo])

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{isEditMode ? 'Edit Item' : 'Create Item'}</DialogTitle>
          <DialogDescription>
            {isEditMode
              ? 'Modify the item JSON below and save.'
              : 'Enter the item data as JSON. Key attributes are required.'}
          </DialogDescription>
        </DialogHeader>

        <div className="px-6 pb-2">
          {error && (
            <div className="mb-3 flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <div className="overflow-hidden rounded-xl border border-border">
            <Editor
              height="400px"
              defaultLanguage="json"
              value={editorValue}
              onChange={(val) => setEditorValue(val ?? '')}
              theme="vs-dark"
              loading={
                <div className="flex h-[400px] items-center justify-center">
                  <Spinner className="h-6 w-6 text-muted-foreground" />
                </div>
              }
              options={{
                minimap: { enabled: false },
                fontSize: 13,
                lineNumbers: 'on',
                scrollBeyondLastLine: false,
                automaticLayout: true,
                tabSize: 2,
                wordWrap: 'on',
                padding: { top: 12, bottom: 12 },
                renderLineHighlight: 'gutter',
                bracketPairColorization: { enabled: true },
                guides: { bracketPairs: true }
              }}
            />
          </div>

          <p className="mt-2 text-xs text-muted-foreground">
            Use plain JSON values — strings, numbers, booleans, arrays, objects. DynamoDB type annotations are handled automatically.
          </p>
        </div>

        <DialogFooter>
          <DialogClose render={<Button variant="ghost" />}>Cancel</DialogClose>
          <Button onClick={handleSave} loading={isSaving}>
            {isEditMode ? 'Save Changes' : 'Create Item'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
