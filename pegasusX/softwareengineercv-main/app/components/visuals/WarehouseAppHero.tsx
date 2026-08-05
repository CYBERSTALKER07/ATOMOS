'use client';

import Image from 'next/image';
import Link from 'next/link';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

const LOCATIONS = ['San Diego', 'Los Angeles', 'Phoenix', 'Dallas'] as const;

type WarehouseAppHeroProps = {
  categoryLabel: string;
  categoryHref: string;
  title: string;
  summary: string;
};

export default function WarehouseAppHero({
  categoryLabel,
  categoryHref,
  title,
  summary,
}: WarehouseAppHeroProps) {
  return (
    <header className="warehouse-app-hero -mx-4 md:-mx-[calc((100vw-100%)/2+1rem)]">
      <div className="warehouse-app-hero__stripes" aria-hidden />

      <div className="warehouse-app-hero__shell">
        <nav
          aria-label="Breadcrumb"
          className="mb-4 flex flex-wrap items-center gap-2 font-mono text-[10px] uppercase tracking-widest text-white/45"
        >
          <Link href="/" className="hover:text-white transition-colors">Home</Link>
          <span>/</span>
          <Link href={categoryHref} className="hover:text-white transition-colors">{categoryLabel}</Link>
          <span>/</span>
          <span className="text-white/70">{title}</span>
        </nav>

        <div className="warehouse-app-hero__topbar">
          <button type="button" className="warehouse-app-hero__icon-btn" aria-label="Back">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden>
              <path d="M14 6l-6 6 6 6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          </button>
          <div className="text-center">
            <p className="text-lg font-semibold tracking-tight">San Diego</p>
            <p className="text-xs text-white/45">California · {title}</p>
          </div>
          <button type="button" className="warehouse-app-hero__icon-btn" aria-label="Settings">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden>
              <circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth="1.5" />
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2" stroke="currentColor" strokeWidth="1.5" />
            </svg>
          </button>
        </div>

        <div className="warehouse-app-hero__locations">
          {LOCATIONS.map((loc, i) => (
            <button
              key={loc}
              type="button"
              className={`warehouse-app-hero__loc ${i === 0 ? 'is-active' : ''}`}
            >
              {i === 0 ? (
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" aria-hidden>
                  <path d="M12 21s6-5.2 6-10a6 6 0 10-12 0c0 4.8 6 10 6 10z" stroke="currentColor" strokeWidth="1.5" />
                  <circle cx="12" cy="11" r="2" stroke="currentColor" strokeWidth="1.5" />
                </svg>
              ) : null}
              <span>{loc}</span>
            </button>
          ))}
        </div>

        <div className="warehouse-app-hero__stage">
          <div className="warehouse-app-hero__truck-card">
            <div className="warehouse-app-hero__truck-visual">
              <Image
                src={SITE_IMAGES.truckTerminal}
                alt="Dispatch truck ready at loading bay"
                fill
                className="object-cover"
                sizes="(max-width: 768px) 100vw, 480px"
                priority
              />
            </div>
            <p className="warehouse-app-hero__truck-caption">{summary}</p>
          </div>

          <div className="warehouse-app-hero__boxes">
            {['BOX A', 'BOX B', 'BOX C'].map((box, i) => (
              <div key={box} className={`warehouse-app-hero__box ${i === 0 ? 'is-active' : ''}`}>
                {box}
              </div>
            ))}
          </div>
        </div>

        <div className="warehouse-app-hero__capacity">
          <span className="warehouse-app-hero__capacity-label">UP TO 8 TONE</span>
          <button type="button" className="warehouse-app-hero__capacity-add" aria-label="Add capacity">
            +
          </button>
        </div>

        <div className="warehouse-app-hero__actions">
          <Link href="/join" className="warehouse-app-hero__cta">Request quote</Link>
          <Link href="/demo/warehouse" className="warehouse-app-hero__cta-secondary">Open dispatch board</Link>
        </div>
      </div>
    </header>
  );
}
