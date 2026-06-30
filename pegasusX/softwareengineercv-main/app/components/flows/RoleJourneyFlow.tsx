'use client';

import type { FlowConfig } from '@/app/data/topicTypes';
import { FlowShell } from './FlowShell';

const DEFAULT_STEPS = ['Intake', 'Act', 'Handoff', 'Confirm'];

type Props = { config?: FlowConfig };

export default function RoleJourneyFlow({ config }: Props) {
  const role = config?.roles?.[0] ?? 'Role';
  const steps = config?.roles?.slice(1) ?? DEFAULT_STEPS;
  const highlight = config?.highlightStep ?? 1;

  return (
    <FlowShell title={`${role} journey`}>
      <div className="space-y-3">
        {steps.map((step, i) => (
          <div
            key={step}
            className={`flex items-center gap-4 border-l-2 py-2 pl-4 ${
              i <= highlight ? 'border-white text-white' : 'border-white/20 text-white/40'
            }`}
          >
            <span className="font-mono text-xs uppercase">{String(i + 1).padStart(2, '0')}</span>
            <span className="text-sm">{step}</span>
          </div>
        ))}
      </div>
    </FlowShell>
  );
}
