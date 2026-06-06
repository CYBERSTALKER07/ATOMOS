"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierFleetOrderRow } from "@pegasusx/types";
import { PortalSurface } from "../../_components/PortalSurface";

const api = createSupplierApi();

export default function SupplierFleetOrdersPage() {
  const [orders, setOrders] = useState<SupplierFleetOrderRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierFleetOrders()
      .then(setOrders)
      .catch((err) => setError(err instanceof Error ? err.message : "fleet_orders_failed"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PortalSurface
      title="Fleet orders"
      description="Supplier-scoped active orders for dispatch oversight."
      loading={loading}
      error={error}
      empty={!loading && orders.length === 0}
      emptyMessage="No active fleet orders."
    >
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        <Link href="/fleet" className="text-[var(--color-md-primary)] underline">
          Fleet & org
        </Link>
      </p>
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {orders.map((row) => (
          <li key={row.order_id} className="p-4 md-typescale-body-medium">
            <div className="flex flex-wrap gap-2 items-center">
              <span className="font-mono text-[var(--color-md-primary)]">{row.order_id}</span>
              <span className="md-chip h-6 text-xs">{row.status}</span>
            </div>
            <p className="mt-1 text-[var(--color-md-outline)]">
              Driver {row.driver_id || "—"} · Retailer {row.retailer_id || "—"}
              {row.route_id ? ` · Route ${row.route_id}` : ""}
            </p>
          </li>
        ))}
      </ul>
    </PortalSurface>
  );
}
