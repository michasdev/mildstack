import { useEffect, useMemo, useState } from 'react';
import {
  PLATFORMS,
  detectRecommendedPlatform,
  fetchLatestRelease,
  mapReleaseDownloads,
  RELEASES_PAGE,
} from '../lib/release';
import type {
  ArchitectureId,
  DownloadPageData,
  LinuxFormat,
  PlatformDefinition,
  PlatformId,
} from '../types';

interface UseDownloadPageResult {
  activePlatformConfig: PlatformDefinition;
  availableArchitectures: ArchitectureId[];
  releaseData: DownloadPageData;
  selectedArch: ArchitectureId;
  selectedDownloadExt: string;
  selectedDownloadUrl: string | null;
  selectedLinuxFormat: LinuxFormat;
  selectedPlatform: PlatformId;
  setSelectedArch: (value: ArchitectureId) => void;
  setSelectedLinuxFormat: (value: LinuxFormat) => void;
  setSelectedPlatform: (value: PlatformId) => void;
}

function useDownloadPage(loaderData: DownloadPageData): UseDownloadPageResult {
  const [releaseData, setReleaseData] = useState<DownloadPageData>(loaderData);
  const [selectedPlatform, setSelectedPlatform] = useState<PlatformId>('macos');
  const [selectedArch, setSelectedArch] = useState<ArchitectureId>('arm64');
  const [selectedLinuxFormat, setSelectedLinuxFormat] = useState<LinuxFormat>('appImage');

  useEffect(() => {
    const recommended = detectRecommendedPlatform();
    setSelectedPlatform(recommended.platform);
    setSelectedArch(recommended.architecture);
  }, []);

  useEffect(() => {
    let isMounted = true;

    async function refreshReleaseData() {
      try {
        const latestRelease = await fetchLatestRelease();
        if (!isMounted) return;

        setReleaseData({
          versionTag: latestRelease.tag_name,
          version: latestRelease.tag_name.replace(/^v/i, ''),
          releaseUrl: latestRelease.html_url || RELEASES_PAGE,
          publishedAt: latestRelease.published_at,
          downloads: mapReleaseDownloads(latestRelease),
        });
      } catch {
        // Keep statically loaded data when runtime refresh fails.
      }
    }

    refreshReleaseData();

    return () => {
      isMounted = false;
    };
  }, []);

  const selectedDownload = useMemo(() => {
    if (selectedPlatform === 'linux') {
      return releaseData.downloads.linux[selectedArch][selectedLinuxFormat];
    }

    return releaseData.downloads[selectedPlatform][selectedArch];
  }, [releaseData.downloads, selectedArch, selectedLinuxFormat, selectedPlatform]);

  const availableArchitectures = useMemo(() => {
    if (selectedPlatform === 'linux') {
      return (['arm64', 'x64'] as ArchitectureId[]).filter((arch) => {
        const linuxAssets = releaseData.downloads.linux[arch];
        return Boolean(linuxAssets.appImage || linuxAssets.deb);
      });
    }

    return (['arm64', 'x64'] as ArchitectureId[]).filter((arch) =>
      Boolean(releaseData.downloads[selectedPlatform][arch])
    );
  }, [releaseData.downloads, selectedPlatform]);

  useEffect(() => {
    if (availableArchitectures.includes(selectedArch)) return;
    setSelectedArch(availableArchitectures[0] ?? 'arm64');
  }, [availableArchitectures, selectedArch]);

  useEffect(() => {
    if (selectedPlatform !== 'linux') return;

    const linuxDownloads = releaseData.downloads.linux[selectedArch];
    if (linuxDownloads[selectedLinuxFormat]) return;

    if (linuxDownloads.appImage) {
      setSelectedLinuxFormat('appImage');
      return;
    }

    if (linuxDownloads.deb) {
      setSelectedLinuxFormat('deb');
    }
  }, [releaseData.downloads.linux, selectedArch, selectedLinuxFormat, selectedPlatform]);

  const activePlatformConfig =
    PLATFORMS.find((platform) => platform.id === selectedPlatform) ?? PLATFORMS[0];

  return {
    activePlatformConfig,
    availableArchitectures,
    releaseData,
    selectedArch,
    selectedDownloadExt: selectedDownload?.ext?.toUpperCase() ?? 'N/A',
    selectedDownloadUrl: selectedDownload?.url ?? null,
    selectedLinuxFormat,
    selectedPlatform,
    setSelectedArch,
    setSelectedLinuxFormat,
    setSelectedPlatform,
  };
}

export { useDownloadPage };

