'use client';

import { useState } from 'react';
import Link from 'next/link';

const BLOCKS = [
  {
    id: 'grow',
    label: 'Grow your network',
    sub: 'Onboard suppliers, warehouses, and retailers on one spine.',
    href: '/join',
    active: true,
  },
  {
    id: 'own',
    label: 'Own the dispatch',
    sub: 'Visual boards, gate seals, and fleet truth after departure.',
    href: '/roles/warehouse',
    active: false,
  },
] as const;

export default function PixelDualHero() {
  const [active, setActive] = useState('grow');

  return (
    <div className="pixel-dual-hero">
      <div className="pixel-dual-hero__grid-bg" aria-hidden />
      <div className="pixel-dual-hero__blocks">
        {BLOCKS.map((block) => {
          const isActive = active === block.id;
          return (
            <button
              key={block.id}
              type="button"
              className={`pixel-dual-hero__block ${isActive ? 'is-active' : ''}`}
              onClick={() => setActive(block.id)}
            >
              <span className="pixel-dual-hero__pixels" aria-hidden />
              <span className="pixel-dual-hero__label">{block.label}</span>
              <span className="pixel-dual-hero__chev" aria-hidden>››</span>
            </button>
          );
        })}
      </div>
      <p className="pixel-dual-hero__sub">
        {BLOCKS.find((b) => b.id === active)?.sub}
      </p>
      <Link href={BLOCKS.find((b) => b.id === active)?.href ?? '/join'} className="pixel-dual-hero__link">
        Continue →
      </Link>
    </div>
  );
}
