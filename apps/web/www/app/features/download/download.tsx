import { Footer } from '@/components/shared/footer';
import { Navbar } from '@/components/shared/navbar';
import { BGPattern } from '@/components/shared/bg-pattern';
import { DownloadHero } from './components/download-hero';
import { DownloadPanel } from './components/download-panel';
import { useDownloadPage } from './hooks/use-download-page';
import type { DownloadPageData } from './types';

interface DownloadPageProps {
  loaderData: DownloadPageData;
}

function DownloadPage({ loaderData }: DownloadPageProps) {
  const {
    activePlatformConfig,
    availableArchitectures,
    releaseData,
    selectedArch,
    selectedDownloadExt,
    selectedDownloadUrl,
    selectedLinuxFormat,
    selectedPlatform,
    setSelectedArch,
    setSelectedLinuxFormat,
    setSelectedPlatform,
  } = useDownloadPage(loaderData);

  return (
    <div className="relative bg-background text-foreground">
      <div className="relative flex min-h-dvh flex-col md:h-dvh">
        <div className="fixed top-0 left-0 right-0 z-50 w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 pointer-events-none">
          <div className="pointer-events-auto">
            <Navbar />
          </div>
        </div>

        <main className="relative isolate mx-auto flex w-full max-w-7xl flex-1 flex-col px-4 pt-20 pb-10 sm:px-6 lg:px-8">
          <BGPattern
            variant="dots"
            mask="fade-y"
            fill="color-mix(in srgb, var(--color-primary) 12%, transparent)"
          />

          <section className="my-8 grid w-full items-center gap-8 pb-8 md:my-auto lg:grid-cols-[1fr_460px] lg:gap-12">
            <DownloadHero releaseUrl={releaseData.releaseUrl} versionTag={releaseData.versionTag} />

            <DownloadPanel
              activePlatformConfig={activePlatformConfig}
              availableArchitectures={availableArchitectures}
              fetchError={releaseData.fetchError}
              publishedAt={releaseData.publishedAt}
              releaseDownloads={releaseData.downloads}
              selectedArch={selectedArch}
              selectedDownloadExt={selectedDownloadExt}
              selectedDownloadUrl={selectedDownloadUrl}
              selectedLinuxFormat={selectedLinuxFormat}
              selectedPlatform={selectedPlatform}
              setSelectedArch={setSelectedArch}
              setSelectedLinuxFormat={setSelectedLinuxFormat}
              setSelectedPlatform={setSelectedPlatform}
              version={releaseData.version}
            />
          </section>
        </main>
      </div>

      <Footer />
    </div>
  );
}

export { DownloadPage };
