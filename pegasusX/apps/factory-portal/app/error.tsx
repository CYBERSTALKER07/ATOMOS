"use client";

import { usePortalT } from "@/lib/i18n";
export default function FactoryPortalError({ error, reset }: { error: Error; reset: () => void }) {
  const t = usePortalT();
  return (
    <main className="min-h-screen flex items-center justify-center p-8">
      <div className="desk-card p-8 max-w-lg space-y-4">
        <h1 className="text-xl font-light">{t("factory_portal.error.text.factory_portal_error")}</h1>
        <p className="text-sm opacity-70">{error.message}</p>
        <button type="button" className="portal-btn portal-btn--primary" onClick={reset}>
          Retry
        </button>
      </div>
    </main>
  );
}
