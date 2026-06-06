"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierOrder } from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

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
            <div className="font-mono text-[var(--color-md-primary)]">{order.order_id}</div>
            <div className="text-[var(--color-md-outline)] mt-1">
              {order.status}
              {order.decision ? ` · ${order.decision}` : ""}
            </div>
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
