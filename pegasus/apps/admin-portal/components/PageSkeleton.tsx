'use client';

/**
 * M3 Page-level loading skeleton.
 * Used by loading.tsx files across all routes.
 */
export default function PageSkeleton({
  variant = 'dashboard',
}: {
  variant?: 'dashboard' | 'table' | 'form' | 'map';
}) {
  if (variant === 'map') {
    return (
      <div className="page-skeleton" style={{ padding: '0', height: '100%' }}>
        <div className="skeleton" style={{ height: '100%', minHeight: '100vh', borderRadius: 0 }} />
      </div>
    );
  }

  if (variant === 'form') {
    return (
      <div className="page-skeleton" style={{ maxWidth: 640, margin: '0 auto' }}>
        <div className="skeleton skeleton-header" />
        <div className="skeleton" style={{ height: 14, width: '60%', borderRadius: 6 }} />
        <div className="flex flex-col gap-4 mt-4">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex flex-col gap-2">
              <div className="skeleton" style={{ height: 12, width: 100, borderRadius: 4 }} />
              <div className="skeleton" style={{ height: 48, borderRadius: 4 }} />
            </div>
          ))}
        </div>
        <div className="skeleton" style={{ height: 40, width: 140, borderRadius: 9999, marginTop: 16 }} />
      </div>
    );
  }

  if (variant === 'table') {
    return (
      <div className="page-skeleton">
        <div className="skeleton skeleton-header" />
        <div className="skeleton" style={{ height: 14, width: '45%', borderRadius: 6 }} />
        <div className="skeleton-kpi-grid mt-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="skeleton skeleton-kpi" />
          ))}
        </div>
        <div className="skeleton skeleton-table mt-2" />
      </div>
    );
  }

  // Default: dashboard
  return (
    <div className="page-skeleton">
      <div className="skeleton skeleton-header" />
      <div className="skeleton" style={{ height: 14, width: '50%', borderRadius: 6 }} />
      <div className="skeleton-kpi-grid mt-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="skeleton skeleton-kpi" />
        ))}
      </div>
      <div className="skeleton" style={{ height: 280, borderRadius: 12 }} />
      <div className="skeleton skeleton-table" />
    </div>
  );
}
