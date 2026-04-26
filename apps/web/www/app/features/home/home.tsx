import { BGPattern } from "@/components/shared/bg-pattern"
import { CallToAction } from "./components/call-to-action"
import { Features } from "./sections/features/features"
import { HeroSection } from "./sections/hero/hero"
import { Footer } from "@/components/shared/footer"


const HomePage = () => {
    return (
        <>
            <HeroSection />
            <div className="relative isolate w-full h-full max-w-7xl mx-auto py-12 pb-40">
                <BGPattern variant="dots" mask="fade-x" />
                <Features />
                <CallToAction />
            </div>
            <Footer />
        </>
    )
}

export { HomePage }