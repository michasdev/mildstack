import { ArrowUpRight, Terminal } from 'lucide-react';
import { RELEASES_PAGE } from '../lib/release';

interface DownloadHeroProps {
  releaseUrl: string;
  versionTag: string;
}

function DownloadHero({ releaseUrl, versionTag }: DownloadHeroProps) {
  return (
    <div className="relative z-10 flex flex-col justify-center">
      <p className="text-primary text-sm font-semibold tracking-[0.2em] uppercase">{versionTag}</p>

      <h1 className="mt-4 text-5xl font-extrabold leading-[0.95] tracking-tight sm:text-6xl">
        Run AWS locally.
      </h1>

      <p className="text-muted-foreground mt-6 max-w-xl text-lg leading-relaxed">
        Install the MildStack Desktop App for macOS, Windows, or Linux and start emulating AWS
        services on your machine.
      </p>

      <div className="mt-8 max-w-xl rounded-2xl border border-primary/25 bg-primary/10 p-4">
        <div className="flex items-start gap-3">
          <div className="mt-0.5 rounded-lg bg-primary p-2 text-primary-foreground">
            <Terminal className="size-4" />
          </div>
          <div>
            <p className="font-semibold text-primary">CLI included in the app</p>
            <p className="text-muted-foreground mt-1 text-sm">
              The MildStack CLI is bundled with the Desktop App. Install the app once and the CLI
              is ready automatically.
            </p>
          </div>
        </div>
      </div>

      <div className="mt-10 flex items-center gap-4 text-sm">
        <a
          href={releaseUrl}
          target="_blank"
          rel="noreferrer noopener"
          className="text-muted-foreground hover:text-primary transition-colors"
        >
          All releases
        </a>
        <span className="h-4 w-px bg-border" />
        <a
          href={RELEASES_PAGE}
          target="_blank"
          rel="noreferrer noopener"
          className="text-muted-foreground hover:text-primary inline-flex items-center gap-1.5 transition-colors"
        >
          <ArrowUpRight className="size-4" />
          GitHub
        </a>
      </div>
    </div>
  );
}

export { DownloadHero };

