import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';

export const metadata: Metadata = pageMetadata({
  title: 'Mobile Apps',
  description:
    'Pegasus native Android and iOS apps for drivers, retailers, suppliers, warehouse, factory, and payload gate teams.',
  path: '/mobile-apps',
});

export default function MobileAppsLayout({ children }: { children: React.ReactNode }) {
  return children;
}
