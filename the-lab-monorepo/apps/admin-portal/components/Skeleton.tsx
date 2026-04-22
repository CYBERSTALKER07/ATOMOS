export function Skeleton({ className = '', style }: { className?: string; style?: React.CSSProperties }) {
  return <div className={`md-skeleton ${className}`} style={style} />;
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

/** Full-page loading skeleton matching common dashboard layouts */
export function PageSkeleton() {
  return (
    <div className="p-6 space-y-6 md-animate-in">
      <div className="md-skeleton md-skeleton-title" />
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <SkeletonCard />
        <SkeletonCard />
        <SkeletonCard />
      </div>
      <div className="space-y-1">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="md-skeleton md-skeleton-row" />
        ))}
      </div>
    </div>
  );
}
