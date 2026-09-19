'use client';

import type { ReactNode } from 'react';
import SubpageHero from '@/app/components/SubpageHero';
import { O9ProofStrip } from '@/app/components/page-sections/o9/O9Hero';
import type { ProofItem } from '@/app/data/topicTypes';
import { DEFAULT_PROOF } from '@/app/data/topicContent/helpers';

type O9HeroSplitProps = {
 categoryLabel: string;
 categoryHref: string;
 title: string;
 summary: string;
 badge?: string;
 primaryHref?: string;
 primaryLabel?: string;
 secondaryHref?: string;
 secondaryLabel?: string;
 imageSrc?: string;
 imageAlt?: string;
 visual?: ReactNode;
 proofItems?: ProofItem[];
 showProofStrip?: boolean;
};

export default function O9HeroSplit({
 categoryLabel,
 categoryHref,
 title,
 summary,
 badge,
 primaryHref = '/join',
 primaryLabel = 'REQUEST DEMO',
 secondaryHref,
 secondaryLabel = 'EXPLORE STACK',
 proofItems = DEFAULT_PROOF,
 showProofStrip = true,
}: O9HeroSplitProps) {
 return (
 <div className="w-full">
 <SubpageHero
 categoryLabel={categoryLabel}
 categoryHref={categoryHref}
 title={title}
 summary={summary}
 badge={badge}
 primaryCta={{
 label: primaryLabel,
 href: primaryHref,
 }}
 secondaryCta={{
 label: secondaryLabel,
 href: secondaryHref || categoryHref || '/platform',
 }}
 breadcrumb={{
 categoryLabel,
 categoryHref,
 currentPage: title,
 }}
 />
 {showProofStrip ? (
 <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 mt-6">
 <O9ProofStrip items={proofItems} />
 </div>
 ) : null}
 </div>
 );
}

