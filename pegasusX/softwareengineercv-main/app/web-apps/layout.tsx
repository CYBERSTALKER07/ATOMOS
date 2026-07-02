import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';

export const metadata: Metadata = pageMetadata({
  title: 'Web Apps',
  description:
    'Pegasus web portals for supplier ops, warehouse dispatch, retailer commerce, fleet telemetry, and treasury reconciliation.',
  path: '/web-apps',
});

export default function WebAppsLayout({ children }: { children: React.ReactNode }) {
  return children;
}
