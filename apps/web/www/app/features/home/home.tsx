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
                <div className="mx-auto flex w-full max-w-7xl items-center justify-between gap-8 px-6 py-5 max-lg:flex-wrap">
                    <div className="flex flex-col gap-1">
                        <p className="text-xs font-semibold tracking-[0.2em] text-white/80 uppercase whitespace-nowrap">
                            Supported Services
                        </p>
                        <p className="text-[11px] font-medium text-white/65 whitespace-nowrap">More services soon</p>
                    </div>
                    <div className="flex flex-1 flex-wrap items-center justify-center gap-x-6 gap-y-3 max-lg:justify-start">
                        {SUPPORTED_SERVICES.map(({ name, provider, Icon }, index) => (
                            <div
                                key={name}
                                className={`flex items-center gap-2.5 ${
                                    index > 0 ? "pl-6 max-md:pl-0 md:border-l md:border-white/25" : ""
                                }`}
                            >
                                <Icon className="size-4 text-white/90" />
                                <div className="flex flex-col leading-tight">
                                    <span className="text-base font-semibold text-white">{name}</span>
                                    <span className="text-xs text-white/80">{provider}</span>
                                </div>
                            </div>
                        ))}
                    </div>
                    <p className="text-sm font-medium text-white/75 whitespace-nowrap">100% API compatible</p>
                </div>
            </div>
            <div className="relative isolate w-full h-full max-w-7xl mx-auto pt-12">
                <BGPattern variant="dots" mask="fade-x" />
                <div id="get-started">
                    <GetStartedSection />
                </div>
                <div id="desktop-app" className="bg-background ">
                    <div className="rounded-3xl transition-colors border border-purple-500/20 bg-gradient-to-br from-purple-600/10 to-transparent">
                        <TryApp />
                    </div>
                </div>
                <div id="faq">
                    <FAQSection />
                </div>
            </div>
            <div className="w-full max-w-7xl mx-auto pt-40 pb-40 max-sm:px-1">
                <CallToAction />
            </div>
            <Footer />
        </>
    )
}

export { HomePage }
