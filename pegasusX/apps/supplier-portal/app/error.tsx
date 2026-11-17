"use client";

import { usePortalT } from "@/lib/i18n";
export default function GlobalError({ error, reset }: { error: Error; reset: () => void }) {
  const t = usePortalT();
  return (
    <main className="min-h-screen flex items-center justify-center p-8">
      <div className="md-card p-8 max-w-lg space-y-4">
        <h1 className="md-typescale-headline-medium">{t("supplier_portal.error.text.something_went_wrong")}</h1>
        <p className="md-typescale-body-medium text-[var(--color-md-outline)]">{error.message}</p>
        <button type="button" className="md-btn md-btn-filled" onClick={reset}>
          Retry
        </button>
      </div>
    </main>
  );
}
