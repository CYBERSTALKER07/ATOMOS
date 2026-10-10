'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useRef, useState } from 'react';
import Icon from '@/components/Icon';

type OrderKebabMenuProps = {
  onViewDetails: () => void;
  onDelay?: () => void;
  onReject?: () => void;
  onReassign?: () => void;
  canDelay?: boolean;
  canReject?: boolean;
  canReassign?: boolean;
  delayLabel?: string;
  rejectLabel?: string;
  reassignLabel?: string;
  disabled?: boolean;
};

export function OrderKebabMenu({
  onViewDetails,
  onDelay,
  onReject,
  onReassign,
  canDelay = false,
  canReject = false,
  canReassign = false,
  delayLabel = 'Delay delivery',
  rejectLabel = 'Reject order',
  reassignLabel = 'Reassign order',
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
        aria-label={t("supplier_portal.orders.order_kebab_menu.text.order_actions")}
        disabled={disabled}
        className="flex items-center justify-center w-11 h-11 rounded-lg hover:bg-[var(--default)] transition-colors disabled:opacity-40"
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
          className="absolute right-0 top-full z-20 mt-1 min-w-44 rounded-lg border border-[var(--border)] bg-[var(--surface)] py-1 shadow-lg"
          role="menu"
        >
          <button
            type="button"
            role="menuitem"
            className="w-full text-left px-3 py-2 text-sm hover:bg-[var(--default)]"
            onClick={(e) => {
              e.stopPropagation();
              setOpen(false);
              onViewDetails();
            }}
          >
            View details
          </button>
          {onDelay ? (
            <button
              type="button"
              role="menuitem"
              disabled={!canDelay}
              title={!canDelay ? 'Delay is only available for pending or loaded orders' : undefined}
              className="w-full text-left px-3 py-2 text-sm hover:bg-[var(--default)] disabled:opacity-40 disabled:cursor-not-allowed"
              style={{ color: canDelay ? 'var(--warning)' : undefined }}
              onClick={(e) => {
                e.stopPropagation();
                if (!canDelay) return;
                setOpen(false);
                onDelay();
              }}
            >
              {delayLabel}
            </button>
          ) : null}
          {onReassign ? (
            <button
              type="button"
              role="menuitem"
              disabled={!canReassign}
              title={!canReassign ? 'Reassign is only available before delivery completes' : undefined}
              className="w-full text-left px-3 py-2 text-sm hover:bg-[var(--default)] disabled:opacity-40 disabled:cursor-not-allowed"
              onClick={(e) => {
                e.stopPropagation();
                if (!canReassign) return;
                setOpen(false);
                onReassign();
              }}
            >
              {reassignLabel}
            </button>
          ) : null}
          {onReject ? (
            <button
              type="button"
              role="menuitem"
              disabled={!canReject}
              title={!canReject ? 'Reject is not available for this state' : undefined}
              className="w-full text-left px-3 py-2 text-sm hover:bg-[var(--default)] disabled:opacity-40 disabled:cursor-not-allowed"
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
