"use client";

export default function PortalError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <div className="p-8 max-w-lg mx-auto space-y-4">
      <h1 className="md-typescale-headline-medium">Portal error</h1>
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">{error.message}</p>
      <button type="button" className="md-btn md-btn-filled" onClick={reset}>
        Reload section
      </button>
    </div>
  );
}
