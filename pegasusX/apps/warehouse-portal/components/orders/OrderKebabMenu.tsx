'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useRef, useState } from 'react';
import Icon from '@/components/Icon';

type OrderKebabMenuProps = {
  onViewDetails: () => void;
  onProposeDate?: () => void;
  onReject?: () => void;
  canProposeDate?: boolean;
  canReject?: boolean;
  proposeDateLabel?: string;
  rejectLabel?: string;
  disabled?: boolean;
};

export function OrderKebabMenu({
  onViewDetails,
  onProposeDate,
  onReject,
  canProposeDate = false,
  canReject = false,
  proposeDateLabel = 'Propose new date',
  rejectLabel = 'Cancel order',
  disabled = false,
}: OrderKebabMenuProps) {
  const t = usePortalT();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, [open]);

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        aria-label={t("warehouse_portal.orders.order_kebab_menu.text.order_actions")}
        disabled={disabled}
        className="desk-icon-btn"
        onClick={(e) => {
          e.stopPropagation();
          e.preventDefault();
          setOpen((v) => !v);
        }}
      >
        <Icon name="more_vert" size={20} />
      </button>
      {open ? (
        <div
          className="md-menu"
          style={{ right: 0, top: '100%', marginTop: 4, minWidth: 200 }}
          role="menu"
        >
          <button
            type="button"
            role="menuitem"
            className="md-menu-item"
            onClick={(e) => {
              e.stopPropagation();
              setOpen(false);
              onViewDetails();
            }}
          >
            View details
          </button>
          {onProposeDate ? (
            <button
              type="button"
              role="menuitem"
              disabled={!canProposeDate}
              title={!canProposeDate ? 'Propose date is not available for this order state' : undefined}
              className="md-menu-item disabled:opacity-40 disabled:cursor-not-allowed"
              style={{ color: canProposeDate ? 'var(--warning)' : undefined }}
              onClick={(e) => {
                e.stopPropagation();
                if (!canProposeDate) return;
                setOpen(false);
                onProposeDate();
              }}
            >
              {proposeDateLabel}
            </button>
          ) : null}
          {onReject ? (
            <button
              type="button"
              role="menuitem"
              disabled={!canReject}
              title={!canReject ? 'Cancel is not available for this order state' : undefined}
              className="md-menu-item disabled:opacity-40 disabled:cursor-not-allowed"
              style={{ color: canReject ? 'var(--danger)' : undefined }}
              onClick={(e) => {
                e.stopPropagation();
                if (!canReject) return;
                setOpen(false);
                onReject();
              }}
            >
              {rejectLabel}
            </button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
