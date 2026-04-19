import { create } from 'zustand';

export interface Instance {
    id: string;
    name: string;
    status: 'running' | 'paused' | 'errored';
    health: string;
    port: number;
}

interface InstanceState {
    instances: Instance[];
    selectedInstanceId: string | null;
    setInstances: (instances: Instance[]) => void;
    selectInstance: (id: string | null) => void;
    getSelectedInstance: () => Instance | undefined;
}

export const useInstanceStore = create<InstanceState>((set, get) => ({
    instances: [
        {
            id: "instance-1",
            name: "Instance #1",
            status: "running",
            health: "All systems operational",
            port: 4566,
        },
        {
            id: "instance-2",
            name: "Instance #2",
            status: "errored",
            health: "Error: Failed to start",
            port: 3566,
        },
        {
            id: "instance-3",
            name: "Instance #3",
            status: "paused",
            health: "Not running.",
            port: 2566,
        }
    ],
    selectedInstanceId: null,
    setInstances: (instances) => set({ instances }),
    selectInstance: (id) => {
        set({ selectedInstanceId: id });
        const instance = get().instances.find((i) => i.id === id);
        if (instance) {
            window.api.instance.setSelected(instance.port);
        }
    },
    getSelectedInstance: () => {
        const { instances, selectedInstanceId } = get();
        return instances.find((instance) => instance.id === selectedInstanceId);
    },
}));
