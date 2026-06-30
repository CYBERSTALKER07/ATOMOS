"use client";

export default function RetailerDesktopError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <main className="min-h-screen flex items-center justify-center p-8">
      <div className="desk-card p-8 max-w-lg space-y-4">
        <h1 className="text-xl font-light">Retailer desktop error</h1>
        <p className="text-sm opacity-70">{error.message}</p>
        <button type="button" className="portal-btn portal-btn--primary" onClick={reset}>
          Retry
        </button>
      </div>
    </main>
  );
}
