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
import { Separator } from "@renderer/components/ui/separator";
import { useInstanceStore } from "@/store/instance-store";
import { Badge } from "@renderer/components/ui/badge";
import { cn } from "@renderer/lib/utils";

const Header = () => {
    const currentPath = useLocation();
    const { getSelectedInstance } = useInstanceStore();
    const selectedInstance = getSelectedInstance();
    const segments = currentPath.pathname.split('/').filter(Boolean);

    const breadcrumbTree: Array<{ label: string; path: string }> = [
        {
            label: "Instances",
            path: "/",
        }
    ];

    segments.forEach((segment, index) => {
        if (index === 0 && segment.toLowerCase().startsWith("instances")) return;
        if (selectedInstance && segment === selectedInstance.instanceId) return;

        const url = `/${segments.slice(0, index + 1).join('/')}`;
        breadcrumbTree.push({
            label: segment.split('-').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' '),
            path: url,
        });
    });

    return (
        <header className="flex flex-row items-center gap-4 px-6">
            <Separator orientation="vertical" className="h-6" />
            {selectedInstance && (
                <div className="flex flex-row items-center gap-2">
                    <span className="text-sm font-semibold">Instance {selectedInstance.port}</span>
                    <Badge className={cn("text-[10px] h-4", {
                        'bg-green-500': selectedInstance.status === 'running',
                        'bg-red-500': selectedInstance.status === 'errored',
                    })}>
                        {selectedInstance.status}
                    </Badge>
                    <Separator orientation="vertical" className="h-6" />
                </div>
            )}
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
                                        <BreadcrumbLink asChild><Link to={item.path}>{item.label}</Link></BreadcrumbLink>
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