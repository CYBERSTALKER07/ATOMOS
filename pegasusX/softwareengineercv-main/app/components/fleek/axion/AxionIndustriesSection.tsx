'use client';

import Link from 'next/link';
import { useState } from 'react';
import type { AxionIndustryCard } from '@/app/data/axionSectionContent';
import AxionIndustryIcon from './AxionIndustryIcon';

type AxionIndustriesSectionProps = {
  eyebrow?: string;
  title?: string;
  description?: string;
  items: AxionIndustryCard[];
};

export default function AxionIndustriesSection({
  eyebrow = '/ INDUSTRIES WE SERVE',
  title = 'Tailored logistics for every business role',
  description = 'Supplier, warehouse, factory, driver, retailer, and gate — one order truth across portal, mobile, and desktop.',
  items,
}: AxionIndustriesSectionProps) {
  const [index, setIndex] = useState(0);
  const visible = items.slice(index, index + 4);
  const canPrev = index > 0;
  const canNext = index + 4 < items.length;

  return (
    <section className="axion-section axion-industries" id="fleek-section-03">
      <div className="axion-industries__head">
        <div>
          <p className="axion-eyebrow">{eyebrow}</p>
          <h2 className="axion-section__title">{title}</h2>
        </div>
        <div className="axion-industries__controls">
          <p className="axion-section__subtitle">{description}</p>
          <div className="axion-carousel-nav">
            <button
              type="button"
              className="axion-carousel-nav__btn"
              disabled={!canPrev}
              onClick={() => setIndex((i) => Math.max(0, i - 1))}
              aria-label="Previous industries"
            >
              ←
            </button>
            <button
              type="button"
              className="axion-carousel-nav__btn"
              disabled={!canNext}
              onClick={() => setIndex((i) => i + 1)}
              aria-label="Next industries"
            >
              →
            </button>
          </div>
        </div>
      </div>
      <div className="axion-industries__row">
        {visible.map((item) => {
          const inner = (
            <>
              <span className="axion-industry-card__icon">
                <AxionIndustryIcon type={item.icon} />
              </span>
              <h3 className="axion-industry-card__title">{item.title}</h3>
              {item.description ? (
                <p className="axion-industry-card__desc">{item.description}</p>
              ) : null}
            </>
          );

          if (item.href) {
            return (
              <Link
                key={item.title}
                href={item.href}
                className={`axion-industry-card ${item.highlight ? 'is-highlight' : ''}`}
              >
                {inner}
              </Link>
            );
          }

          return (
            <article
              key={item.title}
              className={`axion-industry-card ${item.highlight ? 'is-highlight' : ''}`}
            >
              {inner}
            </article>
          );
        })}
      </div>
    </section>
  );
}
