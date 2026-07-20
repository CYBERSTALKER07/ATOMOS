"use client";

import { useCallback, useEffect, useState } from "react";
import {
  checkDesktopUpdate,
  desktopStoreListingUrl,
  installPendingDesktopUpdate,
  isDesktopStoreBuild,
  isTauri,
  openDesktopStoreListing,
  type DesktopUpdateInfo,
} from "@pegasusx/desktop-bridge";

/**
 * Desktop update bootstrap:
 * - enterprise: CDN Tauri updater toast
 * - store (Microsoft Store / Mac App Store): no CDN OTA; open store listing
 */
export function EnterpriseDesktopUpdateBootstrap() {
  const [update, setUpdate] = useState<DesktopUpdateInfo | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [dismissed, setDismissed] = useState(false);
  const store = isDesktopStoreBuild();

  useEffect(() => {
    if (!isTauri() || store) return;
    let cancelled = false;
    void checkDesktopUpdate()
      .then((info) => {
        if (!cancelled && info) setUpdate(info);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [store]);

  const onInstall = useCallback(async () => {
    setBusy(true);
    setError(null);
    if (store) {
      const ok = await openDesktopStoreListing();
      if (!ok) {
        setError("Store listing URL not configured. Set NEXT_PUBLIC_DESKTOP_STORE_URL.");
        setBusy(false);
      }
      return;
    }
    const ok = await installPendingDesktopUpdate();
    if (!ok) {
      setError("Update failed. Try again or reinstall from the website.");
      setBusy(false);
    }
  }, [store]);

  // Store builds: no CDN toast (store owns updates). Banner is client-policy only.
  if (store) return null;
  if (!update || dismissed) return null;

  return (
    <div
      role="status"
      className="fixed bottom-4 right-4 z-[100] max-w-sm rounded-2xl border border-orange-200 bg-orange-50 px-4 py-3 text-orange-950 shadow-lg dark:border-orange-900/50 dark:bg-orange-950/90 dark:text-orange-50"
    >
      <p className="text-sm font-medium">Update available</p>
      <p className="mt-1 text-xs opacity-90">
        Version {update.version}
        {update.currentVersion ? ` (you have ${update.currentVersion})` : ""}.
        {update.body ? ` ${update.body}` : " Signed package from the enterprise CDN."}
      </p>
      {error && <p className="mt-1 text-xs text-red-700 dark:text-red-300">{error}</p>}
      <div className="mt-3 flex gap-2">
        <button
          type="button"
          disabled={busy}
          onClick={() => void onInstall()}
          className="rounded-lg bg-orange-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-60"
        >
          {busy ? "Installing…" : "Update now"}
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={() => setDismissed(true)}
          className="rounded-lg px-3 py-1.5 text-xs font-medium opacity-80 hover:opacity-100"
        >
          Later
        </button>
      </div>
    </div>
  );
}

/** Optional helper for store ClientPolicyBanner "Open store" actions. */
export async function openConfiguredDesktopStore(): Promise<boolean> {
  return openDesktopStoreListing(desktopStoreListingUrl());
}
