"use client";

import { useEffect, useState } from "react";
import { ApiClient } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

export default function AnalyticsPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(0);
  const [skus, setSkus] = useState(0);

  useEffect(() => {
    let cancelled = false;
    api
      .getSupplierDashboard()
      .then((dash) => {
        if (!cancelled) {
          setPending(dash.pending_orders);
          setSkus(dash.inventory_skus);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setError("load_dashboard_failed");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <PortalSurface
      title="Analytics"
      description="Operational KPIs sourced from supplier dashboard authority."
      loading={loading}
      error={error}
    >
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="md-card p-6">
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Pending orders</p>
          <p className="md-typescale-display-small mt-2">{pending}</p>
        </div>
        <div className="md-card p-6">
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Inventory SKUs</p>
          <p className="md-typescale-display-small mt-2">{skus}</p>
        </div>
      </div>
    </PortalSurface>
  );
}
