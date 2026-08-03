"use client";

import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { Building2, Loader2 } from "lucide-react";
import type { RetailerMembershipDTO } from "@pegasusx/types";
import {
  clearStashedMemberships,
  listMemberships,
  loadStashedMemberships,
  readPendingOrgToken,
  selectOrg,
} from "@/lib/multi-org-auth";

/**
 * C1.3 org picker after multi-org login (PendingOrgSelect intermediate token).
 */
export default function SelectOrgPage() {
  const router = useRouter();
  const [items, setItems] = useState<RetailerMembershipDTO[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      if (!readPendingOrgToken()) {
        router.replace("/auth/login");
        return;
      }
      const stashed = loadStashedMemberships();
      if (stashed.length > 0) {
        setItems(stashed.filter((m) => m.is_active));
        setLoading(false);
        return;
      }
      const ms = await listMemberships();
      setItems(ms.filter((m) => m.is_active));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load organizations");
    } finally {
      setLoading(false);
    }
  }, [router]);

  useEffect(() => {
    void load();
  }, [load]);

  async function onSelect(retailerId: string) {
    setBusyId(retailerId);
    setError(null);
    try {
      const data = await selectOrg(retailerId);
      clearStashedMemberships();
      router.replace(data.is_configured === false ? "/setup/tax" : "/dashboard");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not select organization");
    } finally {
      setBusyId(null);
    }
  }

  return (
    <div className="mx-auto flex min-h-[70vh] max-w-md flex-col justify-center gap-6 px-4 py-10">
      <div className="text-center">
        <div
          className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl"
          style={{ background: "var(--desk-accent, #2563eb)", color: "#fff" }}
        >
          <Building2 size={22} />
        </div>
        <h1 className="text-xl font-semibold tracking-tight">Choose organization</h1>
        <p className="mt-1 text-sm text-muted-foreground">
          Your phone is linked to more than one retailer. Select where to work.
        </p>
      </div>

      {loading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="animate-spin" size={24} />
        </div>
      ) : items.length === 0 ? (
        <p className="text-center text-sm text-muted-foreground">
          No active memberships. Contact your manager or sign in again.
        </p>
      ) : (
        <ul className="flex flex-col gap-2">
          {items.map((m) => (
            <li key={`${m.retailer_id}-${m.user_id}`}>
              <button
                type="button"
                disabled={busyId !== null}
                onClick={() => void onSelect(m.retailer_id)}
                className="flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left transition hover:bg-black/5 disabled:opacity-60"
                style={{ borderColor: "var(--desk-border, #e5e7eb)" }}
              >
                <span>
                  <span className="block font-medium">
                    {m.name?.trim() || m.retailer_id}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {m.retailer_role}
                    {m.phone ? ` · ${m.phone}` : ""}
                  </span>
                </span>
                {busyId === m.retailer_id ? (
                  <Loader2 className="animate-spin" size={18} />
                ) : (
                  <span className="text-sm font-medium" style={{ color: "var(--desk-accent, #2563eb)" }}>
                    Continue
                  </span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}

      {error ? (
        <p className="text-center text-sm text-red-600" role="alert">
          {error}
        </p>
      ) : null}

      <button
        type="button"
        className="text-center text-sm text-muted-foreground underline"
        onClick={() => {
          clearStashedMemberships();
          if (typeof document !== "undefined") {
            document.cookie = "pegasus_retailer_pending_jwt=; Max-Age=0; path=/";
          }
          router.replace("/auth/login");
        }}
      >
        Back to sign in
      </button>
    </div>
  );
}
