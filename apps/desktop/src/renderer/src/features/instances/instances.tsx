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
import { PauseIcon, PlayIcon, PlusIcon } from "lucide-react";
const instances = [
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
]

const badgesVariants = {
    running: "success",
    paused: "secondary",
    errored: "error",
}

import { useNavigate } from "react-router";

const InstancesPage = () => {
    const navigate = useNavigate();
    return (
        <>
            <Frame className="w-full">
                <FrameHeader>
                    <FrameTitle>MildStack Instances</FrameTitle>
                    <FrameDescription>Manage your MildStack instances</FrameDescription>
                </FrameHeader>
                {instances.map((instance) => (
                    <FramePanel 
                        className={cn("cursor-pointer", {
                            "border-success": instance.status === "running",
                            "border-border": instance.status === "paused",
                            "border-destructive": instance.status === "errored",
                        })} 
                        key={instance.id}
                        onClick={() => navigate(`/instances/${instance.id}/resources`)}
                    >
                        <div className="flex flex-row gap-2 w-full">
                            <div className="flex flex-col gap-1 w-full">
                                <div className="flex flex-row justify-between w-full gap-2 items-center">
                                    <div className="flex flex-row gap-2 items-center">
                                        <h2 className="font-semibold text-sm">{instance.name}</h2>
                                        <Badge variant={badgesVariants[instance.status]}>{instance.status}</Badge>
                                        <Badge variant="info">Port {instance.port}</Badge>
                                    </div>
                                    <div className="flex flex-row gap-2 items-center">
                                        <Button variant="ghost" size="icon" onClick={(e) => e.stopPropagation()}><PlayIcon /></Button>
                                        <Button variant="ghost" size="icon" onClick={(e) => e.stopPropagation()}><PauseIcon /></Button>
                                    </div>
                                </div>
                                <div className="flex flex-col">
                                    <p className="text-sm">Health: <span className="text-xs font-mono">{instance.health}</span></p>
                                    <p className="text-sm">Endpoint: <span className="text-xs font-mono">http://localhost:{instance.port}/_mildstack/</span></p>
                                </div>
                            </div>
                        </div>
                    </FramePanel>
                ))}
                <FrameFooter>
                    <Button>
                        <PlusIcon /> New Instance
                    </Button>
                </FrameFooter>
            </Frame>
        </>
    )
}

export default InstancesPage;