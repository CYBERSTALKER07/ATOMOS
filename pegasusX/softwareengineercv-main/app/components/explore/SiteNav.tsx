'use client';

import PillNav from '@/app/components/PillNav';
import { getMegaNavCategories } from '@/app/data/megaNavigation';
import { useLanguage } from '@/app/context/LanguageContext';


type SiteNavProps = {
  activeHref?: string;
};

export default function SiteNav({ activeHref }: SiteNavProps) {
  const { t } = useLanguage();

  return (
    <PillNav
      logo=""
      logoAlt="Pegasus Logo"
      showMenuButton={true}
      categories={getMegaNavCategories(t)}
      items={[]}
      activeHref={activeHref}
      baseColor="#000000"
      pillColor="#ffffff"
      hoveredPillTextColor="#ffffff"
      pillTextColor="#000000"
    />
  );
}
