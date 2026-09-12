'use client';

import { memo } from 'react';
import SiteNav from '@/app/components/explore/SiteNav';

type FleekNavProps = {
  activeHref?: string;
};

/** Unified site chrome — same PillNav + mega menu as the home page. */
function FleekNav({ activeHref }: FleekNavProps) {
  return <SiteNav activeHref={activeHref} />;
}

export default memo(FleekNav);
