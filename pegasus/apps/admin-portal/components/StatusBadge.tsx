import StatusChip from './StatusChip';
import type { OrderState } from '@pegasus/types/order';

export default function StatusBadge({ state, className = '' }: { state: string; className?: string }) {
  return <StatusChip status={state} className={className} />;
}

export type { OrderState };
