import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';

export const metadata: Metadata = pageMetadata({
  title: 'Join Pegasus',
  description:
    'Apply to join Pegasus or request a partner conversation — help build the logistics operating system for supplier-led networks.',
  path: '/join',
});

export default function JoinLayout({ children }: { children: React.ReactNode }) {
  return children;
}
