'use client';

import { useEffect, useRef } from 'react';

type OrderProposeDateDialogProps = {
  open: boolean;
  proposedDate: string;
  onProposedDateChange: (value: string) => void;
  reason: string;
  onReasonChange: (value: string) => void;
  submitting?: boolean;
  onConfirm: () => void;
  onClose: () => void;
  title?: string;
};

export function OrderProposeDateDialog({
  open,
  proposedDate,
  onProposedDateChange,
  reason,
  onReasonChange,
  submitting = false,
  onConfirm,
  onClose,
  title = 'Propose new delivery date',
}: OrderProposeDateDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const el = dialogRef.current;
    if (!el) return;
    if (open && !el.open) el.showModal();
    if (!open && el.open) el.close();
  }, [open]);

  if (!open) return null;

  const canSubmit = Boolean(proposedDate && reason.trim());

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
          if (canSubmit) onConfirm();
        }}
      >
        <div>
          <h2 className="text-lg font-semibold">{title}</h2>
          <p className="text-sm text-[var(--muted)] mt-1">
            Pick a new delivery date and reason. The retailer is notified and can accept or reject.
          </p>
        </div>
        <label className="portal-field">
          <span className="portal-label">Proposed delivery date</span>
          <input
            type="date"
            value={proposedDate}
            onChange={(e) => onProposedDateChange(e.target.value)}
            className="portal-input"
            required
          />
        </label>
        <label className="portal-field">
          <span className="portal-label">Reason</span>
          <textarea
            value={reason}
            onChange={(e) => onReasonChange(e.target.value)}
            placeholder="Required — explain why the date is changing"
            rows={3}
            className="portal-input min-h-[88px] py-2"
            required
          />
        </label>
        <div className="flex justify-end gap-2">
          <button type="button" className="portal-btn portal-btn--outline" onClick={onClose} disabled={submitting}>
            Cancel
          </button>
          <button
            type="submit"
            className="portal-btn portal-btn--primary disabled:opacity-50"
            disabled={submitting || !canSubmit}
          >
            {submitting ? 'Sending…' : 'Send proposal'}
          </button>
        </div>
      </form>
    </dialog>
  );
}
