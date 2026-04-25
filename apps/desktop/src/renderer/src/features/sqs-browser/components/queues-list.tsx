/* eslint-disable @typescript-eslint/explicit-function-return-type */

import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useOutletContext } from 'react-router'
import { MessageSquare, Plus, Search, Trash2, Zap } from 'lucide-react'

import { Button } from '@renderer/components/ui/button'
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
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel
} from '@renderer/components/ui/alert-dialog'
import {
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardAction
} from '@renderer/components/ui/card'
import { cn } from '@renderer/lib/utils'
import { toast } from 'sonner' 
import type { SQSQueueSummary } from '../types'
import type { SQSBrowserOutletContext } from '../sqs-layout'

export function QueuesList() {
  const { api, region } = useOutletContext<SQSBrowserOutletContext>()
  const navigate = useNavigate()
  const [queues, setQueues] = useState<SQSQueueSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [newQueueName, setNewQueueName] = useState('')
  const [isFifo, setIsFifo] = useState(false)
  const [isCreating, setIsCreating] = useState(false)
  const [selectedQueues, setSelectedQueues] = useState<Set<string>>(new Set())
  const [isDeleting, setIsDeleting] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const fetchQueues = useCallback(async () => {
    try {
      setLoading(true)
      const summaries = await api.listQueues(region)
      setQueues(summaries)
      setSelectedQueues(new Set())
    } catch (error) {
      console.error('Failed to fetch queues:', error)
      toast.error('Failed to fetch queues', {
        description: error instanceof Error ? error.message : String(error)
      })
    } finally {
      setLoading(false)
    }
  }, [api, region])

  useEffect(() => {
    void fetchQueues()
  }, [fetchQueues])

  const filteredQueues = useMemo(
    () =>
      queues.filter((queue) =>
        queue.QueueName.toLowerCase().includes(searchQuery.toLowerCase())
      ),
    [queues, searchQuery]
  )

  const handleQueueClick = (queueName: string) => {
    navigate(`/resources/sqs/${encodeURIComponent(queueName)}`)
  }

  const toggleQueueSelection = (queueUrl: string) => {
    setSelectedQueues((current) => {
      const next = new Set(current)
      if (next.has(queueUrl)) {
        next.delete(queueUrl)
      } else {
        next.add(queueUrl)
      }
      return next
    })
  }

  const handleCreateQueue = async () => {
    const name = newQueueName.trim()
    if (!name) return

    setIsCreating(true)
    try {
      await api.createQueue(name, isFifo, region)
      setIsCreateOpen(false)
      resetCreateForm()
      await fetchQueues()
      toast.success('Queue created', {
        description: `Queue "${name}" created successfully.`
      })
    } catch (err) {
      console.error('Failed to create queue:', err)
      toast.error('Failed to create queue', {
        description: err instanceof Error ? err.message : String(err)
      })
    } finally {
      setIsCreating(false)
    }
  }

  const resetCreateForm = () => {
    setNewQueueName('')
    setIsFifo(false)
  }

  const executeBulkDelete = async () => {
    setShowDeleteConfirm(false)
    setIsDeleting(true)
    try {
      for (const queueUrl of Array.from(selectedQueues)) {
        try {
          await api.deleteQueue(queueUrl, region)
        } catch (err) {
          console.error(`Failed to delete queue ${queueUrl}:`, err)
          toast.error('Failed to delete queue', {
            description: err instanceof Error ? err.message : String(err)
          })
        }
      }
      await fetchQueues()
    } finally {
      setIsDeleting(false)
    }
  }

  const handleBulkDelete = () => {
    if (selectedQueues.size === 0) return
    setShowDeleteConfirm(true)
  }

  const handlePurge = async (e: React.MouseEvent, queueUrl: string) => {
    e.stopPropagation()
    try {
      await api.purgeQueue(queueUrl, region)
      toast.success('Queue purged', {
        description: 'All messages have been cleared from the queue.'
      })
      await fetchQueues()
    } catch (err) {
      toast.error('Failed to purge queue', {
        description: err instanceof Error ? err.message : String(err)
      })
    }
  }

  return (
    <div className="flex h-full flex-col gap-4 p-4">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div className="relative w-full max-w-md">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search queues..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {selectedQueues.size > 0 && (
            <Button variant="destructive" onClick={handleBulkDelete} disabled={isDeleting}>
              {isDeleting ? <Spinner /> : (
                <Trash2 className="h-4 w-4" />
              )}
              Delete ({selectedQueues.size})
            </Button>
          )}

          <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
            <DialogTrigger asChild>
              <Button variant="outline">
                <Plus className="h-4 w-4" />
                Create Queue
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create Queue</DialogTitle>
                <DialogDescription>
                  Create a new SQS queue in your local environment.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 px-6 pb-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground" htmlFor="queue-name">
                    Queue name
                  </label>
                  <div className="flex items-center gap-2">
                    <Input
                      id="queue-name"
                      type="text"
                      placeholder="my-queue"
                      value={newQueueName}
                      onChange={(e) => setNewQueueName(e.target.value)}
                      autoFocus
                    />
                    {isFifo && (
                      <span className="text-sm text-muted-foreground shrink-0 border border-border px-2 py-1 rounded bg-secondary/50">
                        .fifo
                      </span>
                    )}
                  </div>
                </div>

                <div className="flex items-center space-x-2">
                  <Checkbox
                    id="is-fifo"
                    checked={isFifo}
                    onCheckedChange={(checked) => setIsFifo(!!checked)}
                  />
                  <label
                    htmlFor="is-fifo"
                    className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 text-foreground"
                  >
                    FIFO Queue
                  </label>
                </div>
                <p className="text-xs text-muted-foreground">
                  First-In-First-Out queues maintain the strict order of messages and provide exactly-once processing.
                </p>
              </div>
              <DialogFooter>
                <DialogClose asChild>
                  <Button variant="ghost">Cancel</Button>
                </DialogClose>
                <Button
                  onClick={handleCreateQueue}
                  disabled={!newQueueName.trim() || isCreating}
                >
                  {isCreating ? <Spinner /> : (
                    <Plus className="h-4 w-4" />
                  )}
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
        ) : filteredQueues.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <MessageSquare className="h-6 w-6" />
              </EmptyMedia>
              <EmptyTitle>No queues found</EmptyTitle>
              <EmptyDescription>
                {searchQuery
                  ? `No queues match "${searchQuery}"`
                  : 'Create your first SQS queue to start sending messages.'}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className="grid grid-cols-1 gap-4 pb-6 md:grid-cols-2 xl:grid-cols-3">
            {filteredQueues.map((queue) => {
              const selected = selectedQueues.has(queue.QueueUrl)

              return (
                <SpotlightCard
                  key={queue.QueueUrl}
                  className={cn(
                    'cursor-pointer transition-all duration-200 hover:shadow-md flex flex-col',
                    selected && 'border-primary/40 bg-primary/5'
                  )}
                  onClick={() => handleQueueClick(queue.QueueName)}
                >
                  <CardHeader className="gap-3">
                    <CardAction onClick={(e) => e.stopPropagation()}>
                      <Checkbox
                        checked={selected}
                        onCheckedChange={() => toggleQueueSelection(queue.QueueUrl)}
                        aria-label={`Select ${queue.QueueName}`}
                      />
                    </CardAction>
                    <div className="flex items-center gap-3 w-full min-w-0 pr-8">
                      <div className={cn(
                        "flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border",
                        queue.IsFifo
                          ? "border-amber-500/20 bg-amber-500/10 text-amber-500"
                          : "border-primary/20 bg-primary/10 text-primary"
                      )}>
                        <MessageSquare className="h-5 w-5" />
                      </div>
                      <div className="min-w-0 flex-1">
                        <CardTitle className="truncate text-sm pr-2 flex items-center gap-2">
                          <span className="truncate">{queue.QueueName}</span>
                          {queue.IsFifo && <Badge variant="outline" className="border-amber-500/30 text-amber-500 bg-amber-500/10 uppercase text-[10px] h-5 px-1.5 shrink-0">FIFO</Badge>}
                        </CardTitle>
                        <CardDescription className="mt-1 truncate flex items-center gap-3">
                          <span className="flex items-center gap-1 text-xs" title="Available Messages">
                            <span className="font-medium text-foreground">{queue.MessagesAvailable}</span>
                            <span className="text-muted-foreground">avail</span>
                          </span>
                          <span className="flex items-center gap-1 text-xs" title="Messages in Flight (Invisible)">
                            <span className="font-medium text-foreground">{queue.MessagesInvisible}</span>
                            <span className="text-muted-foreground">flight</span>
                          </span>
                        </CardDescription>
                      </div>
                    </div>
                  </CardHeader>
                  <CardContent className="flex items-center justify-between gap-3 pt-0 mt-auto">
                    <div className="flex items-center gap-2">
                      <Button variant="ghost" size="icon-sm" onClick={(e) => handlePurge(e, queue.QueueUrl)} title="Purge Queue">
                        <Zap className="h-4 w-4 text-muted-foreground hover:text-foreground" />
                      </Button>
                    </div>
                    <span className="text-xs text-muted-foreground">Click to view messages</span>
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
            <AlertDialogTitle>Delete Queues</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete {selectedQueues.size} queue(s)? This action cannot
              be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel asChild>
              <Button variant="ghost">Cancel</Button>
            </AlertDialogCancel>
            <Button variant="destructive" onClick={executeBulkDelete} disabled={isDeleting}>
              {isDeleting ? <Spinner /> : (
                <Trash2 className="h-4 w-4" />
              )}
              Delete
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
