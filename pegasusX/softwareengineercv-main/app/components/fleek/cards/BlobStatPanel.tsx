'use client';

import type { FleekBlobStat } from '@/app/data/fleekPageContent';
import { DEFAULT_BLOB_STATS } from '@/app/data/fleekPageContent';

type BlobStatPanelProps = {
  stats?: FleekBlobStat[];
};

export default function BlobStatPanel({ stats = DEFAULT_BLOB_STATS }: BlobStatPanelProps) {
  return (
    <div className="blob-panel">
      <nav className="blob-panel__nav" aria-label="Industries">
        {['STARTUPS', 'WAREHOUSE', 'FLEET', 'RETAIL'].map((item) => (
          <span key={item} className="blob-panel__nav-item">{item}</span>
        ))}
      </nav>
      <div className="blob-panel__stage" aria-hidden>
        <div className="blob-panel__blob blob-panel__blob--main" />
        <div className="blob-panel__blob blob-panel__blob--sub" />
      </div>
      <div className="blob-panel__callouts">
        {stats.map((stat, i) => (
          <div
            key={stat.value}
            className={`blob-panel__callout ${stat.highlight ? 'is-highlight' : ''} blob-panel__callout--${i}`}
          >
            <span className="blob-panel__value">{stat.value}</span>
            <p className="blob-panel__label">{stat.label}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
