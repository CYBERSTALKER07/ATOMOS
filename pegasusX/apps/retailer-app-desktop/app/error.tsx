"use client";

export default function RetailerDesktopError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <main className="min-h-screen flex items-center justify-center p-8">
      <div className="md-card p-8 max-w-lg space-y-4">
        <h1 className="md-typescale-headline-medium">Retailer desktop error</h1>
        <p className="md-typescale-body-medium text-[var(--color-md-outline)]">{error.message}</p>
        <button type="button" className="md-btn md-btn-filled" onClick={reset}>
          Retry
        </button>
      </div>
    </main>
  );
}
