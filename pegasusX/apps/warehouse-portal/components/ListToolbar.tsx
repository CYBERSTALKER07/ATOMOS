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
    <div className="wh-list-toolbar">
      <p className="wh-list-toolbar-meta">{totalLabel}</p>
      <div className="wh-list-toolbar-actions">
        {onExport ? (
          <button
            type="button"
            className="portal-btn portal-btn--outline"
            disabled={exportDisabled}
            onClick={onExport}
          >
            {exportLabel}
          </button>
        ) : null}
        <button
          type="button"
          className="portal-btn portal-btn--outline"
          disabled={page <= 0}
          onClick={onPrev}
        >
          Previous
        </button>
        <span className="wh-page-indicator">
          {page + 1} / {pageCount}
        </span>
        <button
          type="button"
          className="portal-btn portal-btn--outline"
          disabled={page >= pageCount - 1}
          onClick={onNext}
        >
          Next
        </button>
      </div>
    </div>
  );
}
