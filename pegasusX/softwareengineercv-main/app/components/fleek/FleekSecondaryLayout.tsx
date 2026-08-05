'use client';

import type { ReactNode } from 'react';
import { AxionPageLayout } from '@/app/components/fleek/axion';
import FleekPageShell from '@/app/components/fleek/FleekPageShell';
import FleekDataSection from '@/app/components/fleek/FleekDataSection';

type FleekSecondaryLayoutProps = {
  activeHref?: string;
  sectionTitle?: string;
  title: string;
  summary: string;
  primaryHref?: string;
  primaryLabel?: string;
  secondaryHref?: string;
  secondaryLabel?: string;
  heroVisual?: ReactNode;
  heroImageSrc?: string;
  hubId?: string;
  stackFeatures?: string[];
  section06: ReactNode;
  dataExtra?: ReactNode;
  showStack?: boolean;
};

export default function FleekSecondaryLayout({
  activeHref,
  sectionTitle,
  title,
  summary,
  primaryHref = '/join',
  primaryLabel = 'Learn More',
  heroVisual,
  heroImageSrc,
  hubId,
  section06,
  dataExtra,
}: FleekSecondaryLayoutProps) {
  return (
    <FleekPageShell activeHref={activeHref}>
      <AxionPageLayout
        hero={{
          title,
          summary,
          primaryHref,
          primaryLabel,
          visual: heroVisual,
          imageSrc: heroImageSrc,
        }}
        solutions={{
          title: sectionTitle ? `${sectionTitle} solutions` : 'Logistics Solutions',
          subtitle: summary,
          seeAllHref: activeHref ?? '/capabilities',
        }}
        technology={{
          extra: (
            <>
              <FleekDataSection hubId={hubId} extra={dataExtra} />
            </>
          ),
        }}
        details={section06}
      />
    </FleekPageShell>
  );
}
