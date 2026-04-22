'use client';

import { useEffect } from 'react';

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('[AdminPortal] Unhandled error:', error);
  }, [error]);

  return (
    <div
      className="min-h-screen flex items-center justify-center p-8"
      style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}
    >
      <div className="max-w-md w-full text-center space-y-4">
        <div className="text-4xl font-mono font-bold">SYSTEM FAULT</div>
        <p className="text-sm font-mono opacity-80">{error.message || 'An unexpected error occurred.'}</p>
        {error.digest && (
          <p className="text-xs font-mono opacity-60">Digest: {error.digest}</p>
        )}
        <button
          onClick={reset}
          className="mt-4 px-6 py-2.5 rounded font-bold text-sm"
          style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}
        >
          Retry
        </button>
      </div>
    </div>
  );
}
