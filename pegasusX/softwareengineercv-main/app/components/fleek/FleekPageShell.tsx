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
 * Full width container so heroes span edge-to-edge across the viewport.
 */
export default function FleekPageShell({ activeHref, children }: FleekPageShellProps) {
  return (
    <main className="fleek-docs min-h-screen w-full bg-[#F8FAFC] text-zinc-900 dark:bg-black dark:text-white transition-colors duration-200 overflow-x-hidden">
      <FleekNav activeHref={activeHref} />
      <div className="w-full pt-[4.5rem] md:pt-20">
        <div className="w-full">{children}</div>
      </div>
      <Footer />
    </main>
  );
}
