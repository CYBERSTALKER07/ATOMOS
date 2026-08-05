import type { Metadata } from 'next';
import SiteNav from '@/app/components/explore/SiteNav';
import { pageMetadata } from '@/app/lib/seo';

export const metadata: Metadata = pageMetadata({
  title: 'Admin',
  description: 'Internal Pegasus admin.',
  path: '/admin',
  noIndex: true,
});

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <SiteNav activeHref="/admin" />
      <div className="pt-[4.5rem] md:pt-20">{children}</div>
    </>
  );
}
