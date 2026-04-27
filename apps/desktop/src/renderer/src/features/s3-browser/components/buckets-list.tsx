/* eslint-disable @typescript-eslint/explicit-function-return-type */
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router'
import { HardDrive, Plus, Search, Trash2 } from 'lucide-react'

import { Button } from '@renderer/components/ui/button'
import {
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardAction
} from '@renderer/components/ui/card'
import SpotlightCard from '@renderer/components/ui/spotlight-card'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@renderer/components/ui/dialog'
import { Spinner } from '@renderer/components/ui/spinner'
import { Badge } from '@renderer/components/ui/badge'
import { Input } from '@renderer/components/ui/input'
import { Checkbox } from '@renderer/components/ui/checkbox'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle
} from '@renderer/components/ui/empty'
import { cn } from '@renderer/lib/utils'
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel
} from '@renderer/components/ui/alert-dialog'
import type { S3Bucket } from '../types'
import type { S3BrowserOutletContext } from '../s3-layout'
import { toast } from 'sonner'

export function BucketsList() {
  const { api, region } = useOutletContext<S3BrowserOutletContext>()
  const navigate = useNavigate()
  const [buckets, setBuckets] = useState<S3Bucket[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [newBucketName, setNewBucketName] = useState('')
  const [isCreating, setIsCreating] = useState(false)
  const [selectedBuckets, setSelectedBuckets] = useState<Set<string>>(new Set())
  const [isDeleting, setIsDeleting] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const fetchBuckets = useCallback(async () => {
    try {
      setLoading(true)
      const response = await api.listBuckets(region)
      setBuckets(response)
      setSelectedBuckets(new Set())
    } catch (error) {
      console.error('Failed to fetch buckets:', error)
    } finally {
      setLoading(false)
    }
  }, [api, region])

  useEffect(() => {
    void fetchBuckets()
  }, [fetchBuckets])

  const filteredBuckets = useMemo(
    () =>
      buckets.filter((bucket) => bucket.Name?.toLowerCase().includes(searchQuery.toLowerCase())),
    [buckets, searchQuery]
  )

  const handleBucketClick = (bucketName: string) => {
    navigate(`/resources/s3/${bucketName}`)
  }

  const toggleBucketSelection = (bucketName: string) => {
    setSelectedBuckets((current) => {
      const next = new Set(current)
      if (next.has(bucketName)) {
        next.delete(bucketName)
      } else {
        next.add(bucketName)
      }
      return next
    })
  }

  const handleCreateBucket = async () => {
    const bucketName = newBucketName.trim()
    if (!bucketName) return

    setIsCreating(true)
    try {
      await api.createBucket(bucketName, region)
      setIsCreateOpen(false)
      setNewBucketName('')
      await fetchBuckets()
    } catch (err) {
      console.error('Failed to create bucket:', err)
      toast.error('Failed to create bucket', {
        description: err instanceof Error ? err.message : String(err)
      })
    } finally {
      setIsCreating(false)
    }
  }

  const executeBulkDelete = async () => {
    setShowDeleteConfirm(false)
    setIsDeleting(true)
    try {
      for (const bucket of Array.from(selectedBuckets)) {
        try {
          await api.deleteBucket(bucket, region)
        } catch (err) {
          console.error(`Failed to delete bucket ${bucket}:`, err)
          toast.error(`Failed to delete bucket ${bucket}`, {
            description: err instanceof Error ? err.message : String(err)
          })
        }
      }
      await fetchBuckets()
    } finally {
      setIsDeleting(false)
    }
  }

  const handleBulkDelete = () => {
    if (selectedBuckets.size === 0) return
    setShowDeleteConfirm(true)
  }

  const formatDate = (date?: string) => {
    if (!date) return 'Unknown'
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    }).format(new Date(date))
  }

  return (
    <div className="flex h-full flex-col gap-4 p-4">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="relative w-full max-w-md">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search buckets..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {selectedBuckets.size > 0 && (
            <Button variant="destructive" onClick={handleBulkDelete}>
              {isDeleting && <Spinner className="h-4 w-4" />}
              {!isDeleting && <Trash2 className="h-4 w-4" />}
              Delete ({selectedBuckets.size})
            </Button>
          )}

          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button variant="outline">
                <Plus className="h-4 w-4" />
                Create Bucket
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Bucket</DialogTitle>
                <DialogDescription>
                  Create a new S3 bucket in your local environment.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-2 px-6 pb-4">
                <label className="text-sm font-medium text-foreground" htmlFor="bucket-name">
                  Bucket name
                </label>
                <Input
                  id="bucket-name"
                  type="text"
                  placeholder="my-new-bucket"
                  value={newBucketName}
                  onChange={(e) => setNewBucketName(e.target.value)}
                  autoFocus
                />
              </div>
              <DialogFooter>
                <DialogClose asChild>
                  <Button variant="ghost">
                    Cancel
                  </Button>
                </DialogClose>
                <Button
                  onClick={handleCreateBucket}
                  disabled={isCreating}
                >
                  {isCreating && <Spinner className="h-4 w-4" />}
                  {!isCreating && <Plus className="h-4 w-4" />}
                  Create
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-hidden">
        {loading ? (
          <div className="flex h-40 items-center justify-center">
            <Spinner className="h-6 w-6 text-muted-foreground" />
          </div>
        ) : filteredBuckets.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <HardDrive className="h-6 w-6" />
              </EmptyMedia>
              <EmptyTitle>No buckets found</EmptyTitle>
              <EmptyDescription>
                {searchQuery
                  ? `No buckets match "${searchQuery}"`
                  : 'Create your first local S3 bucket to start uploading files.'}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className="grid grid-cols-1 gap-4 pb-6 md:grid-cols-2 xl:grid-cols-3">
            {filteredBuckets.map((bucket) => {
              const bucketName = bucket.Name ?? ''
              const selected = selectedBuckets.has(bucketName)

              return (
                <SpotlightCard
                  key={bucketName}
                  className={cn(
                    'cursor-pointer transition-all duration-200 hover:shadow-md',
                    selected && 'border-primary/40 bg-primary/5'
                  )}
                  onClick={() => handleBucketClick(bucketName)}
                >
                  <CardHeader className="gap-3">
                    <CardAction onClick={(e) => e.stopPropagation()}>
                      <Checkbox
                        checked={selected}
                        onCheckedChange={() => toggleBucketSelection(bucketName)}
                        aria-label={`Select ${bucketName}`}
                      />
                    </CardAction>
                    <div className="flex items-center gap-3">
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-primary/20 bg-primary/10 text-primary">
                        <HardDrive className="h-5 w-5" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <CardTitle className="truncate text-sm">{bucketName}</CardTitle>
                        <CardDescription className="mt-1 truncate">
                          Created {formatDate(bucket.CreationDate)}
                        </CardDescription>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="flex items-center justify-between gap-3 pt-0">
                    <Badge variant="outline">
                      S3 bucket
                    </Badge>
                    <span className="text-xs text-muted-foreground">Click to browse</span>
                  </CardContent>
                </SpotlightCard>
              )
            })}
          </div>
        )}
      </div>
      <AlertDialog open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Buckets</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete {selectedBuckets.size} bucket(s)? This action cannot
              be undone and the buckets must be empty.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel asChild>
              <Button variant="ghost">
                Cancel
              </Button>
            </AlertDialogCancel>
            <Button variant="destructive" onClick={executeBulkDelete}>
              Delete
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
