import { ORDER_STATE_CHIP } from '@/lib/order-actions';

export function OrderStateChip({ state }: { state: string }) {
  const normalized = state.toUpperCase();
  return (
    <span className={`status-chip ${ORDER_STATE_CHIP[normalized] ?? ''}`}>
      {normalized || '—'}
    </span>
  );
}
