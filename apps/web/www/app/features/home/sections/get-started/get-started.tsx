import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { ArrowLeft, ArrowRight } from 'lucide-react'
import { useState } from 'react'
import { TerminalStepContent } from './terminal-content'

type StepId = 'start' | 'configure' | 'build'

interface StepDefinition {
  id: StepId
  number: string
  label: string
  title: string
  description: string
}

interface UpcomingService {
  name: string
}

const STEPS: StepDefinition[] = [
  {
    id: 'start',
    number: '01',
    label: 'Start',
    title: 'One command to start',
    description:
      'Spin up all AWS services locally in seconds. No Docker, no YAML, no cloud account.',
  },
  {
    id: 'configure',
    number: '02',
    label: 'Configure',
    title: 'Point to localhost',
    description:
      'Set one env variable. Your existing AWS SDKs and CLI commands work as-is.',
  },
  {
    id: 'build',
    number: '03',
    label: 'Build',
    title: 'Build and test locally',
    description:
      'Create real AWS resources with the same commands you already use in production.',
  },
]

const UPCOMING_SERVICES: UpcomingService[] = [
  { name: 'Lambda' },
  { name: 'EventBridge' },
  { name: 'CloudWatch Logs' },
  { name: 'And More!' },
]

function TerminalChrome() {
  return (
    <div className="flex items-center gap-2 border-b border-border/70 bg-muted/20 px-5 py-3">
      <div className="flex items-center gap-1.5 absolute">
        <span className="size-3 rounded-full bg-red-500" />
        <span className="size-3 rounded-full bg-yellow-300" />
        <span className="size-3 rounded-full bg-emerald-600" />
      </div>
      <div className="flex flex-1 justify-center">
        <span className="font-mono text-[11px] text-muted-foreground/70">
          MildStack
        </span>
      </div>
    </div>
  )
}

interface StepPillProps {
  step: StepDefinition
  isActive: boolean
  onClick: () => void
}

function StepPill({ step, isActive, onClick }: StepPillProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={isActive ? 'step' : undefined}
      className={cn(
        'flex items-center gap-2 rounded-full border px-4 py-2 transition-all duration-200',
        isActive
          ? 'border-primary/40 bg-primary/20 text-primary-foreground shadow-lg shadow-primary/15'
          : 'border-border/80 bg-muted/20 text-muted-foreground hover:bg-muted/30 hover:text-foreground'
      )}
    >
      <span
        className={cn(
          'flex size-5 items-center justify-center rounded-full font-mono text-[10px] font-semibold',
          isActive ? 'bg-primary text-primary-foreground' : 'bg-muted text-muted-foreground'
        )}
      >
        {step.number}
      </span>
      <span className="text-sm font-semibold">{step.label}</span>
    </button>
  )
}

function StepSeparator() {
  return <span className="hidden h-px w-8 bg-border md:block" aria-hidden />
}

export function GetStartedSection() {
  const [activeStepIndex, setActiveStepIndex] = useState(0)
  const activeStep = STEPS[activeStepIndex]

  return (
    <section className="mx-auto w-full max-w-6xl px-4 py-16 sm:px-6 md:py-20 lg:py-24">
      <div className="mx-auto max-w-2xl text-center">
        <p className="text-primary text-xs font-semibold tracking-[0.2em] uppercase">
          Get Started
        </p>
        <h2 className="mt-4 text-balance text-4xl font-light lg:text-6xl">
          Run AWS locally
          <br />
          in 3 steps
        </h2>
        <p className="mt-4 text-base text-gray-400 font-light">
          No Docker. No cloud account. No telemetry. No Payment.
        </p>
      </div>

      <div className="mt-10 flex justify-center">
        <div className="flex flex-wrap items-center justify-center gap-3">
          {STEPS.map((step, index) => (
            <div key={step.id} className="flex items-center gap-3">
              <StepPill
                step={step}
                isActive={activeStepIndex === index}
                onClick={() => setActiveStepIndex(index)}
              />
              {index < STEPS.length - 1 ? <StepSeparator /> : null}
            </div>
          ))}
        </div>
      </div>

      <div className="mx-auto mt-8 max-w-4xl overflow-hidden rounded-3xl border border-border/80 bg-card shadow-[0_32px_80px_-50px_color-mix(in_oklab,var(--color-primary)_65%,transparent)]">
        <TerminalChrome />

        <div className="px-5 pt-6 sm:px-7">
          <div className="flex items-start justify-between gap-4">
            <div className="max-w-2xl">
              <p className="text-primary font-mono text-[11px] tracking-[0.17em] uppercase">
                step {activeStep.number}
              </p>
              <h3 className="mt-1 text-2xl font-bold">{activeStep.title}</h3>
              <p className="text-muted-foreground mt-2 text-sm leading-relaxed">
                {activeStep.description}
              </p>
            </div>

            <div className="flex shrink-0 gap-2">
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                disabled={activeStepIndex === 0}
                onClick={() => setActiveStepIndex((prev) => Math.max(0, prev - 1))}
                className="rounded-full border border-border/70 bg-muted/30 text-muted-foreground hover:bg-muted/40 hover:text-foreground disabled:opacity-35"
              >
                <ArrowLeft className="size-4" />
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                disabled={activeStepIndex === STEPS.length - 1}
                onClick={() =>
                  setActiveStepIndex((prev) => Math.min(STEPS.length - 1, prev + 1))
                }
                className="rounded-full border border-border/70 bg-muted/30 text-muted-foreground hover:bg-muted/40 hover:text-foreground disabled:opacity-35"
              >
                <ArrowRight className="size-4" />
              </Button>
            </div>
          </div>

          <div className="my-5 h-px w-full bg-border/70" />

          <div className="pb-7">
            <TerminalStepContent stepId={activeStep.id} />
          </div>
        </div>
      </div>

      <div className="mx-auto mt-5 max-w-4xl rounded-full border border-border/70 bg-muted/15 px-4 py-3">
        <div className="flex flex-wrap items-center justify-center gap-x-3 gap-y-2 text-sm">
          <span className="text-muted-foreground/80 text-xs font-semibold uppercase tracking-wide">
            Coming soon
          </span>
          <span className="hidden h-3 w-px bg-border sm:block" />
          {UPCOMING_SERVICES.map((service) => (
            <>
              <div key={service.name} className="flex items-center gap-2">
                <span className="text-sm text-muted-foreground/70">
                  {service.name}
                </span>
              </div>
              {service.name !== UPCOMING_SERVICES[UPCOMING_SERVICES.length - 1]?.name && (
                <span key={service.name + '_divider'} className="hidden h-3 w-px bg-border sm:block" />
              )}
            </>
          ))}
        </div>
      </div>
    </section>
  )
}

export type { StepId }
