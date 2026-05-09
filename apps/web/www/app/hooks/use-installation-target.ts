import { useEffect, useState } from 'react';

type PlatformFamily = 'macos' | 'windows' | 'linux' | 'unknown';

const DOWNLOAD_ROUTE = '/download';
const INSTALLATION_DOCS_ROUTE = '/docs/getting-started/installation';

function detectPlatform(): PlatformFamily {
  if (typeof navigator === 'undefined') return 'unknown';

  const platform = navigator.platform.toLowerCase();
  const userAgent = navigator.userAgent.toLowerCase();

  if (platform.includes('mac') || userAgent.includes('mac os')) return 'macos';
  if (platform.includes('win') || userAgent.includes('windows')) return 'windows';
  if (platform.includes('linux') || userAgent.includes('linux')) return 'linux';

  return 'unknown';
}

export function useInstallationTarget() {
  const [platform, setPlatform] = useState<PlatformFamily>('unknown');

  useEffect(() => {
    setPlatform(detectPlatform());
  }, []);

  const installationHref =
    platform === 'macos' ? INSTALLATION_DOCS_ROUTE : DOWNLOAD_ROUTE;

  return {
    downloadRoute: DOWNLOAD_ROUTE,
    installationDocsRoute: INSTALLATION_DOCS_ROUTE,
    installationHref,
    isMacOS: platform === 'macos',
    platform,
  };
}

