import { useCallback, useState } from "react";
import {
    Frame,
    FrameDescription,
    FrameFooter,
    FrameHeader,
    FramePanel,
    FrameTitle,
} from "@/components/ui/frame";
import { Badge } from "@renderer/components/ui/badge";
import { Button } from "@renderer/components/ui/button";
import { cn } from "@renderer/lib/utils";
import {
    AlertCircleIcon,
    Loader2Icon,
    PlayIcon,
    PlusIcon,
    RefreshCwIcon,
    SquareIcon,
    Trash2Icon,
} from "lucide-react";
import { useInstanceStore, type Instance } from "@/store/instance-store";
import { useNavigate } from "react-router";
import { toast } from 'sonner'
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogDescription,
    DialogFooter,
    DialogClose,
} from "@renderer/components/ui/dialog";
import { Input } from "@renderer/components/ui/input";
import {
    Empty,
    EmptyHeader,
    EmptyMedia,
    EmptyTitle,
    EmptyDescription,
    EmptyContent,
} from "@renderer/components/ui/empty";
import { Spinner } from "@renderer/components/ui/spinner";
import {
    AlertDialog,
    AlertDialogContent,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogCancel,
} from "@renderer/components/ui/alert-dialog";

const statusBadgeVariant = {
    running: "success",
    not_started: "secondary",
    errored: "error",
} as const;

const statusLabel = {
    running: "Running",
    not_started: "Stopped",
    errored: "Error",
} as const;

const InstancesPage = () => {
    const navigate = useNavigate();
    const {
        instances,
        loading,
        fetchError,
        fetchInstances,
        selectInstance,
        serveInstance,
        stopInstance,
        deleteInstance,
        lastFetchedAt,
    } = useInstanceStore();

    const [newPort, setNewPort] = useState("");
    const [createOpen, setCreateOpen] = useState(false);
    const [creating, setCreating] = useState(false);
    const [refreshing, setRefreshing] = useState(false);

    // Action loading states per instance
    const [actionLoading, setActionLoading] = useState<Record<string, string>>({});

    const handleInstanceClick = (instance: Instance) => {
        if (instance.status !== "running") return;
        selectInstance(instance.instanceId);
        navigate(`/instances/${instance.instanceId}/resources`);
    };

    const handleRefresh = useCallback(async () => {
        setRefreshing(true);
        await fetchInstances();
        setRefreshing(false);
    }, [fetchInstances]);

    const handleCreate = async () => {
        const port = parseInt(newPort, 10);
        if (isNaN(port) || port < 1 || port > 65535) {
            toast.error('Invalid port', {
                description: 'Port must be a number between 1 and 65535.',
            });
            return;
        }

        setCreating(true);
        const result = await serveInstance(port);
        setCreating(false);

        if (result.success) {
            toast.success('Instance started', {
                description: `MildStack instance started on port ${port}.`,
            });
            setCreateOpen(false);
            setNewPort("");
        } else {
            toast.error('Failed to start instance', {
                description: result.error || 'Unknown error',
            });
        }
    };

    const handleStop = async (e: React.MouseEvent, instance: Instance) => {
        e.stopPropagation();
        setActionLoading((prev) => ({ ...prev, [instance.instanceId]: "stopping" }));

        const result = await stopInstance(instance.port);

        setActionLoading((prev) => {
            const next = { ...prev };
            delete next[instance.instanceId];
            return next;
        });

        if (result.success) {
            toast.success('Instance stopped', {
                description: `Instance on port ${instance.port} has been stopped.`,
            });
        } else {
            toast.error('Failed to stop instance', {
                description: result.error || 'Unknown error',
            });
        }
    };

    const handleStart = async (e: React.MouseEvent, instance: Instance) => {
        e.stopPropagation();
        setActionLoading((prev) => ({ ...prev, [instance.instanceId]: "starting" }));

        const result = await serveInstance(instance.port);

        setActionLoading((prev) => {
            const next = { ...prev };
            delete next[instance.instanceId];
            return next;
        });

        if (result.success) {
            toast.success('Instance started', {
                description: `Instance on port ${instance.port} is now running.`,
            });
        } else {
            toast.error('Failed to start instance', {
                description: result.error || 'Unknown error',
            });
        }
    };

    const [deleteTarget, setDeleteTarget] = useState<Instance | null>(null);

    const handleDeleteConfirm = async () => {
        if (!deleteTarget) return;

        setActionLoading((prev) => ({ ...prev, [deleteTarget.instanceId]: "deleting" }));

        const result = await deleteInstance(deleteTarget.port);

        setActionLoading((prev) => {
            const next = { ...prev };
            delete next[deleteTarget.instanceId];
            return next;
        });

        if (result.success) {
            toast.success('Instance deleted', {
                description: `Instance on port ${deleteTarget.port} has been deleted.`,
            });
        } else {
            toast.error('Failed to delete instance', {
                description: result.error || 'Unknown error',
            });
        }

        setDeleteTarget(null);
    };

    const formatLastFetched = () => {
        if (!lastFetchedAt) return null;
        const diff = Math.floor((Date.now() - lastFetchedAt) / 1000);
        if (diff < 5) return "just now";
        if (diff < 60) return `${diff}s ago`;
        return `${Math.floor(diff / 60)}m ago`;
    };

    // Initial loading state (no instances loaded yet)
    if (loading && instances.length === 0 && !fetchError) {
        return (
            <Frame className="w-full">
                <div className="flex items-center justify-center py-20">
                    <Spinner className="h-6 w-6 text-muted-foreground" />
                </div>
            </Frame>
        );
    }

    return (
        <>
            <Frame className="w-full">
                <FrameHeader className="flex flex-row items-center justify-between">
                    <div>
                        <FrameTitle>MildStack Instances</FrameTitle>
                        <FrameDescription>
                            Manage your MildStack runtime instances
                            {lastFetchedAt && (
                                <span className="ml-2 text-xs text-muted-foreground/60">
                                    · Updated {formatLastFetched()}
                                </span>
                            )}
                        </FrameDescription>
                    </div>
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={handleRefresh}
                        disabled={refreshing}
                        className="shrink-0"
                    >
                        <RefreshCwIcon className={cn("h-4 w-4", refreshing && "animate-spin")} />
                    </Button>
                </FrameHeader>

                {fetchError && (
                    <div className="mx-4 mb-2 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
                        <AlertCircleIcon className="h-4 w-4 shrink-0" />
                        <span>{fetchError}</span>
                        <Button
                            variant="ghost"
                            size="sm"
                            className="ml-auto text-xs"
                            onClick={handleRefresh}
                        >
                            Retry
                        </Button>
                    </div>
                )}

                {instances.length === 0 && !loading ? (
                    <Empty className="py-16">
                        <EmptyHeader>
                            <EmptyMedia variant="icon">
                                <PlayIcon className="h-5 w-5" />
                            </EmptyMedia>
                            <EmptyTitle>No instances</EmptyTitle>
                            <EmptyDescription>
                                Start a new MildStack instance to begin working with your local AWS services.
                            </EmptyDescription>
                        </EmptyHeader>
                        <EmptyContent>
                            <Button onClick={() => setCreateOpen(true)}>
                                <PlusIcon /> New Instance
                            </Button>
                        </EmptyContent>
                    </Empty>
                ) : (
                    instances.map((instance) => {
                        const action = actionLoading[instance.instanceId];
                        const isLoading = !!action;

                        return (
                            <FramePanel
                                className={cn("cursor-pointer transition-colors", {
                                    "border-success/50 hover:border-success":
                                        instance.status === "running",
                                    "border-border hover:border-muted-foreground/30":
                                        instance.status === "not_started",
                                    "border-destructive/50 hover:border-destructive":
                                        instance.status === "errored",
                                    "pointer-events-none opacity-60": isLoading,
                                    "cursor-not-allowed opacity-70":
                                        instance.status !== "running" && !isLoading,
                                })}
                                key={instance.instanceId}
                                onClick={() => handleInstanceClick(instance)}
                            >
                                <div className="flex flex-row gap-2 w-full">
                                    <div className="flex flex-col gap-1.5 w-full">
                                        <div className="flex flex-row justify-between w-full gap-2 items-center">
                                            <div className="flex flex-row gap-2 items-center">
                                                <h2 className="font-semibold text-sm font-mono">
                                                    Instance {instance.port}
                                                </h2>
                                                <Badge className={cn({
                                                    'bg-green-500': instance.status === 'running',
                                                    'bg-red-500': instance.status === 'errored',
                                                })}>
                                                    {statusLabel[instance.status] ?? instance.status}
                                                </Badge>
                                                {instance.pid && (
                                                    <span className="text-xs text-muted-foreground font-mono">
                                                        PID {instance.pid}
                                                    </span>
                                                )}
                                            </div>
                                            <div className="flex flex-row gap-1 items-center">
                                                {isLoading ? (
                                                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground px-2">
                                                        <Loader2Icon className="h-3.5 w-3.5 animate-spin" />
                                                        <span className="capitalize">{action}…</span>
                                                    </div>
                                                ) : (
                                                    <>
                                                        {instance.status === "not_started" && (
                                                            <Button
                                                                variant="ghost"
                                                                size="icon"
                                                                onClick={(e) => handleStart(e, instance)}
                                                                title="Start instance"
                                                            >
                                                                <PlayIcon className="h-4 w-4" />
                                                            </Button>
                                                        )}
                                                        {instance.status === "running" && (
                                                            <Button
                                                                variant="ghost"
                                                                size="icon"
                                                                onClick={(e) => handleStop(e, instance)}
                                                                title="Stop instance"
                                                            >
                                                                <SquareIcon className="h-4 w-4" />
                                                            </Button>
                                                        )}
                                                        <Button
                                                            variant="ghost"
                                                            size="icon"
                                                            onClick={(e) => {
                                                                e.stopPropagation();
                                                                setDeleteTarget(instance);
                                                            }}
                                                            title="Delete instance"
                                                            className="text-muted-foreground hover:text-destructive"
                                                        >
                                                            <Trash2Icon className="h-4 w-4" />
                                                        </Button>
                                                    </>
                                                )}
                                            </div>
                                        </div>
                                        <div className="flex flex-col gap-0.5">
                                            <p className="text-sm text-muted-foreground">
                                                Endpoint:{" "}
                                                <span className="text-xs font-mono text-foreground/80">
                                                    http://localhost:{instance.port}
                                                </span>
                                            </p>
                                            {instance.error && (
                                                <p className="text-sm text-destructive flex items-center gap-1">
                                                    <AlertCircleIcon className="h-3.5 w-3.5 shrink-0" />
                                                    {instance.error}
                                                </p>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            </FramePanel>
                        );
                    })
                )}

                <FrameFooter>
                    <Button onClick={() => setCreateOpen(true)}>
                        <PlusIcon /> New Instance
                    </Button>
                </FrameFooter>
            </Frame>

            {/* Create Instance Dialog */}
            <Dialog open={createOpen} onOpenChange={setCreateOpen}>
                <DialogContent>
                    <DialogHeader>
                        <DialogTitle>New Instance</DialogTitle>
                        <DialogDescription>
                            Start a new MildStack instance on a specific port. The instance
                            will run in the background.
                        </DialogDescription>
                    </DialogHeader>
                    <div className="px-6 pb-4">
                        <label className="text-sm font-medium mb-1.5 block">Port</label>
                        <Input
                            type="number"
                            placeholder="4566"
                            value={newPort}
                            onChange={(e) => setNewPort((e.target as HTMLInputElement).value)}
                            onKeyDown={(e) => {
                                if (e.key === "Enter") handleCreate();
                            }}
                            min={1}
                            max={65535}
                        />
                    </div>
                    <DialogFooter>
                        <DialogClose asChild>
                            <Button variant="outline">
                                Cancel
                            </Button>
                        </DialogClose>
                        <Button onClick={handleCreate} disabled={creating || !newPort.trim()}>
                            {creating && <Loader2Icon className="h-4 w-4 animate-spin" />}
                            Start Instance
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* Delete Confirmation Dialog */}
            <AlertDialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>Delete Instance</AlertDialogTitle>
                        <AlertDialogDescription>
                            Are you sure you want to delete the instance on port{" "}
                            <strong>{deleteTarget?.port}</strong>? This action cannot be undone.
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel asChild>
                            <Button variant="outline">
                                Cancel
                            </Button>
                        </AlertDialogCancel>
                        <Button variant="destructive" onClick={handleDeleteConfirm}>
                            Delete
                        </Button>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    );
};

export default InstancesPage;