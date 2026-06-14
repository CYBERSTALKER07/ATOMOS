"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierManifestRow } from "@pegasusx/types";
import { downloadCsv } from "@/lib/csv";
import { usePagination } from "@/lib/use-pagination";
import { ListToolbar } from "@/components/ListToolbar";
import StatusBadge from "@/components/StatusBadge";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

export default function ManifestsPage() {
  const [manifests, setManifests] = useState<SupplierManifestRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierManifests()
      .then((resp) => setManifests(resp.manifests))
      .catch((err) => setError(err instanceof Error ? err.message : "load_manifests_failed"))
      .finally(() => setLoading(false));
  }, []);

  const pagination = usePagination(manifests, 15);

  return (
    <PortalSurface
      title="Manifests"
      description="Loading manifests, seal, and dispatch lifecycle. Use Dispatch for active queue operations."
      loading={loading}
      error={error}
      empty={!loading && manifests.length === 0}
      emptyMessage="No manifests in the current window. Assign orders from Dispatch when routes are ready."
    >
      <ListToolbar
        page={pagination.page}
        pageCount={pagination.pageCount}
        totalLabel={`${manifests.length} manifests`}
        onPrev={pagination.prev}
        onNext={pagination.next}
        onExport={() =>
          downloadCsv(
            "supplier-manifests.csv",
            ["manifest_id", "status", "orders_count", "driver_name", "updated_at"],
            manifests.map((manifest) => [
              manifest.manifest_id,
              manifest.status,
              String(manifest.orders_count),
              manifest.driver_name,
              manifest.updated_at,
            ]),
          )
        }
      />
      <div className="md-card overflow-hidden">
        <table className="desk-table w-full">
          <thead>
            <tr className="border-b border-[var(--color-md-outline-variant)] text-[var(--color-md-outline)]">
              <th className="md-typescale-label-medium p-4 font-medium">Manifest</th>
              <th className="md-typescale-label-medium p-4 font-medium">Status</th>
              <th className="md-typescale-label-medium p-4 font-medium text-right">Orders</th>
              <th className="md-typescale-label-medium p-4 font-medium">Driver</th>
            </tr>
          </thead>
          <tbody>
            {pagination.pageItems.map((manifest) => (
              <tr
                key={manifest.manifest_id}
                className="border-b border-[var(--color-md-outline-variant)] last:border-0"
              >
                <td className="p-4 md-typescale-body-medium font-mono">
                  <Link href={`/manifests/${manifest.manifest_id}`} className="text-[var(--color-md-primary)] underline">
                    {manifest.manifest_id}
                  </Link>
                </td>
                <td className="p-4 md-typescale-body-medium"><StatusBadge state={manifest.status} /></td>
                <td className="p-4 md-typescale-body-medium text-right">{manifest.orders_count}</td>
                <td className="p-4 md-typescale-body-medium">{manifest.driver_name}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="md-typescale-body-medium text-[var(--color-md-outline)] flex flex-wrap gap-4">
        <Link href="/dispatch" className="text-[var(--color-md-primary)] underline">
          Open dispatch queue
        </Link>
        <Link href="/manifest-exceptions" className="text-[var(--color-md-primary)] underline">
          Manifest gate exceptions
        </Link>
      </p>
    </PortalSurface>
  );
}
