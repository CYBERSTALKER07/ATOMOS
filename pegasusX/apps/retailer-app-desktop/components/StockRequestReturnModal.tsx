"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Loader2, RotateCcw, Search, X } from "lucide-react";
import { apiFetch } from "@/lib/auth";
import { getRetailerProfile } from "@/lib/retailer-profile";
import type { LineItem, Order, TrackingOrder } from "@/lib/types";
import { FileClaimPanel } from "./FileClaimPanel";

const CLAIMABLE_STATES = new Set(["COMPLETED", "DELIVERED_ON_CREDIT"]);

type ListOrderRow = {
  order_id: string;
  retailer_id?: string;
  supplier_id?: string;
  amount?: number;
  total_amount?: number;
  total_minor?: number;
  payment_gateway?: string;
  payment_status?: string;
  state?: string;
  status?: string;
  route_id?: string | null;
  order_source?: string | null;
  auto_confirm_at?: string | null;
  deliver_before?: string | null;
  delivery_token?: string | null;
  version?: number;
  created_at?: string;
  items?: Array<{
    line_item_id?: string;
    order_id?: string;
    sku_id?: string;
    sku_name?: string;
    product_id?: string;
    product_name?: string;
    quantity: number;
    unit_price: number;
    status?: string;
  }>;
};

type Props = {
  open: boolean;
  onClose: () => void;
  preferredSku?: string;
};

function orderState(row: ListOrderRow): string {
  return (row.state || row.status || "").trim();
}

function mapItems(orderId: string, row: ListOrderRow): LineItem[] {
  return (row.items ?? []).map((item) => {
    const sku = item.sku_id || item.product_id || item.line_item_id || "";
    return {
      line_item_id: item.line_item_id || sku,
      order_id: item.order_id || orderId,
      sku_id: sku,
      sku_name: item.sku_name || item.product_name,
      quantity: item.quantity,
      unit_price: item.unit_price,
      status: item.status || orderState(row) || "COMPLETED",
    };
  });
}

function toOrder(row: ListOrderRow): Order {
  const state = orderState(row);
  return {
    order_id: row.order_id,
    retailer_id: row.retailer_id || "",
    supplier_id: row.supplier_id || "",
    amount: row.amount ?? row.total_amount ?? row.total_minor ?? 0,
    payment_gateway: row.payment_gateway || "",
    payment_status: row.payment_status || "",
    state,
    route_id: row.route_id ?? null,
    order_source: row.order_source ?? null,
    auto_confirm_at: row.auto_confirm_at ?? null,
    deliver_before: row.deliver_before ?? null,
    delivery_token: row.delivery_token ?? null,
    version: row.version ?? 0,
    created_at: row.created_at || "",
    items: mapItems(row.order_id, row),
  };
}

function trackingToOrder(t: TrackingOrder): Order {
  return {
    order_id: t.order_id,
    retailer_id: "",
    supplier_id: t.supplier_id,
    amount: t.total_amount,
    payment_gateway: "",
    payment_status: "",
    state: t.state,
    route_id: null,
    order_source: t.order_source ?? null,
    auto_confirm_at: null,
    deliver_before: null,
    delivery_token: t.delivery_token ?? null,
    version: 0,
    created_at: t.created_at,
    items: (t.items ?? []).map((item) => ({
      line_item_id: item.product_id,
      order_id: t.order_id,
      sku_id: item.product_id,
      sku_name: item.product_name,
      quantity: item.quantity,
      unit_price: item.unit_price,
      status: t.state,
    })),
  };
}

export function StockRequestReturnModal({
  open,
  onClose,
  preferredSku,
}: Props) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [orders, setOrders] = useState<Order[]>([]);
  const [selected, setSelected] = useState<Order | null>(null);
  const [resolving, setResolving] = useState(false);

  const loadOrders = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const profile = getRetailerProfile();
      const url = profile?.id
        ? `/v1/retailers/${profile.id}/orders`
        : "/v1/orders";
      const res = await apiFetch(url);
      if (!res.ok) throw new Error(`orders_load_failed_${res.status}`);
      const json = (await res.json()) as ListOrderRow[] | { orders?: ListOrderRow[] };
      const rows = Array.isArray(json) ? json : (json.orders ?? []);
      const claimable = rows
        .map(toOrder)
        .filter((o) => CLAIMABLE_STATES.has(o.state));
      setOrders(claimable);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load orders");
      setOrders([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!open) return;
    setSelected(null);
    setQuery("");
    void loadOrders();
  }, [open, loadOrders]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return orders;
    return orders.filter((o) => o.order_id.toLowerCase().includes(q));
  }, [orders, query]);

  const pickOrder = async (order: Order) => {
    setResolving(true);
    setError(null);
    try {
      let next = order;
      if (!next.items?.length) {
        const res = await apiFetch("/v1/retailer/tracking");
        if (res.ok) {
          const body = (await res.json()) as {
            orders?: TrackingOrder[];
            recent_receipts?: TrackingOrder[];
          };
          const pool = [
            ...(body.orders ?? []),
            ...(body.recent_receipts ?? []),
          ];
          const hit = pool.find((t) => t.order_id === order.order_id);
          if (hit) next = trackingToOrder(hit);
        }
      }
      setSelected(next);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load order");
    } finally {
      setResolving(false);
    }
  };

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center">
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Request return or chargeback"
        className="flex max-h-[90vh] w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-border bg-background shadow-xl"
      >
        <div className="flex items-center justify-between border-b border-border px-4 py-3">
          <div className="flex items-center gap-2">
            <RotateCcw className="h-4 w-4" />
            <h2 className="font-semibold text-sm">
              {selected
                ? `File claim · #${selected.order_id.slice(-8)}`
                : "Request return / chargeback"}
            </h2>
          </div>
          <button
            type="button"
            className="rounded-lg p-1 hover:bg-muted"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="overflow-y-auto px-4 py-3 space-y-3">
          {error && (
            <p className="text-sm text-red-600">{error}</p>
          )}

          {!selected && (
            <>
              <p className="text-sm text-muted-foreground">
                Pick a completed delivery, then file the same claim used on
                order detail. Window is within 48h (server enforces).
              </p>
              {preferredSku && (
                <p className="text-xs text-muted-foreground">
                  Preferred SKU from stock:{" "}
                  <span className="font-medium text-foreground">
                    {preferredSku}
                  </span>
                </p>
              )}
              <div className="relative">
                <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <input
                  className="w-full rounded-lg border border-border bg-background py-2 pl-9 pr-3 text-sm"
                  placeholder="Search by order id"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                />
              </div>
              {loading && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" /> Loading orders…
                </div>
              )}
              {!loading && filtered.length === 0 && (
                <p className="text-sm text-muted-foreground">
                  No COMPLETED / DELIVERED_ON_CREDIT orders found.
                </p>
              )}
              <ul className="space-y-2">
                {filtered.map((o) => (
                  <li key={o.order_id}>
                    <button
                      type="button"
                      disabled={resolving}
                      onClick={() => void pickOrder(o)}
                      className="w-full rounded-xl border border-border px-3 py-2 text-left text-sm hover:bg-muted/50 disabled:opacity-50"
                    >
                      <div className="font-medium">
                        #{o.order_id.slice(-8)}{" "}
                        <span className="text-xs font-normal text-muted-foreground">
                          {o.state.replaceAll("_", " ")}
                        </span>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {o.order_id}
                        {o.created_at ? ` · ${o.created_at.slice(0, 10)}` : ""}
                      </div>
                    </button>
                  </li>
                ))}
              </ul>
              {resolving && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" /> Loading line
                  items…
                </div>
              )}
            </>
          )}

          {selected && (
            <>
              <button
                type="button"
                className="text-sm underline"
                onClick={() => setSelected(null)}
              >
                ← Choose another order
              </button>
              <FileClaimPanel
                order={selected}
                initialSku={preferredSku}
                onFiled={() => {
                  /* keep modal open so clerk can see success id */
                }}
              />
            </>
          )}
        </div>
      </div>
    </div>
  );
}
