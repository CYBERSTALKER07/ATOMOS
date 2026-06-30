'use client';

import type { FlowConfig } from '@/app/data/topicTypes';
import { FlowShell } from './FlowShell';

type Props = { config?: FlowConfig };

export default function AiAssistFlow({ config }: Props) {
  const showOverride = (config?.highlightStep ?? 1) >= 1;

  return (
    <FlowShell title="AI assist with human override">
      <div className="flex flex-col items-center gap-6 md:flex-row md:justify-center">
        <div className="border border-dashed border-white/40 px-6 py-4 text-center">
          <p className="font-mono text-xs uppercase text-white/50">Suggestion</p>
          <p className="mt-2 text-sm">Truck B · 94% fit</p>
        </div>
        <span className="font-mono text-white/30">→</span>
        <div
          className={`border px-6 py-4 text-center transition-colors ${
            showOverride ? 'border-white bg-white text-black' : 'border-white/30'
          }`}
        >
          <p className="font-mono text-xs uppercase opacity-60">Warehouse override</p>
          <p className="mt-2 text-sm font-semibold">Confirm load</p>
        </div>
      </div>
    </FlowShell>
  );
}
