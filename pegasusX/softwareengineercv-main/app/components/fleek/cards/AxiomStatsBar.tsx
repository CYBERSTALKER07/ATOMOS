'use client';

import type { FleekStat } from '@/app/data/fleekPageContent';
import { getAxiomStats } from '@/app/data/fleekPageContent';
import { useLanguage } from '@/app/context/LanguageContext';

type AxiomStatsBarProps = {
  stats?: FleekStat[];
  partners?: string[];
};

export default function AxiomStatsBar({
  stats,
  partners,
}: AxiomStatsBarProps) {
  const { language } = useLanguage();
  const resolvedStats = stats ?? getAxiomStats(language);
  const resolvedPartners =
    partners ??
    (language === 'ru'
      ? ['Поставщик', 'Склад', 'Водитель', 'Ритейлер']
      : ['Supplier', 'Warehouse', 'Driver', 'Retailer']);

  return (
    <div className="axiom-bar">
      <div className="axiom-bar__partners">
        {resolvedPartners.map((p) => (
          <span key={p} className="axiom-bar__partner">{p}</span>
        ))}
      </div>
      <div className="axiom-bar__stats">
        {resolvedStats.map((s) => (
          <div key={s.label} className="axiom-bar__stat">
            <p className="axiom-bar__stat-label">{s.label}</p>
            <p className="axiom-bar__stat-value">{s.value}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
