import React from 'react';

interface ProposeDelayDialogProps {
  open: boolean;
  proposedDate: string;
  reason: string;
  submitting: boolean;
  onProposedDateChange: (date: string) => void;
  onReasonChange: (reason: string) => void;
  onClose: () => void;
  onConfirm: () => void;
}

export function ProposeDelayDialog({
  open,
  proposedDate,
  reason,
  submitting,
  onProposedDateChange,
  onReasonChange,
  onClose,
  onConfirm,
}: ProposeDelayDialogProps) {
  if (!open) return null;

  return (
    <dialog
      open
      className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-5 backdrop:bg-black/40 max-w-md w-[calc(100%-2rem)] fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50"
    >
      <div className="space-y-4">
        <div>
          <h2 className="text-lg font-semibold">Delay delivery</h2>
          <p className="text-sm text-[var(--muted)] mt-1">
            Choose a new delivery date. The retailer is notified and can accept or reject.
          </p>
        </div>
        <label className="block text-sm">
          <span className="text-[var(--muted)]">New delivery date</span>
          <input
            type="date"
            value={proposedDate}
            onChange={(e) => onProposedDateChange(e.target.value)}
            className="mt-1 w-full px-3 py-2 rounded-lg border text-sm md-input-outlined"
          />
        </label>
        <textarea
          value={reason}
          onChange={(e) => onReasonChange(e.target.value)}
          placeholder="Reason (required)"
          rows={3}
          className="w-full px-3 py-2 rounded-lg border text-sm md-input-outlined"
        />
        <div className="flex justify-end gap-2">
          <button
            type="button"
            className="md-btn md-btn-outlined px-4 py-2"
            onClick={onClose}
          >
            Cancel
          </button>
          <button
            type="button"
            className="md-btn md-btn-tonal px-4 py-2 disabled:opacity-50"
            disabled={submitting || !proposedDate || !reason.trim()}
            onClick={onConfirm}
          >
            {submitting ? 'Working…' : 'Propose new date'}
          </button>
        </div>
      </div>
    </dialog>
  );
}
