import type { Metadata } from 'next';
import { pageMetadata } from '@/app/lib/seo';
import DemoShell from './DemoShell';

export async function generateMetadata(): Promise<Metadata> {
  return pageMetadata({
    title: 'Demo portal',
    description: 'Interactive Pegasus persona demos — not indexed.',
    path: '/demo',
    noIndex: true,
  });
}

export default function Layout({ children }: { children: React.ReactNode }) {
  return <DemoShell>{children}</DemoShell>;
}
