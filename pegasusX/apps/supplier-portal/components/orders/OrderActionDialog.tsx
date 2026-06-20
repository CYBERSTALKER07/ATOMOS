'use client';

import { useEffect, useRef } from 'react';

type OrderActionDialogProps = {
  open: boolean;
  title: string;
  description?: string;
  confirmLabel: string;
  destructive?: boolean;
  reason: string;
  onReasonChange: (value: string) => void;
  reasonRequired?: boolean;
  submitting?: boolean;
  onConfirm: () => void;
  onClose: () => void;
};

export function OrderActionDialog({
  open,
  title,
  description,
  confirmLabel,
  destructive = false,
  reason,
  onReasonChange,
  reasonRequired = false,
  submitting = false,
  onConfirm,
  onClose,
}: OrderActionDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const el = dialogRef.current;
    if (!el) return;
    if (open && !el.open) el.showModal();
    if (!open && el.open) el.close();
  }, [open]);

  if (!open) return null;

  return (
    <dialog
      ref={dialogRef}
      className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-0 backdrop:bg-black/40 max-w-md w-[calc(100%-2rem)]"
      onClose={onClose}
    >
      <form
        method="dialog"
        className="p-5 space-y-4"
        onSubmit={(e) => {
          e.preventDefault();
          onConfirm();
        }}
      >
        <div>
          <h2 className="text-lg font-semibold">{title}</h2>
          {description ? <p className="text-sm text-[var(--muted)] mt-1">{description}</p> : null}
        </div>
        <textarea
          value={reason}
          onChange={(e) => onReasonChange(e.target.value)}
          placeholder={reasonRequired ? 'Reason (required)' : 'Reason (optional)'}
          rows={3}
          className="w-full px-3 py-2 rounded-lg border text-sm"
          style={{
            background: 'var(--field-background)',
            borderColor: 'var(--field-border)',
            color: 'var(--field-foreground)',
          }}
          autoFocus
        />
        <div className="flex justify-end gap-2">
          <button type="button" className="button--secondary px-4 py-2 rounded-lg text-sm" onClick={onClose} disabled={submitting}>
            Cancel
          </button>
          <button
            type="submit"
            className={`${destructive ? 'button--danger' : 'portal-btn portal-btn--primary'} px-4 py-2 rounded-lg text-sm disabled:opacity-50`}
            disabled={submitting || (reasonRequired && !reason.trim())}
          >
            {submitting ? 'Working…' : confirmLabel}
          </button>
        </div>
      </form>
    </dialog>
  );
}
