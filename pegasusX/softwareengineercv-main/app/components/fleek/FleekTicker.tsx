'use client';

import type { FleekTickerItem } from '@/app/data/fleekPageContent';

type FleekTickerProps = {
  items: FleekTickerItem[];
};

export default function FleekTicker({ items }: FleekTickerProps) {
  const text = items.map((i) => i.text).join(' //////// ');

  return (
    <div className="fleek-ticker" aria-label="Live network updates">
      <div className="fleek-ticker__track">
        <span>{text}</span>
        <span aria-hidden>{text}</span>
      </div>
    </div>
  );
}
