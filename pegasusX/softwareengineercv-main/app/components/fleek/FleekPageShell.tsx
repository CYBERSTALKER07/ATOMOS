'use client';

import type { ReactNode } from 'react';
import dynamic from 'next/dynamic';
import FleekNav from './FleekNav';

const Footer = dynamic(() => import('@/app/components/Footer'), { ssr: false });

type FleekPageShellProps = {
  activeHref?: string;
  children: ReactNode;
};

/**
 * Page chrome stays mounted independently of form state in `children`.
 * FleekNav is memoized so focus/typing in forms won't restart nav GSAP.
 */
export default function FleekPageShell({ activeHref, children }: FleekPageShellProps) {
  return (
    <main className="fleek-docs min-h-screen bg-black text-white">
      <FleekNav activeHref={activeHref} />
      <div className="axion-page pt-[4.5rem] md:pt-20 o9-page-wrap">
        <div className="axion-page__main">{children}</div>
      </div>
      <Footer />
    </main>
  );
}
