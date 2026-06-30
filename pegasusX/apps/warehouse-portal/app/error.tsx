"use client";

export default function WarehousePortalError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <main className="min-h-screen flex items-center justify-center p-8">
      <div className="md-card p-8 max-w-lg space-y-4">
        <h1 className="text-xl font-light">Warehouse portal error</h1>
        <p className="text-sm opacity-70">{error.message}</p>
        <button type="button" className="px-4 py-2 rounded-lg button--primary" onClick={reset}>
          Retry
        </button>
      </div>
    </main>
  );
}
