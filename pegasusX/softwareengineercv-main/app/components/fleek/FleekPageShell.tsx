'use client';

import type { ReactNode } from 'react';
import FleekNav from './FleekNav';
import Footer from '@/app/components/Footer';

type FleekPageShellProps = {
  activeHref?: string;
  children: ReactNode;
};

export default function FleekPageShell({ activeHref, children }: FleekPageShellProps) {
  return (
    <main className="fleek-docs min-h-screen">
      <FleekNav activeHref={activeHref} />
      <div className="axion-page">
        <div className="axion-page__main">{children}</div>
      </div>
      <Footer />
    </main>
  );
}
