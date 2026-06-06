"use client";

type ListToolbarProps = {
  page: number;
  pageCount: number;
  totalLabel: string;
  onPrev: () => void;
  onNext: () => void;
  onExport?: () => void;
  exportLabel?: string;
  exportDisabled?: boolean;
};

export function ListToolbar({
  page,
  pageCount,
  totalLabel,
  onPrev,
  onNext,
  onExport,
  exportLabel = "Export CSV",
  exportDisabled = false,
}: ListToolbarProps) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <p className="text-sm text-[var(--muted)]">{totalLabel}</p>
      <div className="flex items-center gap-2">
        {onExport ? (
          <button
            type="button"
            className="button--secondary inline-flex h-9 items-center rounded-full px-4 text-xs font-semibold uppercase tracking-[0.12em]"
            disabled={exportDisabled}
            onClick={onExport}
          >
            {exportLabel}
          </button>
        ) : null}
        <button
          type="button"
          className="button--secondary inline-flex h-9 items-center rounded-full px-4 text-xs font-semibold uppercase tracking-[0.12em]"
          disabled={page <= 0}
          onClick={onPrev}
        >
          Previous
        </button>
        <span className="text-xs font-semibold uppercase tracking-[0.14em] text-[var(--muted)] px-2">
          Page {page + 1} / {pageCount}
        </span>
        <button
          type="button"
          className="button--secondary inline-flex h-9 items-center rounded-full px-4 text-xs font-semibold uppercase tracking-[0.12em]"
          disabled={page >= pageCount - 1}
          onClick={onNext}
        >
          Next
        </button>
      </div>
    </div>
  );
}
