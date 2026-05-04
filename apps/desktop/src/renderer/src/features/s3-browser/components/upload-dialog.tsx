/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useState, useRef, useEffect } from 'react'
import { Upload, X, File as FileIcon, Loader2 } from 'lucide-react'
import { Button } from '@renderer/components/ui/button'
import {
  Dialog,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogContent,
  DialogDescription
} from '@renderer/components/ui/dialog'
import { cn } from '@renderer/lib/utils'
import type { S3BrowserApi } from '../types'
import { toast } from "sonner"

interface UploadDialogProps {
  api: S3BrowserApi
  bucketName: string
  region: string
  currentPrefix: string
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
  initialFiles?: FileList | null
}

const MAX_FILE_SIZE = 20 * 1024 * 1024 // 20MB

export function UploadDialog({
  api,
  bucketName,
  region,
  currentPrefix,
  isOpen,
  onClose,
  onSuccess,
  initialFiles
}: UploadDialogProps) {
  const [files, setFiles] = useState<File[]>([])
  const [uploading, setUploading] = useState(false)
  const [dragActive, setDragActive] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (initialFiles && isOpen) {
      addFiles(initialFiles)
    }
  }, [initialFiles, isOpen])

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true)
    } else if (e.type === 'dragleave') {
      setDragActive(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      addFiles(e.dataTransfer.files)
    }
  }

  const addFiles = (newFileList: FileList) => {
    const validFiles: File[] = []
    const invalidFiles: File[] = []

    Array.from(newFileList).forEach((file) => {
      if (file.size <= MAX_FILE_SIZE) {
        validFiles.push(file)
      } else {
        invalidFiles.push(file)
      }
    })

    if (invalidFiles.length > 0) {
      toast.warning('File too large', {
        description: `${invalidFiles.length} file(s) exceed the 20MB limit and were ignored.`,
      })
    }

    setFiles((prev) => [...prev, ...validFiles])
  }

  const removeFile = (index: number) => {
    setFiles((prev) => prev.filter((_, i) => i !== index))
  }

  const handleUpload = async () => {
    if (files.length === 0) return

    setUploading(true)
    let successCount = 0
    let errorCount = 0

    try {
      for (const file of files) {
        const key = `${currentPrefix}${file.name}`
        try {
          const arrayBuffer = await file.arrayBuffer()
          await api.putObject(
            bucketName,
            key,
            arrayBuffer,
            file.type || 'application/octet-stream',
            region
          )
          successCount++
        } catch (err) {
          console.error(`Failed to upload ${file.name}`, err)
          errorCount++
        }
      }

      if (successCount > 0) {
        toast.success('Upload successful', {
          description: `Uploaded ${successCount} file(s) to ${bucketName}`,
        })
        onSuccess()
      }

      if (errorCount > 0) {
        toast.error('Upload partially failed', {
          description: `Failed to upload ${errorCount} file(s).`,
        })
      }

      if (errorCount === 0) {
        onClose()
        setFiles([])
      } else {
        // Clear only successful files if we had errors (optional, for now just clear all)
        setFiles([])
      }
    } finally {
      setUploading(false)
    }
  }

  const handleClose = () => {
    if (uploading) return
    setFiles([])
    onClose()
  }

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && handleClose()}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Upload Files</DialogTitle>
          <DialogDescription>
            Upload files to <span className="font-mono text-primary">{currentPrefix || '/'}</span>
          </DialogDescription>
        </DialogHeader>

        <div
          className={cn(
            'flex flex-col gap-4',
            uploading && 'pointer-events-none opacity-50'
          )}
        >
          <div
            className={cn(
              'relative flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-8 transition-colors',
              dragActive ? 'border-primary bg-primary/10' : 'border-border bg-muted/30',
              uploading && 'pointer-events-none opacity-50'
            )}
            onDragEnter={handleDrag}
            onDragLeave={handleDrag}
            onDragOver={handleDrag}
            onDrop={handleDrop}
          >
            <input
              ref={fileInputRef}
              type="file"
              multiple
              className="hidden"
              onChange={(e) => e.target.files && addFiles(e.target.files)}
            />

            <Upload className="mb-4 h-10 w-10 text-muted-foreground" />
            <p className="mb-2 text-sm font-medium">
              Drag & drop files here or{' '}
              <button
                type="button"
                className="text-primary hover:underline"
                onClick={() => fileInputRef.current?.click()}
              >
                browse
              </button>
            </p>
            <p className="text-xs text-muted-foreground">Max file size: 20MB</p>
          </div>

          {files.length > 0 && (
            <div className="mt-4 overflow-hidden rounded-lg border border-border">
              {files.map((file, i) => (
                <div
                  key={`${file.name}-${i}`}
                  className="flex items-center justify-between border-b border-border p-3 last:border-none"
                >
                  <div className="flex items-center gap-3 overflow-hidden">
                    <FileIcon className="h-4 w-4 shrink-0 text-muted-foreground" />
                    <span className="truncate text-sm font-medium">{file.name}</span>
                    <span className="shrink-0 text-xs text-muted-foreground">
                      ({(file.size / 1024 / 1024).toFixed(2)} MB)
                    </span>
                  </div>
                  {!uploading && (
                    <button
                      type="button"
                      onClick={() => removeFile(i)}
                      className="text-muted-foreground hover:text-destructive"
                    >
                      <X className="h-4 w-4" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={handleClose} disabled={uploading}>
            Cancel
          </Button>
          <Button
            onClick={handleUpload}
            disabled={files.length === 0 || uploading}
            className="min-w-[100px]"
          >
            {uploading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Uploading...
              </>
            ) : (
              'Upload'
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
