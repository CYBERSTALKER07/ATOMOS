'use client';

import type { FleekBlobStat } from '@/app/data/fleekPageContent';
import { getBlobStats } from '@/app/data/fleekPageContent';
import { useLanguage } from '@/app/context/LanguageContext';

type BlobStatPanelProps = {
  stats?: FleekBlobStat[];
};

export default function BlobStatPanel({ stats }: BlobStatPanelProps) {
  const { language } = useLanguage();
  const resolved = stats ?? getBlobStats(language);
  const nav =
    language === 'ru'
      ? ['СТАРТАПЫ', 'СКЛАД', 'АВТОПАРК', 'РИТЕЙЛ']
      : ['STARTUPS', 'WAREHOUSE', 'FLEET', 'RETAIL'];

  return (
    <div className="blob-panel">
      <nav className="blob-panel__nav" aria-label={language === 'ru' ? 'Отрасли' : 'Industries'}>
        {nav.map((item) => (
          <span key={item} className="blob-panel__nav-item">{item}</span>
        ))}
      </nav>
      <div className="blob-panel__stage" aria-hidden>
        <div className="blob-panel__blob blob-panel__blob--main" />
        <div className="blob-panel__blob blob-panel__blob--sub" />
      </div>
      <div className="blob-panel__callouts">
        {resolved.map((stat, i) => (
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
