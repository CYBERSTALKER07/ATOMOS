"use client";

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const safeMsg =
    process.env.NODE_ENV === "production"
      ? "An unexpected error occurred in the retailer shell."
      : error.message || "Unexpected error";

  return (
    <html lang="en">
      <body className="flex min-h-screen items-center justify-center bg-slate-50 p-6 text-center text-slate-900">
        <div className="w-full max-w-md rounded-2xl border bg-white p-8 shadow-sm space-y-4">
          <h2 className="text-lg font-bold text-red-600">Application Error</h2>
          <p className="text-xs text-slate-600">{safeMsg}</p>
          <button
            type="button"
            onClick={() => reset()}
            className="rounded-lg bg-slate-900 px-4 py-2 text-xs font-semibold text-white hover:bg-slate-800"
          >
            Reload Session
          </button>
        </div>
      </body>
    </html>
  );
}
