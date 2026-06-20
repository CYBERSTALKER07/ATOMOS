'use client';

import { OrderKebabMenu } from './OrderKebabMenu';
import { OrderStateChip } from './OrderStateChip';
import { orderActionFlags } from '@/lib/order-actions';

export type OrderOpsCardProps = {
  orderId: string;
  retailerName: string;
  state: string;
  amountLabel: string;
  meta?: string;
  badge?: string;
  index?: number;
  disabled?: boolean;
  onOpenDetail: () => void;
  onDelay?: () => void;
  onReject?: () => void;
  showOpsMenu?: boolean;
  delayLabel?: string;
  rejectLabel?: string;
  canDelayOverride?: boolean;
  canRejectOverride?: boolean;
};

export function OrderOpsCard({
  orderId,
  retailerName,
  state,
  amountLabel,
  meta,
  badge,
  index = 0,
  disabled = false,
  onOpenDetail,
  onDelay,
  onReject,
  showOpsMenu = true,
  delayLabel,
  rejectLabel,
  canDelayOverride,
  canRejectOverride,
}: OrderOpsCardProps) {
  const flags = orderActionFlags(state);
  const canDelay = canDelayOverride ?? flags.canDelay;
  const canReject = canRejectOverride ?? flags.canReject;

  return (
    <article
      className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-4 hover:border-[var(--primary)]/30 transition-colors cursor-pointer"
      onDoubleClick={(e: React.MouseEvent) => {
        e.preventDefault();
        onOpenDetail();
      }}
      onClick={() => onOpenDetail()}
    >
      <div className="flex items-start gap-3">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <h3 className="font-medium truncate">{retailerName || 'Unknown retailer'}</h3>
            <OrderStateChip state={state} />
            {badge ? (
              <span className="text-[10px] uppercase tracking-wider px-2 py-0.5 rounded-full bg-[var(--default)] text-[var(--muted)]">
                {badge}
              </span>
            ) : null}
          </div>
          <p className="text-xs font-mono text-[var(--muted)] mt-1 truncate">{orderId}</p>
          {meta ? <p className="text-xs text-[var(--muted)] mt-1">{meta}</p> : null}
        </div>
        <div className="flex items-start gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
          <div className="text-right mr-1">
            <p className="text-sm font-mono tabular-nums">{amountLabel}</p>
          </div>
          {showOpsMenu ? (
            <OrderKebabMenu
              disabled={disabled}
              canDelay={canDelay}
              canReject={canReject}
              onViewDetails={onOpenDetail}
              onDelay={onDelay}
              onReject={onReject}
              delayLabel={delayLabel}
              rejectLabel={rejectLabel}
            />
          ) : null}
        </div>
      </div>
    </article>
  );
}
