/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useCallback, useEffect, useMemo, useRef, useState, type DragEvent } from 'react'
import { useNavigate, useOutletContext, useParams } from 'react-router'
import {
  Check,
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Copy,
  Download,
  Eye,
  File as FileIcon,
  FileText,
  Film,
  Folder,
  Image as ImageIcon,
  Search,
  Trash2,
  Upload
} from 'lucide-react'
import { toastManager } from "@/components/ui/toast"
function base64ToUint8Array(base64: string): Uint8Array {
  const binary = window.atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return bytes
}

import { Button } from '@renderer/components/ui/button'
import { Spinner } from '@renderer/components/ui/spinner'
import { ScrollArea } from '@renderer/components/ui/scroll-area'
import { Input } from '@renderer/components/ui/input'
import { Checkbox } from '@renderer/components/ui/checkbox'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogClose
} from '@renderer/components/ui/alert-dialog'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle
} from '@renderer/components/ui/empty'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@renderer/components/ui/table'
import { cn } from '@renderer/lib/utils'
import type { S3Object } from '../types'
import type { S3BrowserOutletContext } from '../s3-layout'
import { ObjectViewer } from './object-viewer'
import { UploadDialog } from './upload-dialog'

export function ObjectList() {
  const { api, region } = useOutletContext<S3BrowserOutletContext>()
  const { bucketName, '*': wildcardPath } = useParams()
  const navigate = useNavigate()

  const currentPrefix = wildcardPath
    ? wildcardPath.endsWith('/')
      ? wildcardPath
      : `${wildcardPath}/`
    : ''

  const [objects, setObjects] = useState<S3Object[]>([])
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [continuationToken, setContinuationToken] = useState<string | undefined>()
  const [hasMore, setHasMore] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [viewingObject, setViewingObject] = useState<string | null>(null)
  const [isDragging, setIsDragging] = useState(false)
  const [selectedObjects, setSelectedObjects] = useState<Set<string>>(new Set())
  const [isDeleting, setIsDeleting] = useState(false)
  const [copiedKey, setCopiedKey] = useState<string | null>(null)
  const [downloadingKey, setDownloadingKey] = useState<string | null>(null)
  const [isUploadDialogOpen, setIsUploadDialogOpen] = useState(false)
  const [droppedFiles, setDroppedFiles] = useState<FileList | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  type SortKey = 'name' | 'size' | 'lastModified'
  type SortOrder = 'asc' | 'desc'

  const [sortKey, setSortKey] = useState<SortKey>('name')
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc')

  const observerTarget = useRef<HTMLDivElement>(null)

  const searchPrefix = useMemo(() => `${currentPrefix}${searchQuery}`, [currentPrefix, searchQuery])

  const fetchObjects = useCallback(
    async (token?: string, isLoadMore = false) => {
      if (!bucketName) return

      try {
        if (isLoadMore) {
          setLoadingMore(true)
        } else {
          setLoading(true)
        }

        const response = await api.listObjects(bucketName, searchPrefix, token, region)
        if (isLoadMore) {
          setObjects((prev) => [...prev, ...response.objects])
        } else {
          setObjects(response.objects)
          setSelectedObjects(new Set())
        }
        setHasMore(response.hasMore)
        setContinuationToken(response.continuationToken)
      } catch (error) {
        console.error('Failed to fetch objects:', error)
      } finally {
        setLoading(false)
        setLoadingMore(false)
      }
    },
    [api, bucketName, searchPrefix, region]
  )

  const sortedObjects = useMemo(() => {
    const sorted = [...objects]
    sorted.sort((a, b) => {
      // Folders always come first
      if (a.isFolder && !b.isFolder) return -1
      if (!a.isFolder && b.isFolder) return 1

      let aValue: any
      let bValue: any

      switch (sortKey) {
        case 'name':
          aValue = (a.isFolder ? (a.prefix ?? '') : (a.Key ?? '')).toLowerCase()
          bValue = (b.isFolder ? (b.prefix ?? '') : (b.Key ?? '')).toLowerCase()
          break
        case 'size':
          aValue = a.Size ?? 0
          bValue = b.Size ?? 0
          break
        case 'lastModified':
          aValue = a.LastModified ? new Date(a.LastModified).getTime() : 0
          bValue = b.LastModified ? new Date(b.LastModified).getTime() : 0
          break
        default:
          return 0
      }

      if (aValue < bValue) return sortOrder === 'asc' ? -1 : 1
      if (aValue > bValue) return sortOrder === 'asc' ? 1 : -1
      return 0
    })
    return sorted
  }, [objects, sortKey, sortOrder])

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortKey(key)
      setSortOrder('asc')
    }
  }

  const getSortIcon = (key: SortKey) => {
    if (sortKey !== key) return <ChevronsUpDown className="ml-1 h-3.5 w-3.5 opacity-50" />
    return sortOrder === 'asc' ? (
      <ChevronUp className="ml-1 h-3.5 w-3.5 text-primary" />
    ) : (
      <ChevronDown className="ml-1 h-3.5 w-3.5 text-primary" />
    )
  }

  useEffect(() => {
    void fetchObjects()
  }, [fetchObjects])

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loadingMore && !loading) {
          void fetchObjects(continuationToken, true)
        }
      },
      { threshold: 0.1 }
    )

    if (observerTarget.current) {
      observer.observe(observerTarget.current)
    }

    return () => observer.disconnect()
  }, [hasMore, loadingMore, loading, continuationToken, fetchObjects])

  const handleNavigate = (path: string) => {
    navigate(`/resources/s3/${bucketName}/${path}`)
  }

  const onDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const onDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(false)
  }

  const onDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(false)
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      setDroppedFiles(e.dataTransfer.files)
      setIsUploadDialogOpen(true)
    }
  }

  const toggleObjectSelection = (key: string) => {
    setSelectedObjects((current) => {
      const next = new Set(current)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  const executeBulkDelete = async () => {
    if (!bucketName || selectedObjects.size === 0) return
    const keysToDelete = Array.from(selectedObjects).filter(Boolean)

    setShowDeleteConfirm(false)
    setIsDeleting(true)

    try {
      const result = await api.deleteObjects(bucketName, keysToDelete, region)

      if (!result) {
        throw new Error('No response from delete operation')
      }

      const successCount = result.Deleted?.length || 0
      const errorCount = result.Errors?.length || 0

      if (successCount > 0) {
        toastManager.add({
          title: 'Objects deleted',
          description: `Successfully deleted ${successCount} object(s).`,
          type: 'success'
        })
      }

      if (errorCount > 0) {
        toastManager.add({
          title: 'Partial deletion failure',
          description: `Failed to delete ${errorCount} object(s).`,
          type: 'error'
        })
        console.error('Deletion errors:', result.Errors)
      }

      await fetchObjects()
      setSelectedObjects(new Set())
    } catch (err) {
      console.error('Bulk delete failed', err)
      toastManager.add({
        title: 'Deletion failed',
        type: 'error',
        description: err instanceof Error ? err.message : 'A network or system error occurred.'
      })
    } finally {
      setIsDeleting(false)
    }
  }

  const handleBulkDelete = () => {
    if (selectedObjects.size === 0 || !bucketName) return
    setShowDeleteConfirm(true)
  }

  const handleCopyUri = async (key: string) => {
    await navigator.clipboard.writeText(`s3://${bucketName}/${key}`)
    setCopiedKey(key)
    window.setTimeout(() => setCopiedKey(null), 2000)
  }

  const handleDownload = async (key: string) => {
    if (!bucketName) return
    setDownloadingKey(key)
    try {
      const response = await api.getObject(bucketName, key, region)
      const bytes = base64ToUint8Array(response.contentBase64)
      const blob = new Blob([bytes.buffer as ArrayBuffer], {
        type: response.contentType || 'application/octet-stream'
      })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = key.split('/').pop() || 'download'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
    } catch (err) {
      console.error('Download failed', err)
      toastManager.add({
        title: 'Failed to download file',
        type: 'error'
      })
    } finally {
      setDownloadingKey(null)
    }
  }

  const formatSize = (bytes?: number) => {
    if (bytes === undefined) return '-'
    if (bytes === 0) return '0 B'
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const index = Math.floor(Math.log(bytes) / Math.log(1024))
    return `${Number((bytes / 1024 ** index).toFixed(2))} ${sizes[index]}`
  }

  const getFileIcon = (key?: string) => {
    if (!key) return <FileIcon className="h-4 w-4 text-muted-foreground" />
    if (key.match(/\.(jpeg|jpg|gif|png|webp|svg)$/i))
      return <ImageIcon className="h-4 w-4 text-info" />
    if (key.match(/\.(mp4|webm|ogg)$/i))
      return <Film className="h-4 w-4 text-secondary-foreground" />
    if (key.match(/\.(txt|md|json|csv|js|ts|tsx|jsx|html|css)$/i))
      return <FileText className="h-4 w-4 text-muted-foreground" />
    return <FileIcon className="h-4 w-4 text-muted-foreground" />
  }

  return (
    <div
      className="relative flex h-full flex-col gap-4 p-4"
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    >
      {isDragging && (
        <div className="absolute inset-0 z-50 flex items-center justify-center rounded-2xl border border-dashed border-primary bg-primary/10 backdrop-blur-sm">
          <div className="flex items-center gap-3 rounded-2xl border border-border bg-background px-6 py-4 shadow-lg">
            <Upload className="h-6 w-6 text-primary" />
            <span className="text-lg font-medium">Drop files to start upload</span>
          </div>
        </div>
      )}

      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="relative w-full max-w-md">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search by prefix..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>

        <div className="flex flex-wrap gap-2">
          {selectedObjects.size > 0 && (
            <Button variant="destructive" onClick={handleBulkDelete} loading={isDeleting}>
              <Trash2 className="h-4 w-4" />
              Delete ({selectedObjects.size})
            </Button>
          )}
          <Button variant="outline" onClick={() => setIsUploadDialogOpen(true)}>
            <Upload className="h-4 w-4" />
            Upload
          </Button>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-hidden rounded-2xl border border-border bg-card shadow-xs/5">
        <ScrollArea className="h-full">
          <Table variant="card">
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">
                  <Checkbox
                    checked={objects.length > 0 && selectedObjects.size === objects.length}
                    onCheckedChange={(checked) => {
                      if (checked) {
                        const allKeys = sortedObjects
                          .map((object) => object.Key || object.prefix)
                          .filter((key): key is string => Boolean(key))
                        setSelectedObjects(new Set(allKeys))
                      } else {
                        setSelectedObjects(new Set())
                      }
                    }}
                    aria-label="Select all objects"
                  />
                </TableHead>
                <TableHead className="w-10" />
                <TableHead>
                  <button
                    type="button"
                    className="flex items-center hover:text-foreground transition-colors"
                    onClick={() => toggleSort('name')}
                  >
                    Name
                    {getSortIcon('name')}
                  </button>
                </TableHead>
                <TableHead className="text-right">
                  <button
                    type="button"
                    className="ml-auto flex items-center hover:text-foreground transition-colors"
                    onClick={() => toggleSort('size')}
                  >
                    Size
                    {getSortIcon('size')}
                  </button>
                </TableHead>
                <TableHead className="text-right">
                  <button
                    type="button"
                    className="ml-auto flex items-center hover:text-foreground transition-colors"
                    onClick={() => toggleSort('lastModified')}
                  >
                    Last Modified
                    {getSortIcon('lastModified')}
                  </button>
                </TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading && !loadingMore ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-12 text-center">
                    <Spinner className="mx-auto h-6 w-6 text-muted-foreground" />
                  </TableCell>
                </TableRow>
              ) : objects.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="py-12">
                    <Empty className="py-4">
                      <EmptyHeader>
                        <EmptyMedia variant="icon">
                          <Folder className="h-6 w-6" />
                        </EmptyMedia>
                        <EmptyTitle>Folder is empty</EmptyTitle>
                        <EmptyDescription>
                          {searchQuery
                            ? `No objects match "${searchQuery}"`
                            : 'Upload files or create folders to see them here.'}
                        </EmptyDescription>
                      </EmptyHeader>
                    </Empty>
                  </TableCell>
                </TableRow>
              ) : (
                sortedObjects.map((object) => {
                  const itemKey = object.Key || object.prefix || ''
                  const isSelected = selectedObjects.has(itemKey)
                  const label = object.isFolder
                    ? (object.prefix?.slice(currentPrefix.length) ?? '')
                    : (object.Key?.slice(currentPrefix.length) ?? '')

                  return (
                    <TableRow
                      key={itemKey}
                      className={cn(isSelected && 'bg-primary/5')}
                      data-state={isSelected ? 'selected' : undefined}
                    >
                      <TableCell>
                        <Checkbox
                          checked={isSelected}
                          onCheckedChange={() => toggleObjectSelection(itemKey)}
                          aria-label={`Select ${itemKey}`}
                        />
                      </TableCell>
                      <TableCell>
                        {object.isFolder ? (
                          <Folder className="h-4 w-4 text-primary" />
                        ) : (
                          getFileIcon(object.Key)
                        )}
                      </TableCell>
                      <TableCell className="max-w-0">
                        {object.isFolder ? (
                          <button
                            type="button"
                            className="max-w-full truncate text-left font-medium text-foreground hover:underline"
                            onClick={() => object.prefix && handleNavigate(object.prefix)}
                          >
                            {label}
                          </button>
                        ) : (
                          <div className="max-w-full truncate font-medium text-foreground">
                            {label}
                          </div>
                        )}
                      </TableCell>
                      <TableCell className="text-right text-muted-foreground">
                        {object.isFolder ? '-' : formatSize(object.Size)}
                      </TableCell>
                      <TableCell className="text-right text-muted-foreground">
                        {object.LastModified
                          ? new Date(object.LastModified).toLocaleString('en-US')
                          : '-'}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => void handleCopyUri(itemKey)}
                            title="Copy S3 URI"
                          >
                            {copiedKey === itemKey ? (
                              <Check className="h-4 w-4 text-success" />
                            ) : (
                              <Copy className="h-4 w-4" />
                            )}
                          </Button>
                          {!object.isFolder && (
                            <>
                              <Button
                                variant="ghost"
                                size="icon"
                                onClick={() => handleDownload(itemKey)}
                                loading={downloadingKey === itemKey}
                                title="Download"
                              >
                                <Download className="h-4 w-4" />
                              </Button>
                              {(() => {
                                const isImage = itemKey.match(/\.(jpeg|jpg|gif|png|webp|svg)$/i) !== null
                                const isVideo = itemKey.match(/\.(mp4|webm|ogg)$/i) !== null
                                const isText =
                                  itemKey.match(/\.(txt|md|json|csv|js|ts|tsx|jsx|html|css)$/i) !==
                                    null ||
                                  (!isImage && !isVideo)
                                const canView = !isText || (object.Size ?? 0) <= 15360

                                return (
                                  canView && (
                                    <Button
                                      variant="ghost"
                                      size="icon"
                                      onClick={() => object.Key && setViewingObject(object.Key)}
                                      title="View"
                                    >
                                      <Eye className="h-4 w-4" />
                                    </Button>
                                  )
                                )
                              })()}
                            </>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })
              )}
              {hasMore && (
                <TableRow>
                  <TableCell colSpan={6} className="py-0">
                    <div ref={observerTarget} className="h-10">
                      {loadingMore && (
                        <div className="flex h-full items-center justify-center">
                          <Spinner className="h-4 w-4 text-muted-foreground" />
                        </div>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </ScrollArea>
      </div>

      {viewingObject && bucketName && (
        <ObjectViewer
          api={api}
          bucketName={bucketName}
          objectKey={viewingObject}
          region={region}
          onClose={() => setViewingObject(null)}
        />
      )}

      {isUploadDialogOpen && bucketName && (
        <UploadDialog
          api={api}
          bucketName={bucketName}
          region={region}
          currentPrefix={currentPrefix}
          isOpen={isUploadDialogOpen}
          onClose={() => {
            setIsUploadDialogOpen(false)
            setDroppedFiles(null)
          }}
          onSuccess={() => void fetchObjects()}
          initialFiles={droppedFiles}
        />
      )}

      <AlertDialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Objects</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete {selectedObjects.size} item(s)? Folders must be empty
              to be deleted directly via S3 API. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose render={<Button variant="ghost" />}>Cancel</AlertDialogClose>
            <Button variant="destructive" onClick={executeBulkDelete} loading={isDeleting}>
              Delete
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
