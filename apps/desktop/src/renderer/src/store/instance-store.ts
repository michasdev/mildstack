import { create } from 'zustand';

export type InstanceStatus = 'running' | 'not_started' | 'errored';

export interface Instance {
    instanceId: string;
    port: number;
    pid?: number;
    status: InstanceStatus;
    error?: string;
}

interface InstanceState {
    /** All instances from the CLI */
    instances: Instance[];
    /** Global runtime state from the CLI */
    runtimeState: string;
    /** Currently selected instance ID */
    selectedInstanceId: string | null;
    /** Whether we're currently fetching instances */
    loading: boolean;
    /** Last fetch error */
    fetchError: string | null;
    /** Timestamp of last successful fetch */
    lastFetchedAt: number | null;

    /** Fetch instances from CLI via IPC */
    fetchInstances: () => Promise<void>;
    /** Select an instance by ID */
    selectInstance: (id: string | null) => void;
    /** Get the currently selected instance */
    getSelectedInstance: () => Instance | undefined;
    /** Start a new instance on a port */
    serveInstance: (port: number) => Promise<{ success: boolean; error?: string }>;
    /** Stop an instance by port, or all */
    stopInstance: (port?: number, all?: boolean) => Promise<{ success: boolean; error?: string }>;
    /** Delete an instance by port, or all */
    deleteInstance: (port?: number, all?: boolean) => Promise<{ success: boolean; error?: string }>;
}

/** Polling interval ID for auto-refresh */
let pollingInterval: ReturnType<typeof setInterval> | null = null;

export const useInstanceStore = create<InstanceState>((set, get) => ({
    instances: [],
    runtimeState: 'not_started',
    selectedInstanceId: null,
    loading: false,
    fetchError: null,
    lastFetchedAt: null,

    fetchInstances: async () => {
        set({ loading: true, fetchError: null });
        try {
            const response = await window.api.mildstack.instances();
            set({
                instances: response.instances ?? [],
                runtimeState: response.state,
                loading: false,
                lastFetchedAt: Date.now(),
            });

            // If the selected instance no longer exists, deselect
            const { selectedInstanceId, instances } = get();
            if (selectedInstanceId && !instances.find(i => i.instanceId === selectedInstanceId)) {
                set({ selectedInstanceId: null });
            }
        } catch (err) {
            set({
                loading: false,
                fetchError: err instanceof Error ? err.message : 'Failed to fetch instances',
            });
        }
    },

    selectInstance: (id) => {
        set({ selectedInstanceId: id });
        const instance = get().instances.find((i) => i.instanceId === id);
        if (instance) {
            window.api.instance.setSelected(instance.port);
        }
    },

    getSelectedInstance: () => {
        const { instances, selectedInstanceId } = get();
        return instances.find((instance) => instance.instanceId === selectedInstanceId);
    },

    serveInstance: async (port: number) => {
        const result = await window.api.mildstack.start(port);
        if (result.success) {
            // Refresh instances after starting
            await get().fetchInstances();
        }
        return result;
    },

    stopInstance: async (port?: number, all?: boolean) => {
        const result = await window.api.mildstack.stop(port, all);
        // Always refresh after stop attempt
        await get().fetchInstances();
        return result;
    },

    deleteInstance: async (port?: number, all?: boolean) => {
        const result = await window.api.mildstack.delete(port, all);
        // Always refresh after delete attempt
        await get().fetchInstances();
        return result;
    },
}));

/**
 * Start auto-polling instances every 60 seconds.
 * Call this once on app mount.
 */
export function startInstancePolling(): void {
    if (pollingInterval) return;

    // Initial fetch
    useInstanceStore.getState().fetchInstances();

    pollingInterval = setInterval(() => {
        useInstanceStore.getState().fetchInstances();
    }, 60_000);
}

/**
 * Stop auto-polling instances.
 * Call this on app unmount.
 */
export function stopInstancePolling(): void {
    if (pollingInterval) {
        clearInterval(pollingInterval);
        pollingInterval = null;
    }
}
