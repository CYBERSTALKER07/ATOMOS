'use client';

import { memo, useMemo } from 'react';
import PillNav, { type PillNavItem } from '@/app/components/PillNav';
import { getMegaNavCategories } from '@/app/data/megaNavigation';
import { useLanguage } from '@/app/context/LanguageContext';

type SiteNavProps = {
  activeHref?: string;
};

const EMPTY_ITEMS: PillNavItem[] = [];

function SiteNav({ activeHref }: SiteNavProps) {
  const { t, language } = useLanguage();

  // Rebuild only when language changes (t is stable per language via context).
  const categories = useMemo(() => getMegaNavCategories(t), [language, t]);

  return (
    <PillNav
      logo=""
      logoAlt="Pegasus Logo"
      showMenuButton={true}
      categories={categories}
      items={EMPTY_ITEMS}
      activeHref={activeHref}
      baseColor="#000000"
      pillColor="#ffffff"
      hoveredPillTextColor="#ffffff"
      pillTextColor="#000000"
    />
  );
}

export default memo(SiteNav);
