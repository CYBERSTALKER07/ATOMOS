"use client";

import React, { useEffect } from "react";
import { mapToHumanError } from "@/lib/errors";

export default function ErrorBoundary({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Log exception to console
    console.error("Route Error Caught:", error);
  }, [error]);

  const human = mapToHumanError(error);

  return (
    <div className="flex min-h-[400px] flex-col items-center justify-center p-6 text-center">
      <div className="w-full max-w-lg rounded-2xl border border-red-200 bg-white p-6 shadow-sm space-y-4">
        <div className="inline-flex h-12 w-12 items-center justify-center rounded-full bg-red-100 text-red-600 mx-auto">
          <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>

        <div>
          <h2 className="text-base font-bold text-gray-900">{human.title}</h2>
          <p className="mt-1 text-xs text-gray-600">{human.message}</p>
        </div>

        {human.actionGuidance && (
          <div className="rounded-lg bg-gray-50 p-3 text-left text-xs text-gray-700">
            <span className="font-semibold block text-gray-900 mb-1">Recommended Action:</span>
            {human.actionGuidance}
          </div>
        )}

        {error.digest && (
          <p className="font-mono text-[10px] text-gray-400">Digest: {error.digest}</p>
        )}

        <div className="flex justify-center gap-3 pt-2">
          <button
            type="button"
            onClick={() => reset()}
            className="rounded-lg bg-indigo-600 px-4 py-2 text-xs font-semibold text-white hover:bg-indigo-700 transition-colors"
          >
            Retry Action
          </button>
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="rounded-lg border px-4 py-2 text-xs font-semibold text-gray-700 hover:bg-gray-100 transition-colors"
          >
            Reload Page
          </button>
        </div>
      </div>
    </div>
  );
}
