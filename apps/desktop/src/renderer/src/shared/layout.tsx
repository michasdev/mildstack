import { Header } from "@renderer/components/shared/header"
import { Outlet } from "react-router"
import { useEffect } from "react"
import { startInstancePolling, stopInstancePolling } from "@/store/instance-store"
import { SidebarProvider } from "@renderer/components/ui/sidebar"
import { AppSidebar } from "@renderer/components/shared/app-sidebar"

export const Layout = () => {
    useEffect(() => {
        startInstancePolling()
        return () => stopInstancePolling()
    }, [])

    return (
        <SidebarProvider>
            <AppSidebar />
            <div className="w-full">
                <div className="w-full mx-auto p-5 relative">
                    <Header />
                    <div className="mt-5">
                        <Outlet />
                    </div>
                </div>
            </div>
        </SidebarProvider>
    )
}