"use client";

import { useCallback, useEffect, useState } from "react";
import {
  Loader2,
  Package,
  ArrowLeftRight,
  Plus,
  AlertTriangle,
  ClipboardList,
  RotateCcw,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { StockRequestReturnModal } from "@/components/StockRequestReturnModal";
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
  const [claimOpen, setClaimOpen] = useState(false);
  const [claimSku, setClaimSku] = useState<string | undefined>(undefined);

  const openRequestReturn = (sku?: string) => {
    setClaimSku(sku);
    setClaimOpen(true);
  };

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

  const count = async () => {
    setBusy(true);
    setBanner(null);
    try {
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
          <button
            type="button"
            onClick={() => openRequestReturn()}
            className="inline-flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm"
          >
            <RotateCcw className="h-4 w-4" />
            Request return / chargeback
          </button>
        </div>

        <StockRequestReturnModal
          open={claimOpen}
          preferredSku={claimSku}
          onClose={() => {
            setClaimOpen(false);
            setClaimSku(undefined);
          }}
        />

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
            <button
              type="button"
              disabled={busy || !countSku}
              onClick={() => void count()}
              className="rounded-lg border border-border px-3 py-2 text-sm disabled:opacity-50"
            >
              Commit count
            </button>
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
                  <th className="py-2 pr-3">Available</th>
                  <th className="py-2">Actions</th>
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
                    <td className="py-2 pr-3">{row.available}</td>
                    <td className="py-2">
                      <button
                        type="button"
                        className="text-xs underline"
                        onClick={() => openRequestReturn(row.sku)}
                      >
                        Request return
                      </button>
                    </td>
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
