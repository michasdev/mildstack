import type { Route } from './+types/download';
import { DownloadPage } from '@/features/download/download';
import { loadDownloadPageData } from '@/features/download/lib/release';

export async function loader({}: Route.LoaderArgs) {
  return loadDownloadPageData();
}

export function meta({}: Route.MetaArgs): Route.MetaDescriptors {
  return [
    { title: 'MildStack Desktop Download' },
    {
      name: 'description',
      content:
        'Download the MildStack Desktop app for macOS, Windows, or Linux. MildStack CLI is included automatically.',
    },
  ];
}

export default function DownloadRoute({ loaderData }: Route.ComponentProps) {
  return <DownloadPage loaderData={loaderData} />;
}

