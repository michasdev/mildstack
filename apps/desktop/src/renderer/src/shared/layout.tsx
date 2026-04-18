import { Header } from "@renderer/components/shared/header"
import { Outlet } from "react-router"

export const Layout = () => {
    return (
        <div className="max-w-[1440px] mx-auto p-5 relative">
            <Header />
            <div className="mt-5">
                <Outlet />
            </div>
        </div>
    )
}