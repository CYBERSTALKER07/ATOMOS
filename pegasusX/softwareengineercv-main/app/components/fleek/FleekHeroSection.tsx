'use client';

import SubpageHero from '@/app/components/SubpageHero';

type FleekHeroSectionProps = {
  sectionNumber?: string;
  sectionTitle?: string;
  title: string;
  summary: string;
  primaryHref?: string;
  primaryLabel?: string;
  secondaryHref?: string;
  secondaryLabel?: string;
};

export default function FleekHeroSection({
  sectionTitle = 'CONNECTED LOGISTICS NETWORK',
  title,
  summary,
  primaryHref = '/join',
  primaryLabel = 'REQUEST DEMO',
  secondaryHref = '/platform',
  secondaryLabel = 'STAY UPDATED',
}: FleekHeroSectionProps) {
  return (
    <SubpageHero
      categoryLabel={sectionTitle}
      title={title}
      summary={summary}
      primaryCta={{
        label: primaryLabel,
        href: primaryHref,
      }}
      secondaryCta={{
        label: secondaryLabel,
        href: secondaryHref,
      }}
    />
  );
}

