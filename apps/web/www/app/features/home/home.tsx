import { BGPattern } from "@/components/shared/bg-pattern"
import { CallToAction } from "./components/call-to-action"
import { TryApp } from "./sections/try-app/try-app"
import { GetStartedSection } from "./sections/get-started/get-started"
import { HeroSection } from "./sections/hero/hero"
import { Footer } from "@/components/shared/footer"
import { FAQSection } from "./sections/faq/faq"
import { Cloud, Database, HardDrive, LayoutGrid } from "lucide-react"

const SUPPORTED_SERVICES = [
    {
        name: "S3",
        provider: "Amazon S3",
        Icon: HardDrive,
    },
    {
        name: "SQS",
        provider: "Amazon SQS",
        Icon: LayoutGrid,
    },
    {
        name: "SNS",
        provider: "Amazon SNS",
        Icon: Cloud,
    },
    {
        name: "DynamoDB",
        provider: "Amazon DynamoDB",
        Icon: Database,
    },
]

const HomePage = () => {
    return (
        <>
            <div id="home">
                <HeroSection />
            </div>
            <div className="w-full bg-primary">
                <div className="mx-auto w-full max-w-7xl px-4 py-4 sm:px-6 md:py-5">
                    <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between md:gap-8">
                        <div className="flex items-end justify-between gap-4 md:block">
                            <div className="flex flex-col gap-1">
                                <p className="text-xs font-semibold tracking-[0.2em] text-white/80 uppercase whitespace-nowrap">
                                    Supported Services
                                </p>
                                <p className="text-[11px] font-medium text-white/65 whitespace-nowrap">More services soon</p>
                            </div>
                            <p className="text-sm font-medium text-white/75 whitespace-nowrap md:hidden">100% API compatible</p>
                        </div>
                        <div className="grid grid-cols-2 gap-2 sm:gap-2.5 md:flex md:flex-1 md:items-center md:justify-center md:gap-x-6 md:gap-y-3">
                            {SUPPORTED_SERVICES.map(({ name, provider, Icon }, index) => (
                                <div
                                    key={name}
                                    className={`flex items-center gap-2 rounded-lg bg-white/10 px-2.5 py-2 md:rounded-none md:bg-transparent md:p-0 ${
                                        index > 0 ? "md:pl-6 md:border-l md:border-white/25" : ""
                                    }`}
                                >
                                    <Icon className="size-4 text-white/90" />
                                    <div className="flex flex-col leading-tight">
                                        <span className="text-sm font-semibold text-white md:text-base">{name}</span>
                                        <span className="hidden text-xs text-white/80 md:block">{provider}</span>
                                    </div>
                                </div>
                            ))}
                        </div>
                        <p className="hidden text-sm font-medium text-white/75 whitespace-nowrap md:block">100% API compatible</p>
                    </div>
                </div>
            </div>
            <div className="relative isolate w-full h-full max-w-7xl mx-auto pt-12">
                <BGPattern variant="dots" mask="fade-x" />
                <div id="get-started">
                    <GetStartedSection />
                </div>
                <div id="desktop-app" className="bg-background px-2">
                    <div className="rounded-3xl transition-colors border border-purple-500/20 bg-gradient-to-br from-purple-600/10 to-transparent">
                        <TryApp />
                    </div>
                </div>
                <div id="faq">
                    <FAQSection />
                </div>
            </div>
            <div className="w-full max-w-7xl mx-auto pt-40 pb-40 px-2">
                <CallToAction />
            </div>
            <Footer />
        </>
    )
}

export { HomePage }
