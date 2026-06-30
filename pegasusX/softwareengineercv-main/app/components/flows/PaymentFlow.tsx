'use client';

import type { FlowConfig } from '@/app/data/topicTypes';
import { FlowShell, StepNode } from './FlowShell';

const STEPS = ['Checkout', 'Dispatch', 'Arrived', 'Collect', 'Treasury'];

type Props = { config?: FlowConfig };

export default function PaymentFlow({ config }: Props) {
  const highlight = config?.highlightStep ?? 3;

  return (
    <FlowShell title="Pay-at-delivery flow">
      <div className="flex flex-wrap justify-between gap-4">
        {STEPS.map((label, i) => (
          <StepNode key={label} label={label} index={i} active={i <= highlight} />
        ))}
      </div>
    </FlowShell>
  );
}
