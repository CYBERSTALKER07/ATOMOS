"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierManifestRow } from "@pegasusx/types";
import { downloadCsv } from "@/lib/csv";
import { usePagination } from "@/lib/use-pagination";
import { ListToolbar } from "@/components/ListToolbar";
import { PageChrome } from "@/components/PageChrome";
import { ManifestsTable } from "@/components/manifests";

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
    <PageChrome
      icon="manifests"
      title="Manifests"
      description="Loading manifests, seal, and dispatch lifecycle. Use Dispatch for active queue operations."
      loading={loading}
      skeletonVariant="table"
      error={error}
      empty={!loading && !error && manifests.length === 0}
      emptyMessage="No manifests in the current window."
      emptyIcon="manifests"
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
      <ManifestsTable items={pagination.pageItems} />
      <p className="md-typescale-body-medium text-[var(--color-md-outline)] flex flex-wrap gap-4">
        <Link href="/dispatch" className="text-[var(--color-md-primary)] underline">
          Open dispatch queue
        </Link>
        <Link href="/manifest-exceptions" className="text-[var(--color-md-primary)] underline">
          Manifest gate exceptions
        </Link>
      </p>
    </PageChrome>
  );
}
