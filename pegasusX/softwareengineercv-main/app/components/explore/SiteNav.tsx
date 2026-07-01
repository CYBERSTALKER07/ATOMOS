'use client';

import PillNav from '@/app/components/PillNav';
import { MEGA_NAV_CATEGORIES } from '@/app/data/megaNavigation';

type SiteNavProps = {
  activeHref?: string;
};

export default function SiteNav({ activeHref }: SiteNavProps) {
  return (
    <PillNav
      logo=""
      logoAlt="Pegasus Logo"
      showMenuButton={true}
      categories={MEGA_NAV_CATEGORIES}
      items={[]}
      activeHref={activeHref}
      baseColor="#000000"
      pillColor="#ffffff"
      hoveredPillTextColor="#ffffff"
      pillTextColor="#000000"
    />
  );
}
