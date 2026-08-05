'use client';

import SiteNav from '@/app/components/explore/SiteNav';

type FleekNavProps = {
  activeHref?: string;
};

/** Unified site chrome — same PillNav + mega menu as the home page. */
export default function FleekNav({ activeHref }: FleekNavProps) {
  return <SiteNav activeHref={activeHref} />;
}
