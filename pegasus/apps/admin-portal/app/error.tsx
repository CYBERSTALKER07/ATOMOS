'use client';

import { useEffect } from 'react';
import EmptyState from '@/components/EmptyState';

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

  const body = error.message?.trim()
    ? error.message
    : 'An unexpected error blocked this supplier surface.';

  return (
    <div className="min-h-screen flex flex-col items-center justify-center px-6 py-10">
      <div className="w-full max-w-2xl">
        <EmptyState
          variant="error"
          headline="System Fault"
          body={body}
          action="Retry"
          onAction={reset}
        />
        {error.digest && (
          <p className="mt-4 text-center text-xs font-mono text-muted-foreground">
            Digest: {error.digest}
          </p>
        )}
      </div>
    </div>
  );
}
