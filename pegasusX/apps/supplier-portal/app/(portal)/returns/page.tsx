"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierOrder } from "@pegasusx/types";
import StatusBadge from "@/components/StatusBadge";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

function formatMoney(order: SupplierOrder) {
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency: order.currency,
      maximumFractionDigits: 2,
    }).format(order.total_minor / 100);
  } catch {
    return `${order.total_minor} ${order.currency}`;
  }
}

export default function ReturnsPage() {
  const [orders, setOrders] = useState<SupplierOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierOrders({ filter: "RETURNS", limit: 100, offset: 0 })
      .then((resp) => setOrders(resp.orders))
      .catch((err) => setError(err instanceof Error ? err.message : "load_returns_failed"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PortalSurface
      title="Returns"
      description="Retailer return and reversal workflows tied to order lifecycle."
      loading={loading}
      error={error}
      empty={!loading && orders.length === 0}
      emptyMessage="No cancelled or rejected orders in the current window."
    >
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {orders.map((order) => (
          <li key={order.order_id} className="p-4 md-typescale-body-medium">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="font-mono text-[var(--color-md-primary)]">{order.order_id}</div>
              <Link href={`/orders?filter=CANCELLED`} className="text-[var(--color-md-primary)] underline md-typescale-label-large">
                Open in orders
              </Link>
            </div>
            <div className="flex items-center gap-2 mt-1">
              <StatusBadge state={order.status} />
              {order.decision ? (
                <span className="text-[var(--color-md-outline)]">{order.decision}</span>
              ) : null}
            </div>
            <p className="mt-2 text-[var(--color-md-outline)]">
              Retailer {order.retailer_id} · {formatMoney(order)} · updated {order.updated_at}
            </p>
            {order.note ? <p className="mt-2">{order.note}</p> : null}
          </li>
        ))}
      </ul>
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        Treasury reversals route through{" "}
        <Link href="/reconciliation" className="text-[var(--color-md-primary)] underline">
          reconciliation
        </Link>
        .
      </p>
    </PortalSurface>
  );
}
