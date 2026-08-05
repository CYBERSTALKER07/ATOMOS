'use client';

import Link from 'next/link';
import type { ReactNode } from 'react';
import WireframeGlobe from './WireframeGlobe';
import AxiomStatsBar from './cards/AxiomStatsBar';

type FleekHeroSectionProps = {
  sectionNumber?: string;
  sectionTitle: string;
  title: string;
  summary: string;
  primaryHref?: string;
  primaryLabel?: string;
  secondaryHref?: string;
  secondaryLabel?: string;
  visual?: ReactNode;
  showStats?: boolean;
};

export default function FleekHeroSection({
  sectionNumber = '01',
  sectionTitle = 'DECENTRALIZED EDGE NETWORK',
  title,
  summary,
  primaryHref = '/join',
  primaryLabel = 'REQUEST DEMO',
  secondaryHref,
  secondaryLabel = 'STAY UPDATED',
  visual,
  showStats = true,
}: FleekHeroSectionProps) {
  return (
    <section id="fleek-section-01" className="fleek-hero" data-section={sectionNumber}>
      <div className="fleek-hero__grid">
        <div className="fleek-hero__copy">
          <p className="fleek-hero__index">
            <span className="fleek-hero__index-num">{sectionNumber}</span>
            {sectionTitle}
          </p>
          <h1 className="fleek-hero__title">{title}</h1>
          <p className="fleek-hero__summary">{summary}</p>
          <div className="fleek-hero__actions">
            <Link href={primaryHref} className="fleek-btn fleek-btn--ghost">
              {primaryLabel}
            </Link>
            {secondaryHref && secondaryLabel ? (
              <Link href={secondaryHref} className="fleek-btn fleek-btn--ghost">
                {secondaryLabel}
              </Link>
            ) : null}
          </div>
        </div>
        <div className="fleek-hero__visual">
          {visual ?? <WireframeGlobe />}
        </div>
      </div>
      {showStats ? <AxiomStatsBar /> : null}
      <div className="fleek-hero__connector" aria-hidden />
    </section>
  );
}
