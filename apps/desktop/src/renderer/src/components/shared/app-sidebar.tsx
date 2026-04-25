import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarGroup,
    SidebarGroupContent,
    SidebarGroupLabel,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
} from "@/components/ui/sidebar"
import { Link, useLocation } from "react-router"
import mildstackLightLogo from "@renderer/assets/logos/mildstack-logo-full-white.svg"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@renderer/components/ui/dropdown-menu"
import { 
    ChevronDown, 
    Home, 
    Database, 
    Inbox, 
    Bell, 
    Zap, 
    Settings2, 
    Network, 
    Archive,
    Settings
} from "lucide-react"
import { useInstanceStore } from "@/store/instance-store"

export function AppSidebar() {
    const location = useLocation()
    const { instances, selectInstance, getSelectedInstance } = useInstanceStore()
    const selectedInstance = getSelectedInstance()

    return (
        <Sidebar>
            <SidebarHeader className="">
                <Link to="/" className="mb-2 mt-2 mx-auto">
                    <img src={mildstackLightLogo} alt="Mildstack" className="h-8 w-auto pointer-events-none slect-none  " />
                </Link>
                <SidebarMenu>
                    <SidebarMenuItem>
                        <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                                <SidebarMenuButton 
                                    size="lg" 
                                    className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground border border-sidebar-border bg-sidebar-accent/30 hover:bg-sidebar-accent"
                                >
                                    {selectedInstance ? (
                                        <>
                                            <div className="flex h-4 w-4 items-center justify-center rounded-full bg-emerald-500/20">
                                                <div className={`h-2 w-2 rounded-full ${selectedInstance.status === 'running' ? 'bg-emerald-500' : selectedInstance.status === 'errored' ? 'bg-red-500' : 'bg-muted-foreground'}`} />
                                            </div>
                                            <div className="grid flex-1 text-left text-sm leading-tight">
                                                <span className="truncate font-semibold">:{selectedInstance.port}</span>
                                                <span className="truncate text-xs text-muted-foreground font-mono">localhost:{selectedInstance.port}</span>
                                            </div>
                                        </>
                                    ) : (
                                        <>
                                            <div className="flex h-4 w-4 items-center justify-center rounded-full bg-muted">
                                                <div className="h-2 w-2 rounded-full bg-muted-foreground/50" />
                                            </div>
                                            <div className="grid flex-1 text-left text-sm leading-tight">
                                                <span className="truncate font-semibold">No Instance</span>
                                                <span className="truncate text-xs text-muted-foreground">Select one...</span>
                                            </div>
                                        </>
                                    )}
                                    <ChevronDown className="ml-auto" />
                                </SidebarMenuButton>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg" align="start" side="bottom" sideOffset={4}>
                                {instances.map((instance) => (
                                    <DropdownMenuItem 
                                        key={instance.instanceId} 
                                        onClick={() => selectInstance(instance.instanceId)}
                                        className="flex items-center gap-2 cursor-pointer"
                                    >
                                        <div className={`h-2 w-2 rounded-full ${instance.status === 'running' ? 'bg-emerald-500' : instance.status === 'errored' ? 'bg-red-500' : 'bg-muted-foreground'}`} />
                                        <div className="flex flex-col">
                                            <span className="font-medium">:{instance.port}</span>
                                            <span className="text-xs text-muted-foreground">localhost:{instance.port}</span>
                                        </div>
                                    </DropdownMenuItem>
                                ))}
                                {instances.length === 0 && (
                                    <DropdownMenuItem disabled>
                                        No instances available
                                    </DropdownMenuItem>
                                )}
                            </DropdownMenuContent>
                        </DropdownMenu>
                    </SidebarMenuItem>
                </SidebarMenu>
            </SidebarHeader>
            
            <SidebarContent>
                <SidebarGroup>
                    <SidebarMenu>
                        <SidebarMenuItem>
                            <SidebarMenuButton asChild tooltip="Instances" isActive={location.pathname === '/'}>
                                <Link to="/">
                                    <Home />
                                    <span>Instances</span>
                                </Link>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                    </SidebarMenu>
                </SidebarGroup>

                <SidebarGroup>
                    <SidebarGroupLabel>SERVICES</SidebarGroupLabel>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            <SidebarMenuItem>
                                <SidebarMenuButton asChild tooltip="S3" isActive={location.pathname.startsWith('/resources/s3')}>
                                    <Link to="/resources/s3">
                                        <Archive />
                                        <span>S3</span>
                                    </Link>
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                            <SidebarMenuItem>
                                <SidebarMenuButton asChild tooltip="DynamoDB" isActive={location.pathname.startsWith('/resources/dynamodb')}>
                                    <Link to="/resources/dynamodb">
                                        <Database />
                                        <span>DynamoDB</span>
                                    </Link>
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                            <SidebarMenuItem>
                                <SidebarMenuButton asChild tooltip="SQS" isActive={location.pathname.startsWith('/resources/sqs')}>
                                    <Link to="/resources/sqs">
                                        <Inbox />
                                        <span>SQS</span>
                                    </Link>
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>

                <SidebarGroup>
                    <SidebarGroupLabel>COMING SOON</SidebarGroupLabel>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            <SidebarMenuItem>
                                <SidebarMenuButton disabled tooltip="Lambda">
                                    <Zap />
                                    <span>Lambda</span>
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                            <SidebarMenuItem>
                                <SidebarMenuButton disabled tooltip="SNS">
                                    <Bell />
                                    <span>SNS</span>
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                            <SidebarMenuItem>
                                <SidebarMenuButton disabled tooltip="SSM">
                                    <Settings2 />
                                    <span>SSM</span>
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                            <SidebarMenuItem>
                                <SidebarMenuButton disabled tooltip="EventBridge">
                                    <Network />
                                    <span>EventBridge</span>
                                </SidebarMenuButton>
                            </SidebarMenuItem>
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>
            </SidebarContent>

            <SidebarFooter className="p-4">
                <SidebarMenu>
                    <SidebarMenuItem>
                        <SidebarMenuButton tooltip="Settings">
                            <Settings />
                            <span>Settings</span>
                        </SidebarMenuButton>
                    </SidebarMenuItem>
                </SidebarMenu>
            </SidebarFooter>
        </Sidebar>
    )
}
