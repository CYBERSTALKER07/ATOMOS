import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';

export const metadata: Metadata = pageMetadata({
  title: 'Desktop Apps',
  description:
    'Pegasus Tauri desktop apps for retailer, supplier, warehouse, and factory teams — native speed with offline cache, print, and deep links.',
  path: '/desktop-apps',
});

export default function DesktopAppsLayout({ children }: { children: React.ReactNode }) {
  return children;
}
