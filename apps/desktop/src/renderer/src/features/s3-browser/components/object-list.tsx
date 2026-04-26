/* eslint-disable @typescript-eslint/explicit-function-return-type */
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
  FolderPlus,
  Image as ImageIcon,
  Search,
  Trash2,
  Upload
} from 'lucide-react'

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
  AlertDialogCancel
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
import { ObjectViewer } from './object-viewer'
import { UploadDialog } from './upload-dialog'
import { Dialog, DialogClose, DialogDescription, DialogFooter, DialogHeader, DialogContent, DialogTitle } from '@renderer/components/ui/dialog'
import { useObjectList, type SortKey } from '../hooks/use-object-list'

export function ObjectList() {
  const {
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
  } = useObjectList()

  const getSortIcon = (key: SortKey) => {
    if (sortKey !== key) return <ChevronsUpDown className="ml-1 h-3.5 w-3.5 opacity-50" />
    return sortOrder === 'asc' ? (
      <ChevronUp className="ml-1 h-3.5 w-3.5 text-primary" />
    ) : (
      <ChevronDown className="ml-1 h-3.5 w-3.5 text-primary" />
    )
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
            <Button variant="destructive" onClick={handleBulkDelete} disabled={isDeleting}>
              {isDeleting ? <Spinner /> : (
                <Trash2 className="h-4 w-4" />
              )}
              Delete ({selectedObjects.size})
            </Button>
          )}
          <Button variant="outline" onClick={() => setIsCreateFolderDialogOpen(true)}>
            <FolderPlus className="h-4 w-4" />
            New Folder
          </Button>
          <Button variant="outline" onClick={() => setIsUploadDialogOpen(true)}>
            <Upload className="h-4 w-4" />
            Upload
          </Button>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-hidden rounded-2xl border border-border bg-card shadow-xs/5">
        <ScrollArea className="h-full">
          <Table>
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
                                disabled={downloadingKey === itemKey}
                                title="Download"
                              >
                                {downloadingKey === itemKey ? <Spinner /> : (
                                  <Download className="h-4 w-4" />
                                )}
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

      <Dialog open={isCreateFolderDialogOpen} onOpenChange={setIsCreateFolderDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Folder</DialogTitle>
            <DialogDescription>
              Enter a name for the new folder in the current directory.
            </DialogDescription>
          </DialogHeader>
          <DialogContent className="py-4">
            <Input
              autoFocus
              placeholder="Folder name..."
              value={newFolderName}
              onChange={(e) => setNewFolderName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && newFolderName.trim()) {
                  void handleCreateFolder()
                }
              }}
            />
          </DialogContent>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="ghost">Cancel</Button>
            </DialogClose>
            <Button
              onClick={handleCreateFolder}
              disabled={!newFolderName.trim() || isCreatingFolder}
            >
              {isCreatingFolder ? <Spinner /> : (
                <FolderPlus className="h-4 w-4" />
              )}
              Create Folder
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
