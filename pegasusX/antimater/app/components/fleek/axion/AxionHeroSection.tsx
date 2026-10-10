'use client';

import SubpageHero from '@/app/components/SubpageHero';

type AxionHeroSectionProps = {
 title: string;
 summary: string;
 primaryHref?: string;
 primaryLabel?: string;
 secondaryHref?: string;
 secondaryLabel?: string;
};

export default function AxionHeroSection({
 title,
 summary,
 primaryHref = '/join',
 primaryLabel = 'REQUEST DEMO',
 secondaryHref = '/platform',
 secondaryLabel = 'EXPLORE PLATFORM',
}: AxionHeroSectionProps) {
 return (
 <SubpageHero
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

