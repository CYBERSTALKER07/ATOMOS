import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';

export const metadata: Metadata = pageMetadata({
  title: 'Roles',
  description:
    'Explore Pegasus by business role — supplier, warehouse, retailer, finance, driver, factory, and gate — with planning, dispatch, tracking, and payment flows mapped to each team.',
  path: '/roles',
});

export default function RolesLayout({ children }: { children: React.ReactNode }) {
  return children;
}
