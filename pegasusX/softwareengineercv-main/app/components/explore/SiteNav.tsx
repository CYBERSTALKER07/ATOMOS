'use client';

import PillNav from '@/app/components/PillNav';

type SiteNavProps = {
  activeHref?: string;
};

const SITE_NAV_ITEMS = [
  { label: 'Home', href: '/' },
  { label: 'About', href: '/#about' },
  { label: 'Solutions', href: '/solutions' },
  { label: 'Projects', href: '/projects' },
  { label: 'Demo', href: '/join' },
] as const;

export default function SiteNav({ activeHref }: SiteNavProps) {
  return (
    <PillNav
      logo=""
      logoAlt="Pegasus Logo"
      showMenuButton
      items={[...SITE_NAV_ITEMS]}
      activeHref={activeHref}
      baseColor="#000000"
      pillColor="#ffffff"
      hoveredPillTextColor="#ffffff"
      pillTextColor="#000000"
    />
  );
}
