import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';

export const metadata: Metadata = pageMetadata({
  title: 'Projects',
  description:
    'Pegasus platform modules and reference implementations — dispatch engine, fleet maps, payment integrity, and role-specific apps.',
  path: '/projects',
});

export default function ProjectsLayout({ children }: { children: React.ReactNode }) {
  return children;
}
