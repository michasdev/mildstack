import { Accordion, AccordionItem, AccordionContent, AccordionTrigger } from '@/components/ui/accordion'
import { HardDrive, Database, LayoutGrid, Cloud } from 'lucide-react'
import { useState } from 'react'
import { motion, AnimatePresence } from 'motion/react'
import { BorderBeam } from '@/components/ui/border-beam'

function TryApp() {
    type ImageKey = 'item-1' | 'item-2' | 'item-3' | 'item-4'
    const [activeItem, setActiveItem] = useState<ImageKey>('item-1')

    const images = {
        'item-1': {
            image: '/s3-browser.png',
            alt: 'S3 Resource Browser interface',
        },
        'item-2': {
            image: '/dynamodb-browser.png',
            alt: 'DynamoDB Explorer dashboard',
        },
        'item-3': {
            image: '/instances.png',
            alt: 'Multi-instance management panel',
        },
        'item-4': {
            image: '/services.png',
            alt: 'Universal AWS service support',
        },
    }

    return (
        <section className="py-12 md:py-20 lg:py-32">
            <div className="bg-transparent absolute inset-0 -z-10 sm:inset-6 sm:rounded-b-3xl dark:block"></div>
            <div className="mx-auto max-w-5xl space-y-8 px-6 md:space-y-16 lg:space-y-20 dark:[--color-border:color-mix(in_oklab,var(--color-white)_10%,transparent)]">
                <div className="relative z-10 mx-auto max-w-2xl space-y-6 text-center">
                    <p className="text-primary text-xs font-semibold tracking-[0.2em] uppercase">
                        MildStack Desktop
                    </p>
                    <h2 className="text-balance text-4xl font-light lg:text-6xl">Try our Desktop App</h2>
                    <p className='text-gray-400 font-light'>MildStack provides a powerful multiplatform Desktop App with an intuitive visual interface to manage your local cloud resources with ease.</p>
                </div>

                <div className="grid items-start gap-12 sm:px-12 md:grid-cols-2 lg:gap-20 lg:px-0">
                    <Accordion
                        type='single'
                        collapsible
                        value={activeItem}
                        onValueChange={(value) => {
                            if (value) setActiveItem(value as ImageKey)
                        }}
                        className="w-full self-start">
                        <AccordionItem value="item-1">
                            <AccordionTrigger>
                                <div className="flex items-center gap-2 text-base">
                                    <HardDrive className="size-4" />
                                    S3 Resource Browser
                                </div>
                            </AccordionTrigger>
                            <AccordionContent>
                                Navigate through your buckets and files with a powerful explorer. Upload, download, and manage your S3 objects directly from the desktop interface.
                            </AccordionContent>
                        </AccordionItem>
                        <AccordionItem value="item-2">
                            <AccordionTrigger>
                                <div className="flex items-center gap-2 text-base">
                                    <Database className="size-4" />
                                    DynamoDB Explorer
                                </div>
                            </AccordionTrigger>
                            <AccordionContent>
                                A dedicated interface for your NoSQL data. Perform advanced scans and queries, edit items in place, and manage your tables without leaving the app.
                            </AccordionContent>
                        </AccordionItem>
                        <AccordionItem value="item-3">
                            <AccordionTrigger>
                                <div className="flex items-center gap-2 text-base">
                                    <LayoutGrid className="size-4" />
                                    Multi-Instance Management
                                </div>
                            </AccordionTrigger>
                            <AccordionContent>
                                Effortlessly manage and switch between multiple MildStack instances. Monitor resource usage and configuration across your entire local development environment.
                            </AccordionContent>
                        </AccordionItem>
                        <AccordionItem value="item-4">
                            <AccordionTrigger>
                                <div className="flex items-center gap-2 text-base">
                                    <Cloud className="size-4" />
                                    Universal AWS Support
                                </div>
                            </AccordionTrigger>
                            <AccordionContent>
                                Full support for SQS, SNS, DynamoDB, S3, and more. A unified dashboard for your entire local cloud ecosystem, making development faster and simpler.
                            </AccordionContent>
                        </AccordionItem>
                    </Accordion>

                    <div className="bg-background relative flex self-start overflow-hidden rounded-3xl border p-2">
                        <div className="w-15 absolute inset-0 right-0 ml-auto border-l bg-[repeating-linear-gradient(-45deg,var(--color-border),var(--color-border)_1px,transparent_1px,transparent_8px)]"></div>
                        <div className="aspect-video bg-background relative w-full rounded-2xl">
                            <AnimatePresence mode="wait">
                                <motion.div
                                    key={`${activeItem}-id`}
                                    initial={{ opacity: 0, y: 6, scale: 0.98 }}
                                    animate={{ opacity: 1, y: 0, scale: 1 }}
                                    exit={{ opacity: 0, y: 6, scale: 0.98 }}
                                    transition={{ duration: 0.2 }}
                                    className="size-full overflow-hidden rounded-2xl border bg-zinc-900 shadow-md">
                                    <img
                                        src={images[activeItem].image}
                                        className="size-full object-cover object-center dark:mix-blend-lighten"
                                        alt={images[activeItem].alt}
                                        width={1207}
                                        height={929}
                                    />
                                </motion.div>
                            </AnimatePresence>
                        </div>
                        <BorderBeam
                            duration={6}
                            size={200}
                            className="from-transparent to-transparent dark:via-purple-500"
                        />
                    </div>
                </div>
            </div>
        </section>
    )
}

export { TryApp }
