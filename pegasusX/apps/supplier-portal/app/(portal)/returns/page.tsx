"use client";

import { useCallback, useEffect, useState } from "react";
import { Button } from "@heroui/react";
import { supplierResolveReturnKey } from "@pegasusx/api-client";
import { PortalSurface } from "../_components/PortalSurface";
import { supplierFetch } from "@/lib/auth";

type Resolution = "WRITE_OFF" | "RETURN_TO_STOCK";

type SupplierReturnRow = {
  return_id: string;
  line_item_id: string;
  order_id: string;
  sku_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  status: string;
  physical_status: string;
  received_qty: number;
  manifest_id?: string;
  driver_id?: string;
  driver_name?: string;
  reason: string;
  driver_notes?: string;
  retailer_id: string;
  retailer_name: string;
  created_at: string;
};

function formatAmount(amountMinor: number): string {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 0,
  }).format(amountMinor);
}

export default function ReturnsPage() {
  const [items, setItems] = useState<SupplierReturnRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [resolvingId, setResolvingId] = useState<string | null>(null);
  const [resolution, setResolution] = useState<Resolution>("RETURN_TO_STOCK");
  const [notes, setNotes] = useState("");
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const fetchReturns = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await supplierFetch("/v1/supplier/returns?limit=100&offset=0&status=PENDING");
      if (!res.ok) {
        throw new Error("Failed to load returns");
      }
      const json = (await res.json()) as { data?: SupplierReturnRow[] };
      setItems(json.data ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "load_returns_failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchReturns();
  }, [fetchReturns]);

  async function handleResolve(returnId: string) {
    setActionLoading(returnId);
    try {
      const res = await supplierFetch("/v1/supplier/returns/resolve", {
        method: "POST",
        headers: {
          "Idempotency-Key": supplierResolveReturnKey(returnId, resolution),
        },
        body: JSON.stringify({
          return_id: returnId,
          line_item_id: returnId,
          resolution,
          notes,
        }),
      });
      const body = (await res.json().catch(() => ({}))) as { error?: string; message?: string };
      if (!res.ok) {
        throw new Error(body.error || body.message || "Resolution failed");
      }
      setResolvingId(null);
      setNotes("");
      await fetchReturns();
    } catch (err) {
      setError(err instanceof Error ? err.message : "resolve_failed");
    } finally {
      setActionLoading(null);
    }
  }

  const totalDamageValue = items.reduce((sum, item) => sum + item.quantity * item.unit_price, 0);

  return (
    <PortalSurface
      title="Dispute & Returns"
      description="Driver-rejected quantities awaiting write-off or return-to-stock."
      loading={loading}
      error={error}
      empty={!loading && items.length === 0}
      emptyMessage="No open returns — all rejected delivery lines are resolved."
      actions={
        <Button size="sm" variant="flat" onPress={() => void fetchReturns()}>
          Refresh
        </Button>
      }
    >
      <div className="grid gap-4 md:grid-cols-2 mb-6">
        <div className="md-card p-4">
          <p className="md-typescale-label-small text-[var(--color-md-outline)]">Open returns</p>
          <p className="md-typescale-headline-small font-mono text-[var(--color-md-error)]">{items.length}</p>
        </div>
        <div className="md-card p-4">
          <p className="md-typescale-label-small text-[var(--color-md-outline)]">Total damage value</p>
          <p className="md-typescale-headline-small font-mono text-[var(--color-md-error)]">
            {formatAmount(totalDamageValue)}
          </p>
        </div>
      </div>

      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {items.map((item) => (
          <li key={item.return_id} className="p-4 md-typescale-body-medium">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-semibold">{item.product_name}</span>
                  <span className="rounded-full bg-[var(--color-md-error-container)] px-2 py-0.5 text-xs uppercase">
                    {item.reason.replace(/_/g, " ")}
                  </span>
                  <span className="rounded-full bg-[var(--color-md-surface-variant)] px-2 py-0.5 text-xs uppercase">
                    {item.physical_status?.replace(/_/g, " ") || "pending"}
                  </span>
                </div>
                <p className="mt-1 text-[var(--color-md-outline)]">
                  Retailer {item.retailer_name} · Qty {item.quantity}
                  {item.received_qty > 0 ? ` (${item.received_qty} scanned)` : ""} ·{" "}
                  {formatAmount(item.quantity * item.unit_price)} · Order {item.order_id.slice(0, 10)}…
                </p>
                {item.driver_name ? (
                  <p className="mt-1 text-xs text-[var(--color-md-outline)]">Driver {item.driver_name}</p>
                ) : null}
                {item.driver_notes ? (
                  <p className="mt-2 text-[var(--color-md-on-surface-variant)]">{item.driver_notes}</p>
                ) : null}
              </div>

              {resolvingId === item.return_id ? (
                <div className="flex w-full flex-col gap-2 sm:w-auto sm:min-w-[240px]">
                  <select
                    className="rounded-lg border border-[var(--color-md-outline-variant)] bg-transparent px-3 py-2 text-sm"
                    value={resolution}
                    onChange={(e) => setResolution(e.target.value as Resolution)}
                  >
                    <option value="RETURN_TO_STOCK">Return to stock</option>
                    <option value="WRITE_OFF">Write off</option>
                  </select>
                  <input
                    className="rounded-lg border border-[var(--color-md-outline-variant)] bg-transparent px-3 py-2 text-sm"
                    placeholder="Notes (optional)"
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                  />
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      color="primary"
                      isLoading={actionLoading === item.return_id}
                      onPress={() => void handleResolve(item.return_id)}
                    >
                      Confirm
                    </Button>
                    <Button size="sm" variant="light" onPress={() => setResolvingId(null)}>
                      Cancel
                    </Button>
                  </div>
                </div>
              ) : item.physical_status === "RESTOCKED" || item.physical_status === "WRITTEN_OFF" ? (
                <span className="text-xs text-[var(--color-md-outline)]">Gate resolved</span>
              ) : (
                <Button size="sm" variant="flat" onPress={() => setResolvingId(item.return_id)}>
                  Dispute / override
                </Button>
              )}
            </div>
          </li>
        ))}
      </ul>
    </PortalSurface>
  );
}
