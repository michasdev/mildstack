import {
    Breadcrumb,
    BreadcrumbItem,
    BreadcrumbLink,
    BreadcrumbList,
    BreadcrumbPage,
    BreadcrumbSeparator,
} from "@renderer/components/ui/breadcrumb";
import { Fragment } from "react";
import { Link, useLocation } from "react-router";

const Header = () => {
    const currentPath = useLocation();
    const segments = currentPath.pathname.split('/').filter(Boolean);

    const breadcrumbTree: Array<{ label: string; path: string }> = [
        {
            label: "Instances",
            path: "/",
        }
    ];

    segments.forEach((segment, index) => {
        if (index === 0 && segment.toLowerCase() === "instances") return;
        
        const url = `/${segments.slice(0, index + 1).join('/')}`;
        breadcrumbTree.push({
            label: segment.split('-').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' '),
            path: url,
        });
    });

    return (
        <header className="flex flex-row items-center gap-2">
            <h1 className="font-bold text-lg">MildStack</h1>
            <Breadcrumb>
                <BreadcrumbList>
                    {breadcrumbTree.map((item, index) => {
                        const isLast = index === breadcrumbTree.length - 1;
                        return (
                            <Fragment key={index}>
                                <BreadcrumbItem>
                                    {isLast ? (
                                        <BreadcrumbPage>{item.label}</BreadcrumbPage>
                                    ) : (
                                        <BreadcrumbLink render={<Link to={item.path} />}>{item.label}</BreadcrumbLink>
                                    )}
                                </BreadcrumbItem>
                                {!isLast && <BreadcrumbSeparator>/</BreadcrumbSeparator>}
                            </Fragment>
                        );
                    })}
                </BreadcrumbList>
            </Breadcrumb>
        </header>
    )
}

export { Header }