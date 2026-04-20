import { Header } from "@renderer/components/shared/header"
import { Outlet } from "react-router"
import { useEffect } from "react"
import { startInstancePolling, stopInstancePolling } from "@/store/instance-store"

export const Layout = () => {
    useEffect(() => {
        startInstancePolling()
        return () => stopInstancePolling()
    }, [])

    return (
        <div className="max-w-[1440px] mx-auto p-5 relative">
            <Header />
            <div className="mt-5">
                <Outlet />
            </div>
        </div>
    )
}