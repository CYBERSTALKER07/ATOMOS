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
      <p className="md-typescale-body-small text-[var(--color-md-outline)]">{totalLabel}</p>
      <div className="flex items-center gap-2">
        {onExport ? (
          <button
            type="button"
            className="md-btn md-btn-outlined text-sm"
            disabled={exportDisabled}
            onClick={onExport}
          >
            {exportLabel}
          </button>
        ) : null}
        <button type="button" className="md-btn md-btn-outlined text-sm" disabled={page <= 0} onClick={onPrev}>
          Previous
        </button>
        <span className="md-typescale-label-medium px-2">
          Page {page + 1} / {pageCount}
        </span>
        <button
          type="button"
          className="md-btn md-btn-outlined text-sm"
          disabled={page >= pageCount - 1}
          onClick={onNext}
        >
          Next
        </button>
      </div>
    </div>
  );
}
