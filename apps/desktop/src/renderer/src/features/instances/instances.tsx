import {
    Frame,
    FrameDescription,
    FrameFooter,
    FrameHeader,
    FramePanel,
    FrameTitle,
} from "@/components/ui/frame";

const InstancesPage = () => {
    return (
        <>
            <Frame className="w-full">
                <FrameHeader>
                    <FrameTitle>Instances</FrameTitle>
                    <FrameDescription>Manage your MildStack instances</FrameDescription>
                </FrameHeader>
                <FramePanel>
                    <h2 className="font-semibold text-sm">Instance #1</h2>
                    <p className="text-muted-foreground text-sm">Running</p>
                </FramePanel>
                <FrameFooter>
                    <p className="text-muted-foreground text-sm">Footer</p>
                </FrameFooter>
            </Frame>
        </>
    )
}

export default InstancesPage;