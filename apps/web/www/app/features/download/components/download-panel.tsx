import { Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { PLATFORMS, formatPublishedDate } from '../lib/release';
import type {
  ArchitectureId,
  LinuxFormat,
  PlatformDefinition,
  PlatformId,
  ReleaseDownloads,
} from '../types';

interface DownloadPanelProps {
  activePlatformConfig: PlatformDefinition;
  availableArchitectures: ArchitectureId[];
  fetchError?: string;
  publishedAt: string | null;
  releaseDownloads: ReleaseDownloads;
  selectedArch: ArchitectureId;
  selectedDownloadExt: string;
  selectedDownloadUrl: string | null;
  selectedLinuxFormat: LinuxFormat;
  selectedPlatform: PlatformId;
  setSelectedArch: (value: ArchitectureId) => void;
  setSelectedLinuxFormat: (value: LinuxFormat) => void;
  setSelectedPlatform: (value: PlatformId) => void;
  version: string;
}

function DownloadPanel({
  activePlatformConfig,
  availableArchitectures,
  fetchError,
  publishedAt,
  releaseDownloads,
  selectedArch,
  selectedDownloadExt,
  selectedDownloadUrl,
  selectedLinuxFormat,
  selectedPlatform,
  setSelectedArch,
  setSelectedLinuxFormat,
  setSelectedPlatform,
  version,
}: DownloadPanelProps) {
  return (
    <div className="relative z-10 rounded-3xl border border-primary/20 bg-card/90 p-5 shadow-[0_30px_80px_-40px_color-mix(in_oklab,var(--color-primary)_60%,transparent)]">
      <div className="grid grid-cols-3 gap-2">
        {PLATFORMS.map((platform) => {
          const Icon = platform.icon;
          const isSelected = selectedPlatform === platform.id;

          return (
            <button
              key={platform.id}
              type="button"
              onClick={() => setSelectedPlatform(platform.id)}
              className={cn(
                'flex flex-col items-center justify-center gap-1 rounded-xl border px-2 py-3 text-sm font-semibold transition-all',
                isSelected
                  ? 'border-primary/55 bg-primary text-primary-foreground'
                  : 'border-border bg-background/40 text-muted-foreground hover:border-primary/35 hover:text-foreground'
              )}
            >
              {platform.iconSrc ? (
                <img
                  src={platform.iconSrc}
                  alt={platform.iconAlt ?? `${platform.label} icon`}
                  className={cn(
                    'size-4 object-contain brightness-0 invert',
                    isSelected ? 'opacity-100' : 'opacity-70'
                  )}
                />
              ) : Icon ? (
                <Icon
                  className={cn(
                    'size-4 text-white',
                    isSelected ? 'opacity-100' : 'opacity-70'
                  )}
                />
              ) : null}
              {platform.label}
            </button>
          );
        })}
      </div>

      <div className="mt-5">
        <p className="text-muted-foreground text-xs font-semibold tracking-[0.17em] uppercase">
          Architecture
        </p>

        <div className="mt-2 grid grid-cols-2 gap-2">
          {(['arm64', 'x64'] as ArchitectureId[]).map((architecture) => {
            const isSelected = selectedArch === architecture;
            const isAvailable = availableArchitectures.includes(architecture);

            return (
              <button
                key={architecture}
                type="button"
                disabled={!isAvailable}
                onClick={() => setSelectedArch(architecture)}
                className={cn(
                  'rounded-xl border px-3 py-3 text-left transition-all',
                  isSelected && isAvailable
                    ? 'border-primary/55 bg-primary/90 text-primary-foreground'
                    : 'border-border bg-background/40 text-foreground/90',
                  !isAvailable && 'cursor-not-allowed opacity-40'
                )}
              >
                <p className="text-sm font-semibold">
                  {activePlatformConfig.architectureLabel[architecture]}
                </p>
                <p className="text-xs opacity-75">{architecture === 'arm64' ? 'arm64' : 'x86_64'}</p>
              </button>
            );
          })}
        </div>
      </div>

      {selectedPlatform === 'linux' ? (
        <div className="mt-4">
          <p className="text-muted-foreground text-xs font-semibold tracking-[0.17em] uppercase">
            Package format
          </p>
          <div className="mt-2 grid grid-cols-2 gap-2">
            {([
              { id: 'appImage', label: 'AppImage' },
              { id: 'deb', label: '.deb' },
            ] as Array<{ id: LinuxFormat; label: string }>).map((format) => {
              const isSelected = selectedLinuxFormat === format.id;
              const isAvailable = Boolean(releaseDownloads.linux[selectedArch][format.id]);

              return (
                <button
                  key={format.id}
                  type="button"
                  disabled={!isAvailable}
                  onClick={() => setSelectedLinuxFormat(format.id)}
                  className={cn(
                    'rounded-xl border px-3 py-2 text-sm font-semibold transition-all',
                    isSelected && isAvailable
                      ? 'border-primary/55 bg-primary/90 text-primary-foreground'
                      : 'border-border bg-background/40 text-foreground/90',
                    !isAvailable && 'cursor-not-allowed opacity-40'
                  )}
                >
                  {format.label}
                </button>
              );
            })}
          </div>
        </div>
      ) : null}

      <div className="mt-5">
        <Button
          asChild
          className={cn(
            'h-12 w-full rounded-xl text-base font-semibold',
            selectedDownloadUrl ? 'bg-primary hover:bg-primary/90' : 'cursor-not-allowed opacity-50'
          )}
        >
          {selectedDownloadUrl ? (
            <a href={selectedDownloadUrl} rel="noreferrer noopener">
              <Download className="size-4" />
              Download for {activePlatformConfig.label}
            </a>
          ) : (
            <span>
              <Download className="size-4" />
              Download unavailable
            </span>
          )}
        </Button>
      </div>

      <p className="text-muted-foreground mt-4 text-center font-mono text-xs">
        v{version} • {selectedDownloadExt} • {activePlatformConfig.label}{' '}
        {activePlatformConfig.architectureLabel[selectedArch]}
      </p>

      <p className="text-muted-foreground mt-2 text-center text-xs">
        Published {formatPublishedDate(publishedAt)}
      </p>

      {fetchError ? (
        <p className="mt-3 rounded-lg border border-destructive/40 bg-destructive/10 p-2 text-center text-xs text-destructive">
          We could not load release metadata from GitHub right now.
        </p>
      ) : null}
    </div>
  );
}

export { DownloadPanel };
