'use client';

import { motion } from 'framer-motion';
import { OrderKebabMenu } from './OrderKebabMenu';
import { OrderStateChip } from './OrderStateChip';
import { orderActionFlags } from '@/lib/order-actions';

export type OrderDetailOpenMode = 'single' | 'double';

export type OrderOpsCardProps = {
  orderId: string;
  retailerName: string;
  state: string;
  amountLabel: string;
  meta?: string;
  badge?: string;
  index?: number;
  disabled?: boolean;
  detailOpenMode?: OrderDetailOpenMode;
  onOpenDetail: () => void;
  onProposeDate?: () => void;
  onReject?: () => void;
  showOpsMenu?: boolean;
  showQuickActions?: boolean;
  proposeDateLabel?: string;
  rejectLabel?: string;
  canProposeDateOverride?: boolean;
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
  detailOpenMode = 'single',
  onOpenDetail,
  onProposeDate,
  onReject,
  showOpsMenu = true,
  showQuickActions = true,
  proposeDateLabel = 'Propose date',
  rejectLabel = 'Cancel order',
  canProposeDateOverride,
  canRejectOverride,
}: OrderOpsCardProps) {
  const flags = orderActionFlags(state);
  const canProposeDate = canProposeDateOverride ?? flags.canDelay;
  const canReject = canRejectOverride ?? flags.canReject;
  const hasQuickActions = showQuickActions && (onProposeDate || onReject);

  function handleCardClick() {
    if (detailOpenMode === 'single') onOpenDetail();
  }

  function handleCardDoubleClick(e: React.MouseEvent) {
    if (detailOpenMode === 'double') {
      e.preventDefault();
      onOpenDetail();
    }
  }

  return (
    <motion.article
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.03 }}
      className="wh-bay-panel wh-bay--ops wh-ops-card"
      onClick={handleCardClick}
      onDoubleClick={handleCardDoubleClick}
      title={detailOpenMode === 'double' ? 'Double-click to open order detail' : 'Click to open order detail'}
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
          <p className="wh-ops-card-id mt-1 truncate">{orderId}</p>
          {meta ? <p className="text-xs text-[var(--muted)] mt-1">{meta}</p> : null}
        </div>
        <div className="flex items-start gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
          <div className="text-right mr-1">
            <p className="wh-ops-card-amount">{amountLabel}</p>
          </div>
          {showOpsMenu ? (
            <OrderKebabMenu
              disabled={disabled}
              canProposeDate={canProposeDate}
              canReject={canReject}
              onViewDetails={onOpenDetail}
              onProposeDate={onProposeDate}
              onReject={onReject}
              proposeDateLabel={proposeDateLabel}
              rejectLabel={rejectLabel}
            />
          ) : null}
        </div>
      </div>

      {hasQuickActions ? (
        <div
          className="flex flex-wrap gap-2 mt-3 pt-3 border-t border-[var(--border)]"
          onClick={(e) => e.stopPropagation()}
        >
          {onProposeDate ? (
            <button
              type="button"
              className="portal-btn portal-btn--outline text-xs min-h-[36px] px-3"
              disabled={disabled || !canProposeDate}
              title={!canProposeDate ? 'Not available for this order state' : undefined}
              onClick={() => canProposeDate && onProposeDate()}
            >
              {proposeDateLabel}
            </button>
          ) : null}
          {onReject ? (
            <button
              type="button"
              className="portal-btn portal-btn--outline text-xs min-h-[36px] px-3"
              disabled={disabled || !canReject}
              style={{ color: canReject ? 'var(--danger)' : undefined }}
              title={!canReject ? 'Not available for this order state' : undefined}
              onClick={() => canReject && onReject()}
            >
              {rejectLabel}
            </button>
          ) : null}
        </div>
      ) : null}
    </motion.article>
  );
}
