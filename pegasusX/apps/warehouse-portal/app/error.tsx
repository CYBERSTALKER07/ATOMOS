"use client";

import { usePortalT } from "@/lib/i18n";
export default function WarehousePortalError({ error, reset }: { error: Error; reset: () => void }) {
  const t = usePortalT();
  return (
    <main className="min-h-screen flex items-center justify-center p-8">
      <div className="md-card p-8 max-w-lg space-y-4">
        <h1 className="text-xl font-light">{t("warehouse_portal.error.text.warehouse_portal_error")}</h1>
        <p className="text-sm opacity-70">{error.message}</p>
        <button type="button" className="px-4 py-2 rounded-lg button--primary" onClick={reset}>
          Retry
        </button>
      </div>
    </main>
  );
}
