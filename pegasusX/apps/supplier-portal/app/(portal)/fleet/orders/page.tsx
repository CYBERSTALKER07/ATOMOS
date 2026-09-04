"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierFleetOrderRow } from "@pegasusx/types";
import StatusBadge from "@/components/StatusBadge";
import { ListToolbar } from "@/components/ListToolbar";
import { PageChrome } from "@/components/PageChrome";
import { DataList, DataListRow } from "@/components/portal";
import { usePagination } from "@/lib/use-pagination";

const api = createSupplierApi();

export default function SupplierFleetOrdersPage() {
  const t = usePortalT();
  const [orders, setOrders] = useState<SupplierFleetOrderRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierFleetOrders()
      .then(setOrders)
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.fleet_orders_failed")))
      .finally(() => setLoading(false));
  }, []);

  const pagination = usePagination(orders, 15);

  return (
    <PageChrome
      title={t("supplier_portal.fleet.orders.text.fleet_orders")}
      description={t("supplier_portal.residual.text.supplier_scoped_active_orders_for_dispatch_oversight")}
      icon="fleet"
      loading={loading}
      error={error}
      empty={!loading && orders.length === 0}
      emptyMessage={t("supplier_portal.residual.text.no_active_fleet_orders")}
    >
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        <Link href="/fleet" className="text-[var(--color-md-primary)] underline">
          Fleet & org
        </Link>
      </p>
      <ListToolbar
        page={pagination.page}
        pageCount={pagination.pageCount}
        totalLabel={`${orders.length} fleet orders`}
        onPrev={pagination.prev}
        onNext={pagination.next}
      />
      <DataList>
        {pagination.pageItems.map((row) => (
          <DataListRow key={row.order_id}>
            <div className="min-w-0 md-typescale-body-medium">
              <div className="flex flex-wrap gap-2 items-center">
                <span className="font-mono text-[var(--color-md-primary)]">{row.order_id}</span>
                <StatusBadge state={row.status} />
              </div>
              <p className="mt-1 text-[var(--color-md-outline)]">
                Driver {row.driver_id || "—"} · Retailer {row.retailer_id || "—"}
                {row.route_id ? ` · Route ${row.route_id}` : ""}
              </p>
            </div>
          </DataListRow>
        ))}
      </DataList>
    </PageChrome>
  );
}
