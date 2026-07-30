import { supplierWarehouseOps } from '@/lib/supplier-warehouse-ops';

interface OrderOpsActionsProps {
  canWarehouseMutate: boolean;
  flags: { canDelay: boolean; canReject: boolean };
  acting: boolean;
  proposedDate: string;
  setProposedDate: (date: string) => void;
  reason: string;
  setReason: (reason: string) => void;
  onDelay: () => void;
  onReject: () => void;
}

export function OrderOpsActions({
  canWarehouseMutate,
  flags,
  acting,
  proposedDate,
  setProposedDate,
  reason,
  setReason,
  onDelay,
  onReject,
}: OrderOpsActionsProps) {
  if (!canWarehouseMutate || (!flags.canDelay && !flags.canReject)) return null;

  return (
    <div className="md-card p-5 space-y-4">
      <p className="md-typescale-title-small">Warehouse admin actions</p>
      {flags.canDelay ? (
        <label className="block text-sm">
          <span className="text-[var(--color-md-outline)]">New delivery date</span>
          <input
            type="date"
            value={proposedDate}
            onChange={(e) => setProposedDate(e.target.value)}
            className="md-input-outlined w-full px-3 py-2 mt-1"
            disabled={acting}
          />
        </label>
      ) : null}
      <textarea
        className="md-input-outlined w-full px-3 py-2 min-h-[80px]"
        placeholder="Reason (required)"
        value={reason}
        onChange={(e) => setReason(e.target.value)}
        disabled={acting}
      />
      <div className="flex flex-wrap gap-2">
        {flags.canDelay ? (
          <button
            type="button"
            className="md-btn md-btn-tonal px-4 py-2"
            disabled={acting || !proposedDate || !reason.trim()}
            onClick={onDelay}
          >
            Delay delivery
          </button>
        ) : null}
        {flags.canReject ? (
          <button
            type="button"
            className="md-btn md-btn-outlined px-4 py-2 text-[var(--color-md-error)]"
            disabled={acting}
            onClick={onReject}
          >
            Cancel order
          </button>
        ) : null}
      </div>
    </div>
  );
}
