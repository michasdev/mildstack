import { useCallback, useEffect, useMemo, useRef, useState, type DragEvent } from 'react'
import { useNavigate, useOutletContext, useParams } from 'react-router'
import { toast } from "sonner"
import type { S3Object } from '../types'
import type { S3BrowserOutletContext } from '../s3-layout'

export type SortKey = 'name' | 'size' | 'lastModified'
export type SortOrder = 'asc' | 'desc'

export function useObjectList() {
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
  const [isCreateFolderDialogOpen, setIsCreateFolderDialogOpen] = useState(false)
  const [newFolderName, setNewFolderName] = useState('')
  const [isCreatingFolder, setIsCreatingFolder] = useState(false)
  const [droppedFiles, setDroppedFiles] = useState<FileList | null>(null)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

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
        toast.success('Objects deleted', {
          description: `Successfully deleted ${successCount} object(s).`,
        })
      }

      if (errorCount > 0) {
        toast.error('Partial deletion failure', {
          description: `Failed to delete ${errorCount} object(s).`,
        })
        console.error('Deletion errors:', result.Errors)
      }

      await fetchObjects()
      setSelectedObjects(new Set())
    } catch (err) {
      console.error('Bulk delete failed', err)
      toast.error('Deletion failed', {
        description: err instanceof Error ? err.message : 'A network or system error occurred.'
      })
    } finally {
      setIsDeleting(false)
    }
  }

  const handleCreateFolder = async () => {
    if (!bucketName || !newFolderName.trim()) return

    setIsCreatingFolder(true)
    try {
      const folderKey = currentPrefix + newFolderName.trim() + '/'
      await api.putObject(
        bucketName,
        folderKey,
        new Uint8Array().buffer,
        'application/x-directory',
        region
      )
      toast.success('Folder created', {
        description: `Successfully created folder "${newFolderName.trim()}"`,
      })
      setNewFolderName('')
      setIsCreateFolderDialogOpen(false)
      await fetchObjects()
    } catch (err) {
      console.error('Failed to create folder', err)
      toast.error('Failed to create folder', {
        description: err instanceof Error ? err.message : 'Unknown error'
      })
    } finally {
      setIsCreatingFolder(false)
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
      toast.error('Failed to download file', {
        description: err instanceof Error ? err.message : 'Unknown error'
      })
    } finally {
      setDownloadingKey(null)
    }
  }

  function base64ToUint8Array(base64: string): Uint8Array {
    const binary = window.atob(base64)
    const bytes = new Uint8Array(binary.length)
    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index)
    }
    return bytes
  }

  return {
    api,
    region,
    bucketName,
    currentPrefix,
    objects,
    loading,
    loadingMore,
    hasMore,
    searchQuery,
    setSearchQuery,
    viewingObject,
    setViewingObject,
    isDragging,
    selectedObjects,
    setSelectedObjects,
    isDeleting,
    copiedKey,
    downloadingKey,
    isUploadDialogOpen,
    setIsUploadDialogOpen,
    isCreateFolderDialogOpen,
    setIsCreateFolderDialogOpen,
    newFolderName,
    setNewFolderName,
    isCreatingFolder,
    droppedFiles,
    setDroppedFiles,
    showDeleteConfirm,
    setShowDeleteConfirm,
    sortKey,
    sortOrder,
    observerTarget,
    sortedObjects,
    toggleSort,
    handleNavigate,
    onDragOver,
    onDragLeave,
    onDrop,
    toggleObjectSelection,
    executeBulkDelete,
    handleCreateFolder,
    handleBulkDelete,
    handleCopyUri,
    handleDownload,
    fetchObjects
  }
}
