"use client";

import { useCallback, useEffect, useState } from "react";
import {
  Loader2,
  Package,
  ArrowLeftRight,
  Plus,
  AlertTriangle,
  ClipboardList,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";

type Balance = {
  location_id: string;
  stock_bin: string;
  sku: string;
  on_hand: number;
  reserved: number;
  available: number;
};

type Location = {
  location_id: string;
  name: string;
  is_primary: boolean;
};

export default function StockPage() {
  const [items, setItems] = useState<Balance[]>([]);
  const [locations, setLocations] = useState<Location[]>([]);
  const [locationId, setLocationId] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [banner, setBanner] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const [orderId, setOrderId] = useState("");
  const [adjustSku, setAdjustSku] = useState("");
  const [adjustDelta, setAdjustDelta] = useState("0");
  const [adjustBin, setAdjustBin] = useState("BACKROOM");
  const [xferSku, setXferSku] = useState("");
  const [xferQty, setXferQty] = useState("1");
  const [countSku, setCountSku] = useState("");
  const [countQty, setCountQty] = useState("0");
  const [countBin, setCountBin] = useState("FLOOR");
  const [countVersion, setCountVersion] = useState(0);
  const [countConflict, setCountConflict] = useState<string | null>(null);
  const [retailerRole, setRetailerRole] = useState("");

  const loadCountVersion = useCallback(async () => {
    if (!locationId) return;
    try {
      const res = await apiFetch(
        `/v1/retailer/stock/counts/version?location_id=${encodeURIComponent(locationId)}&stock_bin=${encodeURIComponent(countBin)}`,
      );
      if (res.status === 404) return;
      if (!res.ok) return;
      const json = (await res.json()) as { version?: number };
      setCountVersion(json.version ?? 0);
    } catch {
      /* offline count flag off or network */
    }
  }, [locationId, countBin]);

  const loadLocations = useCallback(async () => {
    try {
      const res = await apiFetch("/v1/retailer/locations");
      if (!res.ok) return;
      const json = (await res.json()) as {
        items?: Location[];
        active_location_id?: string;
      };
      setLocations(json.items ?? []);
      setLocationId((prev) => prev || json.active_location_id || json.items?.[0]?.location_id || "");
    } catch {
      /* ignore */
    }
  }, []);

  const loadStock = useCallback(async () => {
    if (!locationId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch(
        `/v1/retailer/stock?location_id=${encodeURIComponent(locationId)}`,
      );
      if (!res.ok) throw new Error(`load_failed_${res.status}`);
      const json = (await res.json()) as { items?: Balance[] };
      setItems(json.items ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load stock");
    } finally {
      setLoading(false);
    }
  }, [locationId]);

  useEffect(() => {
    void loadLocations();
  }, [loadLocations]);

  useEffect(() => {
    void loadStock();
  }, [loadStock]);

  useEffect(() => {
    void loadCountVersion();
  }, [loadCountVersion]);

  useEffect(() => {
    void (async () => {
      try {
        const res = await apiFetch("/v1/retailer/me");
        if (!res.ok) return;
        const json = (await res.json()) as { retailer_role?: string };
        setRetailerRole(json.retailer_role ?? "");
      } catch {
        /* ignore */
      }
    })();
  }, []);

  const receiveOrder = async () => {
    setBusy(true);
    setBanner(null);
    try {
      const res = await apiFetch("/v1/retailer/stock/receive-sessions", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `recv-${orderId}-${Date.now()}`,
        },
        body: JSON.stringify({
          order_id: orderId,
          location_id: locationId,
          confirm: true,
          stock_bin: "BACKROOM",
        }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(
          (json as { error?: string }).error || `receive_failed_${res.status}`,
        );
      }
      setBanner("Order received into BACKROOM stock");
      setOrderId("");
      await loadStock();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Receive failed");
    } finally {
      setBusy(false);
    }
  };

  const adjust = async () => {
    setBusy(true);
    setBanner(null);
    try {
      const res = await apiFetch("/v1/retailer/stock/adjust", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `adj-${Date.now()}`,
        },
        body: JSON.stringify({
          location_id: locationId,
          sku: adjustSku,
          qty_delta: Number(adjustDelta),
          stock_bin: adjustBin,
          note: "manual_adjust",
        }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(
          (json as { error?: string }).error || `adjust_failed_${res.status}`,
        );
      }
      setBanner("Stock adjusted");
      await loadStock();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Adjust failed");
    } finally {
      setBusy(false);
    }
  };

  const transfer = async () => {
    setBusy(true);
    setBanner(null);
    try {
      const res = await apiFetch("/v1/retailer/stock/transfer", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `xfer-${Date.now()}`,
        },
        body: JSON.stringify({
          location_id: locationId,
          sku: xferSku,
          qty: Number(xferQty),
          from_bin: "BACKROOM",
          to_bin: "FLOOR",
          note: "putaway",
        }),
      });
      const json = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(
          (json as { error?: string }).error || `transfer_failed_${res.status}`,
        );
      }
      setBanner("Transferred BACKROOM → FLOOR");
      await loadStock();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Transfer failed");
    } finally {
      setBusy(false);
    }
  };

  const count = async (force = false) => {
    setBusy(true);
    setBanner(null);
    setCountConflict(null);
    try {
      const commitRes = await apiFetch("/v1/retailer/stock/counts/commit", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": `cnt-${countSku}-${Date.now()}`,
        },
        body: JSON.stringify({
          location_id: locationId,
          stock_bin: countBin,
          base_version: countVersion,
          force,
          lines: [{ sku_id: countSku, counted_qty: Number(countQty) }],
        }),
      });
      if (commitRes.status === 404) {
        const res = await apiFetch("/v1/retailer/stock/counts", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "Idempotency-Key": `cnt-${Date.now()}`,
          },
          body: JSON.stringify({
            location_id: locationId,
            stock_bin: countBin,
            commit: true,
            lines: [{ sku: countSku, counted_qty: Number(countQty) }],
          }),
        });
        const json = await res.json().catch(() => ({}));
        if (!res.ok) {
          throw new Error(
            (json as { error?: string }).error || `count_failed_${res.status}`,
          );
        }
        setBanner("Cycle count committed");
        await loadStock();
        return;
      }
      const json = await commitRes.json().catch(() => ({}));
      if (commitRes.status === 409) {
        const conflict = json as {
          server_version?: number;
          server_lines?: Array<{ sku_id: string; on_hand: number }>;
        };
        if (conflict.server_version != null) {
          setCountVersion(conflict.server_version);
        }
        const lines = (conflict.server_lines ?? [])
          .map((l) => `${l.sku_id}: on hand ${l.on_hand}`)
          .join("; ");
        setCountConflict(lines || "Stock changed since draft — refresh and recount.");
        setBanner("Count conflict — server stock differs from your draft");
        return;
      }
      if (!commitRes.ok) {
        throw new Error(
          (json as { error?: string }).error || `count_failed_${commitRes.status}`,
        );
      }
      const ok = json as { new_version?: number };
      if (ok.new_version != null) setCountVersion(ok.new_version);
      setBanner(force ? "Cycle count force-applied" : "Cycle count committed");
      await loadStock();
      await loadCountVersion();
    } catch (e) {
      setBanner(e instanceof Error ? e.message : "Count failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <PageChrome
      title="Store stock"
      description="Backroom / floor inventory separate from supplier warehouse ATP. Receive Pegasus deliveries, transfer, adjust, count."
    >
      <div className="mx-auto max-w-4xl space-y-6 px-4 pb-16 pt-2">
        {banner && (
          <div className="rounded-lg border border-border bg-muted/40 px-3 py-2 text-sm">
            {banner}
          </div>
        )}
        {error && (
          <div className="flex items-center gap-2 text-sm text-red-600">
            <AlertTriangle className="h-4 w-4" />
            {error}
          </div>
        )}

        <div className="flex flex-wrap items-center gap-3">
          <label className="text-sm">
            Location{" "}
            <select
              className="ml-2 rounded-lg border border-border bg-background px-3 py-2"
              value={locationId}
              onChange={(e) => setLocationId(e.target.value)}
            >
              {locations.map((l) => (
                <option key={l.location_id} value={l.location_id}>
                  {l.name}
                  {l.is_primary ? " (primary)" : ""}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            className="text-sm underline"
            onClick={() => void loadStock()}
          >
            Refresh
          </button>
        </div>

        <section className="grid gap-4 md:grid-cols-2">
          <div className="rounded-xl border border-border bg-card p-4 space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <Plus className="h-4 w-4" /> Receive order into stock
            </h3>
            <input
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              placeholder="Order ID (COMPLETED / ARRIVED)"
              value={orderId}
              onChange={(e) => setOrderId(e.target.value)}
            />
            <button
              type="button"
              disabled={busy || !orderId}
              onClick={() => void receiveOrder()}
              className="rounded-lg bg-foreground px-3 py-2 text-sm text-background disabled:opacity-50"
            >
              Receive to BACKROOM
            </button>
          </div>

          <div className="rounded-xl border border-border bg-card p-4 space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <ArrowLeftRight className="h-4 w-4" /> Putaway transfer
            </h3>
            <input
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              placeholder="SKU"
              value={xferSku}
              onChange={(e) => setXferSku(e.target.value)}
            />
            <input
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              placeholder="Qty"
              value={xferQty}
              onChange={(e) => setXferQty(e.target.value)}
            />
            <button
              type="button"
              disabled={busy || !xferSku}
              onClick={() => void transfer()}
              className="rounded-lg border border-border px-3 py-2 text-sm disabled:opacity-50"
            >
              BACKROOM → FLOOR
            </button>
          </div>

          <div className="rounded-xl border border-border bg-card p-4 space-y-2">
            <h3 className="font-semibold">Adjust</h3>
            <input
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              placeholder="SKU"
              value={adjustSku}
              onChange={(e) => setAdjustSku(e.target.value)}
            />
            <input
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              placeholder="Qty delta (+/-)"
              value={adjustDelta}
              onChange={(e) => setAdjustDelta(e.target.value)}
            />
            <select
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              value={adjustBin}
              onChange={(e) => setAdjustBin(e.target.value)}
            >
              <option value="BACKROOM">BACKROOM</option>
              <option value="FLOOR">FLOOR</option>
              <option value="QUARANTINE">QUARANTINE</option>
            </select>
            <button
              type="button"
              disabled={busy || !adjustSku}
              onClick={() => void adjust()}
              className="rounded-lg border border-border px-3 py-2 text-sm disabled:opacity-50"
            >
              Apply adjust
            </button>
          </div>

          <div className="rounded-xl border border-border bg-card p-4 space-y-2">
            <h3 className="font-semibold flex items-center gap-2">
              <ClipboardList className="h-4 w-4" /> Cycle count
            </h3>
            <input
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              placeholder="SKU"
              value={countSku}
              onChange={(e) => setCountSku(e.target.value)}
            />
            <input
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              placeholder="Counted qty"
              value={countQty}
              onChange={(e) => setCountQty(e.target.value)}
            />
            <select
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm"
              value={countBin}
              onChange={(e) => setCountBin(e.target.value)}
            >
              <option value="FLOOR">FLOOR</option>
              <option value="BACKROOM">BACKROOM</option>
            </select>
            <p className="text-xs text-muted-foreground">
              Stock version: {countVersion} (refresh after other changes)
            </p>
            {countConflict && (
              <p className="text-xs text-amber-700 dark:text-amber-400">
                Conflict: {countConflict}
              </p>
            )}
            <button
              type="button"
              disabled={busy || !countSku}
              onClick={() => void count()}
              className="rounded-lg border border-border px-3 py-2 text-sm disabled:opacity-50"
            >
              Commit count
            </button>
            {countConflict &&
              (retailerRole === "MANAGER" || retailerRole === "OWNER") && (
                <button
                  type="button"
                  disabled={busy || !countSku}
                  onClick={() => void count(true)}
                  className="rounded-lg border border-amber-600 px-3 py-2 text-sm text-amber-800 disabled:opacity-50"
                >
                  Force apply (manager)
                </button>
              )}
          </div>
        </section>

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 flex items-center gap-2 font-semibold">
            <Package className="h-4 w-4" /> Balances
          </h2>
          {loading && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" /> Loading…
            </div>
          )}
          {!loading && items.length === 0 && (
            <p className="text-sm text-muted-foreground">
              No store stock yet. Receive a completed delivery or run an adjust.
            </p>
          )}
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-muted-foreground border-b border-border">
                  <th className="py-2 pr-3">SKU</th>
                  <th className="py-2 pr-3">Bin</th>
                  <th className="py-2 pr-3">On hand</th>
                  <th className="py-2 pr-3">Reserved</th>
                  <th className="py-2">Available</th>
                </tr>
              </thead>
              <tbody>
                {items.map((row) => (
                  <tr
                    key={`${row.sku}-${row.stock_bin}`}
                    className="border-b border-border/60"
                  >
                    <td className="py-2 pr-3 font-medium">{row.sku}</td>
                    <td className="py-2 pr-3">{row.stock_bin}</td>
                    <td className="py-2 pr-3">{row.on_hand}</td>
                    <td className="py-2 pr-3">{row.reserved}</td>
                    <td className="py-2">{row.available}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </PageChrome>
  );
}
