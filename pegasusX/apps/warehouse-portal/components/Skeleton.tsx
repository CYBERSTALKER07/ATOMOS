import type { CSSProperties } from 'react';

export function Skeleton({ className = '', style }: { className?: string; style?: CSSProperties }) {
  return <div aria-hidden="true" className={`md-skeleton ${className}`} style={style} />;
}

export function SkeletonText({ lines = 3, className = '' }: { lines?: number; className?: string }) {
  return (
    <div className={className}>
      {Array.from({ length: lines }).map((_, i) => (
        <div
          key={i}
          className="md-skeleton md-skeleton-text"
          style={i === lines - 1 ? { width: '60%' } : undefined}
        />
      ))}
    </div>
  );
}

export function SkeletonCard({ className = '' }: { className?: string }) {
  return <div className={`md-skeleton md-skeleton-card ${className}`} />;
}

export function PageSkeleton({
  variant = "dashboard",
}: {
  variant?: "dashboard" | "table" | "form";
}) {
  if (variant === "form") {
    return (
      <div aria-hidden="true" className="page-skeleton" style={{ maxWidth: 640 }}>
        <Skeleton className="md-skeleton-title" style={{ height: 32, width: 200, borderRadius: 8 }} />
        <Skeleton style={{ height: 14, width: "60%", borderRadius: 6 }} />
        <div className="flex flex-col gap-4 mt-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex flex-col gap-2">
              <Skeleton style={{ height: 12, width: 100, borderRadius: 4 }} />
              <Skeleton style={{ height: 48, borderRadius: 8 }} />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (variant === "table") {
    return (
      <div aria-hidden="true" className="page-skeleton">
        <div className="skeleton-header md-skeleton" />
        <div className="skeleton-kpi-grid">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="skeleton-kpi md-skeleton" />
          ))}
        </div>
        <div className="skeleton-table md-skeleton" />
      </div>
    );
  }

  return (
    <div aria-hidden="true" className="page-skeleton">
      <div className="skeleton-header md-skeleton" />
      <div className="skeleton-kpi-grid">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="skeleton-kpi md-skeleton" />
        ))}
      </div>
      <Skeleton style={{ height: 280, borderRadius: 12 }} />
    </div>
  );
}
