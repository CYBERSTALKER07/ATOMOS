'use client';

import Link from 'next/link';
import ChamferButton from '@/app/components/ChamferButton';
import type { ProofItem } from '@/app/data/topicTypes';

type O9HeroProps = {
  categoryLabel: string;
  categoryHref: string;
  title: string;
  summary: string;
  badge?: string;
};

export function O9Hero({
  categoryLabel,
  categoryHref,
  title,
  summary,
  badge,
}: O9HeroProps) {
  return (
    <header className="docs-reveal max-w-4xl">
      <nav
        aria-label="Breadcrumb"
        className="mb-8 flex flex-wrap items-center gap-2 font-mono text-[10px] uppercase tracking-widest text-white/45"
      >
        <Link href="/" className="min-h-11 inline-flex items-center hover:text-white transition-colors duration-200">
          Home
        </Link>
        <span aria-hidden>/</span>
        <Link
          href={categoryHref}
          className="min-h-11 inline-flex items-center hover:text-white transition-colors duration-200"
        >
          {categoryLabel}
        </Link>
        <span aria-hidden>/</span>
        <span className="text-white/70" aria-current="page">
          {title.length > 40 ? `${title.slice(0, 40)}…` : title}
        </span>
      </nav>

      <p className="editorial-eyebrow">{categoryLabel}</p>
      <h1 className="docs-hero-title mt-4 text-4xl font-semibold tracking-tight md:text-5xl lg:text-6xl">
        {title}
        {badge ? (
          <span className="ml-3 align-middle font-mono text-xs tracking-widest text-white/55">
            [{badge}]
          </span>
        ) : null}
      </h1>
      <p className="docs-body mt-6 max-w-3xl text-lg leading-relaxed text-white/70 md:text-xl">{summary}</p>
      <div className="mt-8 flex flex-col gap-3 sm:flex-row">
        <ChamferButton href="/join" variant="fill">
          Request demo
        </ChamferButton>
        <ChamferButton href="/platform" variant="ghost">
          Take platform tour
        </ChamferButton>
      </div>
    </header>
  );
}

export function O9ProofStrip({ items }: { items: ProofItem[] }) {
  if (!items.length) return null;
  return (
    <div className="docs-proof docs-grain">
      <p className="border-b border-white/10 px-4 py-2.5 font-mono text-[10px] uppercase tracking-[0.2em] text-white/40">
        Built for supplier-led logistics networks
      </p>
      <div className="grid grid-cols-2 divide-x divide-y divide-white/10 md:grid-cols-4 md:divide-y-0">
        {items.map((item) => (
          <div key={item.label} className="px-4 py-5 md:px-5 md:py-6">
            <p className="font-mono text-[10px] uppercase tracking-widest text-white/45">{item.label}</p>
            <p className="mt-2 text-sm font-medium text-white/90 md:text-base">{item.value}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
