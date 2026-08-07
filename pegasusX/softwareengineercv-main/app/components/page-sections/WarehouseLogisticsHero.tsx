'use client';

import { useLanguage } from '@/app/context/LanguageContext';

import { useState } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { SITE_IMAGES } from '@/app/lib/siteAssets';

type WarehouseLogisticsHeroProps = {
  categoryLabel: string;
  categoryHref: string;
  title: string;
  summary: string;
};

const TRANSPORT_TABS = [
  { id: 'trucks', label: 'Trucks', count: '31,081', icon: 'truck' },
  { id: 'local', label: 'Local', count: '215,076', icon: 'local' },
  { id: 'trains', label: 'Trains', count: '5,053', icon: 'train' },
  { id: 'airplanes', label: 'Airplanes', count: '1,875', icon: 'plane' },
] as const;

const ROUTE_CARDS = [
  { type: 'FCL - 20ST', from: 'Shanghai', to: 'London', price: 1050 },
  { type: 'LCL - 40ST', from: 'Rotterdam', to: 'Mumbai', price: 980 },
  { type: 'FCL - 20ST', from: 'Singapore', to: 'Dubai', price: 1120 },
  { type: 'LCL - 40ST', from: 'Hong Kong', to: 'New York', price: 1340 },
  { type: 'FCL - 20ST', from: 'Busan', to: 'São Paulo', price: 1185 },
  { type: 'LCL - 40ST', from: 'Shanghai', to: 'London', price: 1050 },
] as const;

function TabIcon({ type }: { type: string }) {
  if (type === 'truck') {
    return (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden>
        <path
          d="M3 17h1M6 17h2M15 17h2M19 17h1M5 17V9h10v8M15 9l3 4h2v4"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
    );
  }
  if (type === 'local') {
    return (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden>
        <path d="M12 21s6-5.2 6-10a6 6 0 10-12 0c0 4.8 6 10 6 10z" stroke="currentColor" strokeWidth="1.5" />
        <circle cx="12" cy="11" r="2" stroke="currentColor" strokeWidth="1.5" />
      </svg>
    );
  }
  if (type === 'train') {
    return (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden>
        <rect x="4" y="5" width="16" height="12" rx="2" stroke="currentColor" strokeWidth="1.5" />
        <path d="M4 13h16M8 21v-2M16 21v-2" stroke="currentColor" strokeWidth="1.5" />
      </svg>
    );
  }
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden>
      <path
        d="M3 12h5l2-4h4l1 4h5l-1.5 2H14l-1 4h-2l-1-4H4.5L3 12z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function StarRating() {
  return (
    <div className="flex items-center gap-2">
      <div className="flex gap-0.5" aria-hidden>
        {Array.from({ length: 5 }).map((_, i) => (
          <svg key={i} width="16" height="16" viewBox="0 0 16 16" className="warehouse-logistics-hero__star">
            <path d="M8 1.2l1.55 3.14 3.45.5-2.5 2.43.59 3.44L8 8.77l-3.09 1.64.59-3.44-2.5-2.43 3.45-.5L8 1.2z" />
          </svg>
        ))}
      </div>
      <span className="text-sm text-white/70">
        4.87 based on 146,824 reviews
        <svg className="ml-1 inline-block opacity-60" width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden>
          <path d="M3 9l6-6M9 3h-4M9 3v4" stroke="currentColor" strokeWidth="1.25" strokeLinecap="round" />
        </svg>
      </span>
    </div>
  );
}

export default function WarehouseLogisticsHero({
  categoryLabel,
  categoryHref,
  title,
  summary,
}: WarehouseLogisticsHeroProps) {
  const { language } = useLanguage();

  const [activeTab, setActiveTab] = useState<string>('trucks');

  return (
    <header className="warehouse-logistics-hero">
      <div className="warehouse-logistics-hero__bg">
        <Image
          src={SITE_IMAGES.containerShip}
          alt=""
          fill
          priority
          className="object-cover object-right-bottom"
          sizes="100vw"
        />
        <div className="warehouse-logistics-hero__gradient" aria-hidden />
      </div>

      <div className="warehouse-logistics-hero__content">
        <nav
          aria-label="Breadcrumb"
          className="mb-4 flex flex-wrap items-center gap-2 font-mono text-[10px] uppercase tracking-widest text-white/45"
        >
          <Link href="/" className="min-h-11 inline-flex items-center hover:text-white transition-colors">
            Home
          </Link>
          <span aria-hidden>/</span>
          <Link href={categoryHref} className="min-h-11 inline-flex items-center hover:text-white transition-colors">
            {categoryLabel}
          </Link>
          <span aria-hidden>/</span>
          <span className="text-white/70" aria-current="page">{title}</span>
        </nav>

        <div className="warehouse-logistics-hero__copy">
          <p className="font-mono text-[10px] uppercase tracking-[0.22em] text-white/45">
            01 · {categoryLabel}
          </p>
          <h1 className="warehouse-logistics-hero__title mt-3">
            Minimize costs.
            <br />
            Transport goods
          </h1>
          <p className="warehouse-logistics-hero__summary mt-5 max-w-lg text-base leading-relaxed text-white/65 md:text-lg">
            {summary}
          </p>
          <div className="mt-6">
            <StarRating />
          </div>
          <Link href="/join" className="warehouse-logistics-hero__cta mt-8">
            Request quote
          </Link>
        </div>

        <div className="warehouse-logistics-hero__panel">
          <div className="warehouse-logistics-hero__toolbar">
            <div className="warehouse-logistics-hero__tabs" role="tablist" aria-label="Transport modes">
              {TRANSPORT_TABS.map((tab) => (
                <button
                  key={tab.id}
                  type="button"
                  role="tab"
                  aria-selected={activeTab === tab.id}
                  className={`warehouse-logistics-hero__tab ${activeTab === tab.id ? 'is-active' : ''}`}
                  onClick={() => setActiveTab(tab.id)}
                >
                  <TabIcon type={tab.icon} />
                  <span>{tab.label}</span>
                  <span className="warehouse-logistics-hero__tab-count">({tab.count})</span>
                </button>
              ))}
            </div>

            <div className="warehouse-logistics-hero__search">
              <div className="warehouse-logistics-hero__field">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden>
                  <circle cx="12" cy="10" r="3" stroke="currentColor" strokeWidth="1.5" />
                  <path d="M12 13v3M9 21h6" stroke="currentColor" strokeWidth="1.5" />
                </svg>
                <span className="text-white/40">Source</span>
              </div>
              <div className="warehouse-logistics-hero__field">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden>
                  <circle cx="12" cy="10" r="3" stroke="currentColor" strokeWidth="1.5" />
                  <path d="M12 13v3M9 21h6" stroke="currentColor" strokeWidth="1.5" />
                </svg>
                <span className="text-white/40">Target</span>
              </div>
              <div className="warehouse-logistics-hero__field">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden>
                  <rect x="4" y="5" width="16" height="16" rx="2" stroke="currentColor" strokeWidth="1.5" />
                  <path d="M8 3v4M16 3v4M4 11h16" stroke="currentColor" strokeWidth="1.5" />
                </svg>
                <span className="text-white/55">{language === 'ru' ? '07 июля 2025' : 'July 07, 2025'}</span>
              </div>
              <div className="warehouse-logistics-hero__field">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden>
                  <path
                    d="M4 8h16v12H4zM8 8V6a2 2 0 012-2h4a2 2 0 012 2v2"
                    stroke="currentColor"
                    strokeWidth="1.5"
                  />
                </svg>
                <span className="text-white/55">100kg / 0.5m³</span>
              </div>
            </div>
          </div>

          <div className="warehouse-logistics-hero__routes">
            {ROUTE_CARDS.map((route, index) => (
              <div key={`${route.type}-${route.from}-${index}`} className="warehouse-logistics-hero__route-card">
                <p className="warehouse-logistics-hero__route-type">{route.type}</p>
                <div className="warehouse-logistics-hero__route-lane">
                  <span className="warehouse-logistics-hero__route-city">{route.from}</span>
                  <span className="warehouse-logistics-hero__route-arrow" aria-hidden>→</span>
                  <span className="warehouse-logistics-hero__route-city">{route.to}</span>
                </div>
                <p className="warehouse-logistics-hero__route-price">
                  USD {route.price.toLocaleString()} <span>starting from</span>
                </p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </header>
  );
}
