import { Header } from "@renderer/components/shared/header"
import { Outlet } from "react-router"

export const Layout = () => {
    return (
        <div className="p-5">
            <Header />
            <div className="mt-5">
                <Outlet />
            </div>
        </div>
    )
}