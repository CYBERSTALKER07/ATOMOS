'use client';

import type { FlowConfig } from '@/app/data/topicTypes';
import { FlowShell } from './FlowShell';

const LAYERS = [
  { label: 'Role apps', sub: 'Portal · Mobile · Desktop' },
  { label: 'Platform API', sub: 'Role-scoped routes' },
  { label: 'Shared record', sub: 'Transactional writes' },
  { label: 'Events + cache', sub: 'Events · cache' },
  { label: 'Live updates', sub: 'Instant screen sync' },
];

type Props = { config?: FlowConfig };

export default function TechStackFlow({ config }: Props) {
  const highlight = config?.highlightStep ?? 2;

  return (
    <FlowShell title="Technology stack">
      <div className="mx-auto max-w-md space-y-2">
        {LAYERS.map((layer, i) => (
          <div
            key={layer.label}
            className={`border px-4 py-3 transition-colors ${
              i === highlight ? 'border-white bg-white text-black' : 'border-white/25 text-white/70'
            }`}
          >
            <p className="font-mono text-xs uppercase tracking-wide">{layer.label}</p>
            <p className={`mt-1 text-xs ${i === highlight ? 'text-black/70' : 'text-white/40'}`}>{layer.sub}</p>
          </div>
        ))}
      </div>
    </FlowShell>
  );
}
