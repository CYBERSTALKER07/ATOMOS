"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { NegotiationProposalRow } from "@pegasusx/types";
import { PortalSurface } from "../../_components/PortalSurface";

const api = createSupplierApi();

function resolveIdempotencyKey(proposalId: string, action: string): string {
  return `negotiate-resolve:${proposalId}:${action}`;
}

export default function NegotiationsExceptionsPage() {
  const [rows, setRows] = useState<NegotiationProposalRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api
      .getSupplierNegotiationsPending({ limit: 500, offset: 0 })
      .then((resp) => setRows(resp.data ?? []))
      .catch((err) => setError(err instanceof Error ? err.message : "load_failed"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const resolve = async (proposalId: string, action: "APPROVE" | "REJECT") => {
    setBusyId(proposalId);
    try {
      await api.resolveSupplierNegotiation(
        { proposal_id: proposalId, action },
        resolveIdempotencyKey(proposalId, action),
      );
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "resolve_failed");
    } finally {
      setBusyId(null);
    }
  };

  return (
    <PortalSurface
      title="Quantity negotiations"
      description="Driver-proposed line-item adjustments awaiting supplier decision."
      loading={loading}
      error={error}
      empty={!loading && rows.length === 0}
      emptyMessage="No pending negotiations."
    >
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        <Link href="/exceptions" className="text-[var(--color-md-primary)] underline">
          All exceptions
        </Link>
      </p>
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {rows.map((row) => (
          <li key={row.proposal_id} className="p-4 space-y-3">
            <div className="flex flex-wrap gap-2 items-center">
              <span className="md-chip h-6 text-xs">PENDING</span>
              <span className="font-mono text-[var(--color-md-primary)]">{row.order_id}</span>
            </div>
            <p className="md-typescale-body-small text-[var(--color-md-outline)]">
              Driver {row.driver_id} · Proposal {row.proposal_id}
            </p>
            {row.items?.length ? (
              <ul className="md-typescale-body-small text-[var(--color-md-on-surface)] space-y-1">
                {row.items.map((item) => (
                  <li key={item.sku_id}>
                    {item.sku_id}: {item.original_qty} → {item.proposed_qty}
                  </li>
                ))}
              </ul>
            ) : null}
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                className="md-btn md-btn-filled md-typescale-label-large"
                disabled={busyId === row.proposal_id}
                onClick={() => resolve(row.proposal_id, "APPROVE")}
              >
                Approve
              </button>
              <button
                type="button"
                className="md-btn md-btn-outlined md-typescale-label-large"
                disabled={busyId === row.proposal_id}
                onClick={() => resolve(row.proposal_id, "REJECT")}
              >
                Reject
              </button>
            </div>
          </li>
        ))}
      </ul>
    </PortalSurface>
  );
}
