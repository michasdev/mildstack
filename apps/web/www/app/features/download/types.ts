import type { ComponentType } from 'react';

export type PlatformId = 'macos' | 'windows' | 'linux';
export type ArchitectureId = 'x64' | 'arm64';
export type LinuxFormat = 'appImage' | 'deb';
export type AssetExt = 'dmg' | 'exe' | 'appimage' | 'deb';

export interface GitHubAsset {
  name: string;
  browser_download_url: string;
  download_count: number;
}

export interface GitHubRelease {
  tag_name: string;
  html_url: string;
  draft: boolean;
  published_at: string | null;
  assets: GitHubAsset[];
}

export interface DownloadAsset {
  name: string;
  url: string;
  ext: AssetExt;
  downloadCount: number;
}

export interface ReleaseDownloads {
  macos: Record<ArchitectureId, DownloadAsset | null>;
  windows: Record<ArchitectureId, DownloadAsset | null>;
  linux: Record<ArchitectureId, Record<LinuxFormat, DownloadAsset | null>>;
}

export interface DownloadPageData {
  versionTag: string;
  version: string;
  releaseUrl: string;
  publishedAt: string | null;
  downloads: ReleaseDownloads;
  fetchError?: string;
}

export interface PlatformDefinition {
  id: PlatformId;
  label: string;
  icon?: ComponentType<{ className?: string }>;
  iconAlt?: string;
  iconSrc?: string;
  architectureLabel: Record<ArchitectureId, string>;
}
