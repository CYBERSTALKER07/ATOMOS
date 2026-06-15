import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { SUPPLIER_DISPATCH_REFRESH_EVENTS } from "@/lib/supplier-ws-events";
import { useSupplierWsRefresh } from "@/lib/use-supplier-ws-refresh";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";

export interface ManifestData {
  id: string;
  status: "DRAFT" | "LOADING" | "DISPATCHED";
  driverName: string;
  vehiclePlate: string;
  orderCount: number;
  totalVu: number;
  estimatedTime: string;
}

const api = createSupplierApi();

function formatEta(status: string, updatedAt: string): string {
  const ts = Date.parse(updatedAt);
  if (Number.isNaN(ts)) {
    return status === "DISPATCHED" ? "In route" : "Pending assignment";
  }
  const label = new Date(ts).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false });
  if (status === "DISPATCHED") {
    return `ETA: ${label}`;
  }
  if (status === "LOADING") {
    return `Dep: ${label}`;
  }
  return "Pending assignment";
}

export function useDispatchData() {
  const [manifests, setManifests] = useState<ManifestData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const resp = await api.getSupplierManifests();
      const mapped = resp.manifests.map((row): ManifestData => {
        const status: ManifestData["status"] =
          row.status === "LOADING" || row.status === "DISPATCHED" ? row.status : "DRAFT";
        return {
          id: row.manifest_id,
          status,
          driverName: row.driver_name || "Unassigned",
          vehiclePlate: row.vehicle_plate || "-",
          orderCount: row.orders_count,
          totalVu: row.total_vu,
          estimatedTime: formatEta(status, row.updated_at),
        };
      });
      setManifests(mapped);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "load_manifests_failed");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    const interval = setInterval(() => {
      void refresh();
    }, 30_000);
    return () => clearInterval(interval);
  }, [refresh]);

  useSupplierWsRefresh(
    () => {
      void refresh();
    },
    {
      eventTypes: SUPPLIER_DISPATCH_REFRESH_EVENTS,
      debounceMs: 500,
    },
  );

  useSupplierSessionReconcile(() => {
    void refresh();
  });

  return { manifests, setManifests, loading, error, refresh };
}
