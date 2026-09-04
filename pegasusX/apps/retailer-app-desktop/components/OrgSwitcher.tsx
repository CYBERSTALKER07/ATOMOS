"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { Building2, ChevronDown, Loader2 } from "lucide-react";
import type { RetailerMembershipDTO } from "@pegasusx/types";
import { listMemberships, readActiveOrgId, switchOrg } from "@/lib/multi-org-auth";
import { getRetailerId } from "@/lib/retailer-profile";

/**
 * C1.3 in-session org switcher. Hidden when only one membership.
 * On switch: clear-on-switch contract + full page reload into new org.
 */
export function OrgSwitcher() {
  const t = usePortalT();
  const [items, setItems] = useState<RetailerMembershipDTO[]>([]);
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentId, setCurrentId] = useState<string>("");

  const load = useCallback(async () => {
    try {
      const ms = await listMemberships();
      const active = ms.filter((m) => m.is_active);
      setItems(active);
      const rid = readActiveOrgId() || getRetailerId() || "";
      setCurrentId(rid);
    } catch {
      setItems([]);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (items.length <= 1) {
    return null;
  }

  const current =
    items.find((m) => m.retailer_id === currentId) || items[0];

  async function onSwitch(retailerId: string) {
    if (retailerId === currentId || busy) return;
    setBusy(true);
    setError(null);
    try {
      await switchOrg(retailerId);
      setOpen(false);
      // Hard reload so POS/cart/assist remount with new JWT scope.
      window.location.href = "/dashboard";
    } catch (e) {
      setError(e instanceof Error ? e.message : t("retailer_desktop.residual.text.switch_failed"));
      setBusy(false);
    }
  }

  return (
    <div className="relative px-2.5 pb-2">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2 rounded-lg border px-2.5 py-2 text-left text-sm transition hover:bg-black/5"
        style={{ borderColor: "var(--desk-border)" }}
        aria-expanded={open}
        aria-haspopup="listbox"
      >
        <Building2 size={16} className="shrink-0 opacity-70" />
        <span className="min-w-0 flex-1 truncate font-medium">
          {current?.name?.trim() || current?.retailer_id || "Organization"}
        </span>
        {busy ? (
          <Loader2 size={14} className="animate-spin shrink-0" />
        ) : (
          <ChevronDown size={14} className="shrink-0 opacity-60" />
        )}
      </button>
      {open ? (
        <ul
          className="absolute left-2.5 right-2.5 z-50 mt-1 max-h-56 overflow-auto rounded-lg border bg-white py-1 shadow-lg dark:bg-zinc-900"
          style={{ borderColor: "var(--desk-border)" }}
          role="listbox"
        >
          {items.map((m) => (
            <li key={`${m.retailer_id}-${m.user_id}`}>
              <button
                type="button"
                role="option"
                aria-selected={m.retailer_id === currentId}
                disabled={busy}
                className="flex w-full flex-col px-3 py-2 text-left text-sm hover:bg-black/5 disabled:opacity-50"
                onClick={() => void onSwitch(m.retailer_id)}
              >
                <span className="font-medium">
                  {m.name?.trim() || m.retailer_id}
                </span>
                <span className="text-xs text-muted-foreground">
                  {m.retailer_role}
                  {m.retailer_id === currentId ? " · current" : ""}
                </span>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
      {error ? (
        <p className="mt-1 px-1 text-xs text-red-600" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  );
}
