import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';

export const metadata: Metadata = pageMetadata({
  title: 'Platform Overview',
  description:
    'Pegasus platform overview — logistics operating system architecture, modules, and deployment surfaces across six roles.',
  path: '/resume',
});

export default function ResumeLayout({ children }: { children: React.ReactNode }) {
  return children;
}
