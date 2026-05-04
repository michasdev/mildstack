import { cn } from '@/lib/utils'
import type { ReactNode } from 'react'
import type { StepId } from './get-started'

function Cursor() {
  return (
    <span className="bg-primary animate-terminal-cursor inline-block h-[14px] w-[7px] rounded-sm align-middle" />
  )
}

function Prompt() {
  return <span className="text-primary">$</span>
}

function Line({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn('font-mono text-sm leading-7 sm:leading-8', className)}>
      {children}
    </div>
  )
}

function OutputBlock({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-2xl border border-border/70 bg-muted/20 px-4 py-3">
      {children}
    </div>
  )
}

function StartStep() {
  return (
    <div className="animate-terminal-slide-up space-y-2">
      <Line>
        <Prompt /> <span className="text-foreground/85">mildstack start</span>
      </Line>

      <Line className="text-emerald-400">
        {'✓ MildStack running on http://localhost:4566'}
      </Line>

      <div className="mt-3 rounded-2xl border border-border/70 border-dashed bg-muted/15 px-4 py-3">
        <p className="text-muted-foreground mb-2 font-mono text-[10px] tracking-[0.12em] uppercase">
          # all services on a single port
        </p>
        {['S3', 'DynamoDB', 'SQS', 'SNS'].map((serviceName) => (
          <Line key={serviceName}>
            <span className="text-emerald-500/60">{'✓ '}</span>
            <span className="text-foreground/60">{serviceName.padEnd(12)}</span>
            <span className="text-muted-foreground">:4566</span>
          </Line>
        ))}
        <Line className="mt-1 text-primary">{'-> http://localhost:4566'}</Line>
      </div>

      <Line>
        <Prompt /> <Cursor />
      </Line>
    </div>
  )
}

function ConfigureStep() {
  return (
    <div className="animate-terminal-slide-up space-y-2">
      <Line className="text-muted-foreground"># Set for all AWS CLI commands</Line>
      <Line>
        <Prompt /> <span className="text-foreground/85">export </span>
        <span className="text-primary">AWS_ENDPOINT_URL</span>
        <span className="text-foreground/85">=http://localhost:4566</span>
      </Line>

      <OutputBlock>
        <Line className="text-muted-foreground">{'// AWS SDK (example using Node.js, works with any AWS SDK)'}</Line>
        <Line>
          <span className="text-primary">const</span>
          <span className="text-foreground/80"> client </span>
          <span className="text-foreground/50">= </span>
          <span className="text-primary">new</span>
          <span className="text-foreground/80">
            {' S3Client({'}
          </span>
        </Line>
        <Line className="pl-5 text-foreground/65">
          endpoint: <span className="text-emerald-300">"http://localhost:4566"</span>,
        </Line>
        <Line className="pl-5 text-foreground/65">
          region: <span className="text-emerald-300">"us-east-1"</span>
        </Line>
        <Line className="text-foreground/80">{'});'}</Line>
      </OutputBlock>

      <Line className="pt-1">
        <Prompt /> <Cursor />
      </Line>
    </div>
  )
}

function BuildStep() {
  return (
    <div className="animate-terminal-slide-up space-y-1">
      <Line>
        <Prompt /> <span className="text-foreground/85">aws s3 mb s3://my-app-bucket</span>
      </Line>
      <Line className="pl-4 text-emerald-400">make_bucket: my-app-bucket</Line>

      <div className="h-1.5" />

      <Line>
        <Prompt /> <span className="text-foreground/85">aws dynamodb create-table \</span>
      </Line>
      <Line className="pl-4 text-foreground/50">--table-name users \</Line>
      <Line className="pl-4 text-foreground/50">--billing-mode PAY_PER_REQUEST</Line>
      <Line className="pl-4 text-emerald-400">Table "users" created</Line>

      <div className="h-1.5" />

      <Line>
        <Prompt /> <span className="text-foreground/85">aws sqs create-queue --queue-name jobs</span>
      </Line>
      <Line className="pl-4 text-emerald-400">
        {`{ "QueueUrl": "http://localhost:4566/queue/jobs" }`}
      </Line>

      <div className="h-1.5" />

      <Line>
        <Prompt /> <span className="text-foreground/85">aws sns create-topic --name alerts</span>
      </Line>
      <Line className="pl-4 text-emerald-400">
        {`{ "TopicArn": "arn:aws:sns:us-east-1:000:alerts" }`}
      </Line>

      <Line className="pt-2">
        <Prompt /> <Cursor />
      </Line>
    </div>
  )
}

const STEP_CONTENT: Record<StepId, ReactNode> = {
  start: <StartStep />,
  configure: <ConfigureStep />,
  build: <BuildStep />,
}

export function TerminalStepContent({ stepId }: { stepId: StepId }) {
  return <>{STEP_CONTENT[stepId]}</>
}
