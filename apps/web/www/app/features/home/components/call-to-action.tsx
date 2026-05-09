import { ArrowRightIcon, PlusIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useInstallationTarget } from "@/hooks/use-installation-target";

export function CallToAction() {
  const { installationHref } = useInstallationTarget();

  return (
    <div className="relative mx-auto flex w-full max-w-3xl flex-col justify-between gap-y-6 border-y bg-background bg-[radial-gradient(35%_80%_at_75%_0%,color-mix(in_oklab,var(--color-primary)_35%,var(--color-background)),transparent)] px-4 py-8">
      <PlusIcon
        className="absolute top-[-12.5px] left-[-11.5px] z-1 size-6"
        strokeWidth={1}
      />
      <PlusIcon
        className="absolute top-[-12.5px] right-[-11.5px] z-1 size-6"
        strokeWidth={1}
      />
      <PlusIcon
        className="absolute bottom-[-12.5px] left-[-11.5px] z-1 size-6"
        strokeWidth={1}
      />
      <PlusIcon
        className="absolute right-[-11.5px] bottom-[-12.5px] z-1 size-6"
        strokeWidth={1}
      />

      <div className="-inset-y-6 pointer-events-none absolute left-0 w-px border-l" />
      <div className="-inset-y-6 pointer-events-none absolute right-0 w-px border-r" />

      <div className="-z-10 absolute top-0 left-1/2 h-full border-l border-dashed" />


      <div className="space-y-1">
        <h2 className="text-center font-bold text-2xl">
          Download and run your own AWS locally
        </h2>
        <p className="text-center text-muted-foreground">
          Get started with MildStack.
        </p>
      </div>

      <div className="flex items-center justify-center gap-2">
        <Button asChild variant="outline">
          <a href="https://github.com/michasdev/mildstack" target="_blank" rel="noopener noreferrer">
            View source
          </a>
        </Button>
        <Button asChild>
          <a href={installationHref}>
            Download <ArrowRightIcon className="size-4 ml-1" />
          </a>
        </Button>
      </div>
    </div>
  );
}
