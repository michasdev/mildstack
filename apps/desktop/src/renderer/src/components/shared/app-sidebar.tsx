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
    Box,
    SquircleDashed,
    Settings
} from "lucide-react"
import { useInstanceStore } from "@/store/instance-store"
import { useState } from "react"
import { motion, AnimatePresence } from "motion/react"
import { cn } from "@renderer/lib/utils"

function NavItem({ 
    id, 
    icon: Icon, 
    label, 
    to, 
    isActive, 
    disabled = false,
    hoveredItem,
    setHoveredItem
}: { 
    id: string, 
    icon: React.ElementType, 
    label: string, 
    to?: string, 
    isActive?: boolean, 
    disabled?: boolean,
    hoveredItem: string | null,
    setHoveredItem: (id: string | null) => void
}) {
    return (
        <SidebarMenuItem 
            onMouseEnter={() => !disabled && setHoveredItem(id)} 
            onMouseLeave={() => setHoveredItem(null)}
            className="relative"
        >
            <AnimatePresence>
                {hoveredItem === id && !disabled && (
                    <motion.div
                        layoutId="sidebar-hover-bg"
                        className="absolute inset-0 bg-sidebar-accent rounded-md z-0"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 0.2 }}
                    />
                )}
            </AnimatePresence>
            <SidebarMenuButton 
                asChild={!disabled} 
                tooltip={label} 
                isActive={isActive} 
                disabled={disabled}
                className={cn("relative z-10 transition-colors", !disabled && "hover:bg-transparent")}
            >
                {disabled ? (
                    <>
                        <Icon />
                        <span>{label}</span>
                    </>
                ) : (
                    <Link to={to!}>
                        <Icon />
                        <span>{label}</span>
                    </Link>
                )}
            </SidebarMenuButton>
        </SidebarMenuItem>
    )
}

export function AppSidebar() {
    const location = useLocation()
    const { instances, selectInstance, getSelectedInstance } = useInstanceStore()
    const selectedInstance = getSelectedInstance()
    const isInstanceRunning = selectedInstance?.status === 'running'
    
    const [hoveredItem, setHoveredItem] = useState<string | null>(null)

    return (
        <Sidebar>
            <SidebarHeader className="">
                <Link to="/" className="mb-2 mt-2 mx-auto">
                    <img src={mildstackLightLogo} alt="Mildstack" className="h-8 w-auto pointer-events-none select-none" />
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
                        <NavItem 
                            id="home"
                            icon={Home}
                            label="Instances"
                            to="/"
                            isActive={location.pathname === '/'}
                            hoveredItem={hoveredItem}
                            setHoveredItem={setHoveredItem}
                        />
                    </SidebarMenu>
                </SidebarGroup>
                <SidebarGroup>
                    <SidebarGroupLabel className="text-xs">SERVICES</SidebarGroupLabel>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            <NavItem 
                                id="s3"
                                icon={Box}
                                label="S3"
                                to="/resources/s3"
                                isActive={location.pathname.startsWith('/resources/s3')}
                                disabled={!isInstanceRunning}
                                hoveredItem={hoveredItem}
                                setHoveredItem={setHoveredItem}
                            />
                            <NavItem 
                                id="dynamodb"
                                icon={Database}
                                label="DynamoDB"
                                to="/resources/dynamodb"
                                isActive={location.pathname.startsWith('/resources/dynamodb')}
                                disabled={!isInstanceRunning}
                                hoveredItem={hoveredItem}
                                setHoveredItem={setHoveredItem}
                            />
                            <NavItem 
                                id="sqs"
                                icon={Inbox}
                                label="SQS"
                                to="/resources/sqs"
                                isActive={location.pathname.startsWith('/resources/sqs')}
                                disabled={!isInstanceRunning}
                                hoveredItem={hoveredItem}
                                setHoveredItem={setHoveredItem}
                            />
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>

                <SidebarGroup>
                    <SidebarGroupLabel className="text-xs">COMING SOON</SidebarGroupLabel>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            <NavItem 
                                id="lambda"
                                icon={SquircleDashed}
                                label="Lambda"
                                disabled={true}
                                hoveredItem={hoveredItem}
                                setHoveredItem={setHoveredItem}
                            />
                            <NavItem 
                                id="sns"
                                icon={SquircleDashed}
                                label="SNS"
                                disabled={true}
                                hoveredItem={hoveredItem}
                                setHoveredItem={setHoveredItem}
                            />
                            <NavItem 
                                id="ssm"
                                icon={SquircleDashed}
                                label="SSM"
                                disabled={true}
                                hoveredItem={hoveredItem}
                                setHoveredItem={setHoveredItem}
                            />
                            <NavItem 
                                id="eventbridge"
                                icon={SquircleDashed}
                                label="EventBridge"
                                disabled={true}
                                hoveredItem={hoveredItem}
                                setHoveredItem={setHoveredItem}
                            />
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>
            </SidebarContent>

            <SidebarFooter className="p-4">
                <SidebarMenu>
                    <NavItem 
                        id="settings"
                        icon={Settings}
                        label="Settings"
                        to="/settings"
                        hoveredItem={hoveredItem}
                        setHoveredItem={setHoveredItem}
                    />
                </SidebarMenu>
            </SidebarFooter>
        </Sidebar>
    )
}
