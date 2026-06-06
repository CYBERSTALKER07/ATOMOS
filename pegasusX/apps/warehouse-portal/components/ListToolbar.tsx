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
            className="px-3 py-1.5 rounded-lg text-sm button--secondary"
            disabled={exportDisabled}
            onClick={onExport}
          >
            {exportLabel}
          </button>
        ) : null}
        <button type="button" className="px-3 py-1.5 rounded-lg text-sm button--secondary" disabled={page <= 0} onClick={onPrev}>
          Previous
        </button>
        <span className="text-sm px-2">
          Page {page + 1} / {pageCount}
        </span>
        <button
          type="button"
          className="px-3 py-1.5 rounded-lg text-sm button--secondary"
          disabled={page >= pageCount - 1}
          onClick={onNext}
        >
          Next
        </button>
      </div>
    </div>
  );
}
