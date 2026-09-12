'use client';

import Image from 'next/image';
import Link from 'next/link';
import type { AxionSolutionCard } from '@/app/data/axionSectionContent';

type AxionSolutionsGridProps = {
  title?: string;
  subtitle?: string;
  items: AxionSolutionCard[];
  seeAllHref?: string;
  seeAllLabel?: string;
};

export default function AxionSolutionsGrid({
  title = 'Logistics Solutions',
  subtitle = 'Diverse needs of supplier-led networks — dispatch, fleet, treasury, and role apps on one spine.',
  items,
  seeAllHref = '/capabilities',
  seeAllLabel = 'See All',
}: AxionSolutionsGridProps) {
  const large = items.filter((i) => i.size === 'large').slice(0, 2);
  const small = items.filter((i) => i.size !== 'large').slice(0, 3);

  return (
    <section className="axion-section axion-solutions" id="fleek-section-02">
      <div className="axion-section__head">
        <div>
          <h2 className="axion-section__title">{title}</h2>
          <p className="axion-section__subtitle">{subtitle}</p>
        </div>
        <Link href={seeAllHref} className="axion-btn axion-btn--primary axion-btn--sm">
          {seeAllLabel}
        </Link>
      </div>
      <div className="axion-solutions__grid">
        {large.map((card) => (
          <Link key={card.title} href={card.href} className="axion-solution-card axion-solution-card--lg">
            <Image src={card.image} alt="" fill className="object-cover" sizes="50vw" />
            <div className="axion-solution-card__overlay" />
            <span className="axion-solution-card__label">{card.title}</span>
          </Link>
        ))}
        {small.map((card) => (
          <Link key={card.title} href={card.href} className="axion-solution-card axion-solution-card--sm">
            <Image src={card.image} alt="" fill className="object-cover" sizes="33vw" />
            <div className="axion-solution-card__overlay" />
            <span className="axion-solution-card__label">{card.title}</span>
          </Link>
        ))}
      </div>
    </section>
  );
}
