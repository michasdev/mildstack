import { BGPattern } from "@/components/shared/bg-pattern"
import { CallToAction } from "./components/call-to-action"
import { Features } from "./sections/features/features"
import { HeroSection } from "./sections/hero/hero"


const HomePage = () => {
    return (
        <>
            <HeroSection />
            <div className="relative isolate w-full h-full max-w-7xl mx-auto">
                <BGPattern variant="dots" mask="fade-x" />
                <Features />
                <CallToAction />
            </div>
        </>
    )
}

export { HomePage }