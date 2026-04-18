/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useEffect, useMemo, useState } from 'react'
import { Download, FileText, Film, Image as ImageIcon, X } from 'lucide-react'

import { Button } from '@renderer/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@renderer/components/ui/dialog'
import { Spinner } from '@renderer/components/ui/spinner'
import type { S3BrowserApi } from '../types'

interface ObjectViewerProps {
  api: S3BrowserApi
  bucketName: string
  objectKey: string
  region: string
  onClose: () => void
}

function base64ToUint8Array(base64: string): Uint8Array {
  const binary = window.atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return bytes
}

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

export function ObjectViewer({ api, bucketName, objectKey, region, onClose }: ObjectViewerProps) {
  const [contentUrl, setContentUrl] = useState<string | null>(null)
  const [binaryContent, setBinaryContent] = useState<Uint8Array | null>(null)
  const [textContent, setTextContent] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const isImage = useMemo(
    () => objectKey.match(/\.(jpeg|jpg|gif|png|webp|svg)$/i) !== null,
    [objectKey]
  )
  const isVideo = useMemo(() => objectKey.match(/\.(mp4|webm|ogg)$/i) !== null, [objectKey])
  const isText = useMemo(
    () =>
      objectKey.match(/\.(txt|md|json|csv|js|ts|tsx|jsx|html|css)$/i) !== null ||
      (!isImage && !isVideo),
    [objectKey, isImage, isVideo]
  )

  useEffect(() => {
    let objectUrl: string | null = null

    const fetchObject = async () => {
      try {
        setLoading(true)
        setError(null)
        setContentUrl(null)
        setBinaryContent(null)
        setTextContent(null)
        const response = await api.getObject(bucketName, objectKey, region)
        const bytes = base64ToUint8Array(response.contentBase64)
        const contentType = response.contentType || 'application/octet-stream'

        if (isImage || isVideo) {
          setBinaryContent(bytes)
          objectUrl = URL.createObjectURL(
            new Blob(
              [
                bytes.buffer.slice(
                  bytes.byteOffset,
                  bytes.byteOffset + bytes.byteLength
                ) as ArrayBuffer
              ],
              {
                type: contentType
              }
            )
          )
          setContentUrl(objectUrl)
        } else {
          setBinaryContent(null)
          setTextContent(new TextDecoder().decode(bytes))
        }
      } catch (err) {
        console.error('Error fetching object:', err)
        setError(err instanceof Error ? err.message : 'Failed to load object')
      } finally {
        setLoading(false)
      }
    }

    void fetchObject()

    return () => {
      if (objectUrl) {
        URL.revokeObjectURL(objectUrl)
      }
    }
  }, [api, bucketName, objectKey, region, isImage, isVideo])

  const handleDownload = () => {
    if (binaryContent) {
      const filename = objectKey.split('/').pop() || 'download'
      downloadBlob(
        new Blob([
          binaryContent.buffer.slice(
            binaryContent.byteOffset,
            binaryContent.byteOffset + binaryContent.byteLength
          ) as ArrayBuffer
        ]),
        filename
      )
      return
    }

    if (textContent !== null) {
      downloadBlob(
        new Blob([textContent], { type: 'text/plain' }),
        objectKey.split('/').pop() || 'download'
      )
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 truncate">
            {isImage ? (
              <ImageIcon className="h-5 w-5 text-info" />
            ) : isVideo ? (
              <Film className="h-5 w-5 text-secondary-foreground" />
            ) : (
              <FileText className="h-5 w-5 text-muted-foreground" />
            )}
            <span className="truncate">{objectKey}</span>
          </DialogTitle>
          <DialogDescription>Preview and download the selected object.</DialogDescription>
        </DialogHeader>

        <div className="min-h-[320px] rounded-xl border border-border bg-muted/30 p-4">
          {loading ? (
            <div className="flex min-h-[280px] items-center justify-center">
              <Spinner className="h-8 w-8 text-muted-foreground" />
            </div>
          ) : error ? (
            <div className="flex min-h-[280px] flex-col items-center justify-center gap-2 text-center text-destructive">
              <p className="font-medium">Error loading file</p>
              <p className="text-sm text-muted-foreground">{error}</p>
            </div>
          ) : isImage && contentUrl ? (
            <img src={contentUrl} alt={objectKey} className="max-h-[70vh] w-full object-contain" />
          ) : isVideo && contentUrl ? (
            <video src={contentUrl} controls autoPlay className="max-h-[70vh] w-full" />
          ) : isText && textContent !== null ? (
            <pre className="h-full overflow-auto whitespace-pre-wrap rounded-lg border border-border bg-background p-4 font-mono text-sm text-foreground">
              {textContent}
            </pre>
          ) : (
            <div className="flex min-h-[280px] items-center justify-center text-muted-foreground">
              Preview not available for this file type.
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={onClose}>
            <X className="h-4 w-4" />
            Close
          </Button>
          <Button onClick={handleDownload} disabled={loading || !!error}>
            <Download className="h-4 w-4" />
            Download
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
