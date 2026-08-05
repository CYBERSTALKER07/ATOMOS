'use client';

import type { FleekStat } from '@/app/data/fleekPageContent';
import { DEFAULT_AXIOM_STATS } from '@/app/data/fleekPageContent';

type AxiomStatsBarProps = {
  stats?: FleekStat[];
  partners?: string[];
};

export default function AxiomStatsBar({
  stats = DEFAULT_AXIOM_STATS,
  partners = ['Supplier', 'Warehouse', 'Driver', 'Retailer'],
}: AxiomStatsBarProps) {
  return (
    <div className="axiom-bar">
      <div className="axiom-bar__partners">
        {partners.map((p) => (
          <span key={p} className="axiom-bar__partner">{p}</span>
        ))}
      </div>
      <div className="axiom-bar__stats">
        {stats.map((s) => (
          <div key={s.label} className="axiom-bar__stat">
            <p className="axiom-bar__stat-label">{s.label}</p>
            <p className="axiom-bar__stat-value">{s.value}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
