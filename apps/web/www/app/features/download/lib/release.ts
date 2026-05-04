import { Monitor } from 'lucide-react';
import { gitConfig } from '@/lib/shared';
import appleIcon from '@/assets/apple.svg';
import linuxIcon from '@/assets/linux.svg';
import type {
  ArchitectureId,
  AssetExt,
  DownloadAsset,
  DownloadPageData,
  GitHubAsset,
  GitHubRelease,
  PlatformDefinition,
  PlatformId,
  ReleaseDownloads,
} from '../types';

const REPO_SLUG = `${gitConfig.user}/${gitConfig.repo}`;
const RELEASES_API_BASE = `https://api.github.com/repos/${REPO_SLUG}/releases`;
const RELEASES_PAGE = `https://github.com/${REPO_SLUG}/releases`;

const PLATFORMS: PlatformDefinition[] = [
  {
    id: 'macos',
    label: 'macOS',
    iconAlt: 'Apple logo',
    iconSrc: appleIcon,
    architectureLabel: {
      arm64: 'Apple Silicon',
      x64: 'Intel',
    },
  },
  {
    id: 'windows',
    label: 'Windows',
    icon: Monitor,
    architectureLabel: {
      arm64: 'ARM64',
      x64: 'x64',
    },
  },
  {
    id: 'linux',
    label: 'Linux',
    iconAlt: 'Linux logo',
    iconSrc: linuxIcon,
    architectureLabel: {
      arm64: 'ARM64',
      x64: 'x64',
    },
  },
];

function createEmptyDownloads(): ReleaseDownloads {
  return {
    macos: {
      arm64: null,
      x64: null,
    },
    windows: {
      arm64: null,
      x64: null,
    },
    linux: {
      arm64: {
        appImage: null,
        deb: null,
      },
      x64: {
        appImage: null,
        deb: null,
      },
    },
  };
}

function getAssetExtension(name: string): AssetExt | null {
  if (/\.dmg$/i.test(name)) return 'dmg';
  if (/\.exe$/i.test(name)) return 'exe';
  if (/\.appimage$/i.test(name)) return 'appimage';
  if (/\.deb$/i.test(name)) return 'deb';
  return null;
}

function toDownloadAsset(asset: GitHubAsset): DownloadAsset | null {
  const ext = getAssetExtension(asset.name);
  if (!ext) return null;

  return {
    name: asset.name,
    url: asset.browser_download_url,
    ext,
    downloadCount: asset.download_count,
  };
}

function pickLatestRelease(releases: GitHubRelease[]): GitHubRelease | null {
  const published = releases.filter((release) => !release.draft);
  if (published.length === 0) return null;

  return published.sort((a, b) => {
    const aDate = new Date(a.published_at ?? 0).getTime();
    const bDate = new Date(b.published_at ?? 0).getTime();
    return bDate - aDate;
  })[0] ?? null;
}

async function fetchLatestRelease(): Promise<GitHubRelease> {
  const latestResponse = await fetch(`${RELEASES_API_BASE}/latest`, {
    headers: {
      Accept: 'application/vnd.github+json',
    },
  });

  if (latestResponse.ok) {
    return (await latestResponse.json()) as GitHubRelease;
  }

  if (latestResponse.status !== 404) {
    throw new Error(`GitHub releases/latest failed with status ${latestResponse.status}`);
  }

  const fallbackResponse = await fetch(`${RELEASES_API_BASE}?per_page=20`, {
    headers: {
      Accept: 'application/vnd.github+json',
    },
  });

  if (!fallbackResponse.ok) {
    throw new Error(`GitHub releases list failed with status ${fallbackResponse.status}`);
  }

  const releases = (await fallbackResponse.json()) as GitHubRelease[];
  const latestByPublishedDate = pickLatestRelease(releases);

  if (!latestByPublishedDate) {
    throw new Error('No release found in repository');
  }

  return latestByPublishedDate;
}

function mapReleaseDownloads(release: GitHubRelease): ReleaseDownloads {
  const downloads = createEmptyDownloads();

  for (const asset of release.assets) {
    const { name } = asset;

    if (/\.blockmap$/i.test(name)) continue;
    if (/^latest(-mac)?\.yml$/i.test(name)) continue;

    const normalized = toDownloadAsset(asset);
    if (!normalized) continue;

    if (/-mac-arm64\.dmg$/i.test(name)) {
      downloads.macos.arm64 = normalized;
      continue;
    }

    if (/-mac-(x64|intel)\.dmg$/i.test(name)) {
      downloads.macos.x64 = normalized;
      continue;
    }

    if (/-windows-arm64-setup\.exe$/i.test(name)) {
      downloads.windows.arm64 = normalized;
      continue;
    }

    if (/-windows-x64-setup\.exe$/i.test(name)) {
      downloads.windows.x64 = normalized;
      continue;
    }

    if (/-windows-setup\.exe$/i.test(name) && !downloads.windows.x64) {
      downloads.windows.x64 = normalized;
      continue;
    }

    if (/-linux-(amd64|x86_64)\.AppImage$/i.test(name)) {
      downloads.linux.x64.appImage = normalized;
      continue;
    }

    if (/-linux-arm64\.AppImage$/i.test(name)) {
      downloads.linux.arm64.appImage = normalized;
      continue;
    }

    if (/-linux-(amd64|x86_64)\.deb$/i.test(name)) {
      downloads.linux.x64.deb = normalized;
      continue;
    }

    if (/-linux-arm64\.deb$/i.test(name)) {
      downloads.linux.arm64.deb = normalized;
    }
  }

  return downloads;
}

function createFallbackData(error: unknown): DownloadPageData {
  return {
    versionTag: 'unavailable',
    version: 'unavailable',
    releaseUrl: RELEASES_PAGE,
    publishedAt: null,
    downloads: createEmptyDownloads(),
    fetchError: error instanceof Error ? error.message : 'Unable to fetch release data',
  };
}

function formatPublishedDate(dateString: string | null): string {
  if (!dateString) return 'date unavailable';

  try {
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'short',
      day: '2-digit',
    }).format(new Date(dateString));
  } catch {
    return 'date unavailable';
  }
}

function detectRecommendedPlatform() {
  if (typeof navigator === 'undefined') {
    return {
      platform: 'macos' as PlatformId,
      architecture: 'arm64' as ArchitectureId,
    };
  }

  const userAgent = navigator.userAgent.toLowerCase();
  const platform = navigator.platform.toLowerCase();

  const isMac = platform.includes('mac');
  const isWindows = platform.includes('win');
  const isLinux = platform.includes('linux');
  const isArm = userAgent.includes('arm64') || userAgent.includes('aarch64');

  if (isWindows) {
    return {
      platform: 'windows' as PlatformId,
      architecture: isArm ? ('arm64' as ArchitectureId) : ('x64' as ArchitectureId),
    };
  }

  if (isLinux) {
    return {
      platform: 'linux' as PlatformId,
      architecture: isArm ? ('arm64' as ArchitectureId) : ('x64' as ArchitectureId),
    };
  }

  if (isMac) {
    return {
      platform: 'macos' as PlatformId,
      architecture: isArm ? ('arm64' as ArchitectureId) : ('x64' as ArchitectureId),
    };
  }

  return {
    platform: 'macos' as PlatformId,
    architecture: 'arm64' as ArchitectureId,
  };
}

async function loadDownloadPageData(): Promise<DownloadPageData> {
  try {
    const release = await fetchLatestRelease();

    return {
      versionTag: release.tag_name,
      version: release.tag_name.replace(/^v/i, ''),
      releaseUrl: release.html_url || RELEASES_PAGE,
      publishedAt: release.published_at,
      downloads: mapReleaseDownloads(release),
    };
  } catch (error) {
    return createFallbackData(error);
  }
}

export {
  PLATFORMS,
  RELEASES_PAGE,
  createEmptyDownloads,
  detectRecommendedPlatform,
  fetchLatestRelease,
  formatPublishedDate,
  loadDownloadPageData,
  mapReleaseDownloads,
};
