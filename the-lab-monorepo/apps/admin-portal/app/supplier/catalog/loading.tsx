function SkeletonBlock({ className = '' }: { className?: string }) {
  return (
    <div
      className={`rounded animate-pulse ${className}`.trim()}
      style={{ background: 'var(--surface)' }}
    />
  );
}

function SkeletonCard() {
  return (
    <div className="md-card md-card-elevated p-6 space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div className="space-y-2 flex-1">
          <SkeletonBlock className="h-4 w-40" />
          <SkeletonBlock className="h-3 w-64" />
        </div>
        <SkeletonBlock className="h-4 w-28" />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <SkeletonBlock className="h-3 w-24" />
          <SkeletonBlock className="h-11 w-full" />
        </div>
        <div className="space-y-2">
          <SkeletonBlock className="h-3 w-28" />
          <SkeletonBlock className="h-11 w-full" />
        </div>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="space-y-2">
          <SkeletonBlock className="h-3 w-20" />
          <SkeletonBlock className="h-11 w-full" />
        </div>
        <div className="space-y-2">
          <SkeletonBlock className="h-3 w-24" />
          <SkeletonBlock className="h-11 w-full" />
        </div>
      </div>
      <div className="space-y-2">
        <SkeletonBlock className="h-3 w-24" />
        <SkeletonBlock className="h-24 w-full" />
      </div>
      <SkeletonBlock className="h-24 w-full" />
      <SkeletonBlock className="h-14 w-full rounded-full" />
    </div>
  );
}

export default function Loading() {
  return (
    <div className="min-h-full p-6 md:p-8" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      <header className="mb-8 pb-4" style={{ borderBottom: '1px solid var(--border)' }}>
        <SkeletonBlock className="h-8 w-56" />
        <SkeletonBlock className="h-4 w-72 mt-3" />
      </header>

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-8 items-start">
        <SkeletonCard />

        <div className="space-y-8">
          <SkeletonCard />
          <div className="md-card md-card-elevated p-6 space-y-4">
            <SkeletonBlock className="h-4 w-40" />
            {Array.from({ length: 4 }).map((_, index) => (
              <div key={index} className="flex items-center justify-between gap-4 pb-2" style={{ borderBottom: '1px solid var(--border)' }}>
                <SkeletonBlock className="h-4 w-40" />
                <div className="flex items-center gap-3">
                  <SkeletonBlock className="h-5 w-16 rounded-full" />
                  <SkeletonBlock className="h-4 w-24" />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}