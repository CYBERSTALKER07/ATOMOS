"use client";

import React from "react";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <html lang="en">
      <body className="flex min-h-screen items-center justify-center bg-gray-50 p-6 text-center text-gray-900">
        <div className="w-full max-w-md rounded-2xl border bg-white p-8 shadow-sm space-y-4">
          <h2 className="text-lg font-bold text-red-600">Global Application Error</h2>
          <p className="text-xs text-gray-600">
            A critical error occurred in the platform root shell: {error.message}
          </p>
          <button
            type="button"
            onClick={() => reset()}
            className="rounded-lg bg-indigo-600 px-4 py-2 text-xs font-semibold text-white hover:bg-indigo-700"
          >
            Reset Session
          </button>
        </div>
      </body>
    </html>
  );
}
