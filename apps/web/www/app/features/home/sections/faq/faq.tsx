import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import type { ReactNode } from 'react'

interface FaqItem {
  id: string
  question: string
  answer: ReactNode
}

const FAQ_ITEMS: FaqItem[] = [
  {
    id: 'what-is-mildstack',
    question: 'What is MildStack?',
    answer:
      'MildStack is a lightweight, local-first AWS emulator that runs natively on your machine. It helps you build and test cloud workflows locally without Docker.',
  },
  {
    id: 'docker-required',
    question: 'Do I need Docker to use MildStack?',
    answer:
      'No. MildStack runs as a native binary and starts quickly with low CPU and RAM usage, so you can keep your local feedback loop fast.',
  },
  {
    id: 'supported-services',
    question: 'Which AWS services are supported today?',
    answer:
      'MildStack currently supports S3, DynamoDB, SQS, and SNS for local development workflows.',
  },
  {
    id: 'sdk-cli-compatible',
    question: 'Can I use the official AWS CLI and SDKs?',
    answer:
      'Yes. MildStack is AWS API compatible for supported operations. Point your tooling to the local endpoint (for example, http://localhost:4566) and keep using your existing commands.',
  },
  {
    id: 'persistence',
    question: 'Does data persist between restarts?',
    answer:
      'Yes. All MildStack services persist data locally by default, scoped per instance. If you delete an instance, all service data associated with that instance is permanently removed.',
  },
  {
    id: 'desktop-app',
    question: 'Is there a visual interface, or only the CLI?',
    answer:
      'You can use both. MildStack includes a Desktop App to browse S3 buckets, inspect DynamoDB tables, and monitor SQS queues, while the CLI handles instance lifecycle and automation.',
  },
]

export function FAQSection() {
  return (
    <section className="mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 md:py-20 lg:py-24">
      <div className="mx-auto max-w-2xl text-center">
        <p className="text-primary text-xs font-semibold tracking-[0.2em] uppercase">FAQ</p>
        <h2 className="mt-4 text-balance text-4xl font-light lg:text-6xl">
          Frequently asked questions
        </h2>
        <p className="mt-4 text-base text-gray-400 font-light">
          Answers to the most common questions about running AWS locally with MildStack.
        </p>
      </div>

      <div className="mx-auto mt-10 max-w-4xl rounded-3xl border border-border/80 bg-card/70 p-3 backdrop-blur-sm shadow-[0_24px_80px_-60px_color-mix(in_oklab,var(--color-primary)_65%,transparent)]">
        <Accordion type="single" collapsible className="w-full">
          {FAQ_ITEMS.map((item) => (
            <AccordionItem key={item.id} value={item.id}>
              <AccordionTrigger className="text-left text-base font-semibold">
                {item.question}
              </AccordionTrigger>
              <AccordionContent className="text-muted-foreground text-sm leading-relaxed">
                {item.answer}
              </AccordionContent>
            </AccordionItem>
          ))}
        </Accordion>
      </div>
    </section>
  )
}
