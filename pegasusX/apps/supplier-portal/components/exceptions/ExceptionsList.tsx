"use client";

import { usePortalT } from "@/lib/i18n";
import React, { useState } from "react";
import type { SupplierExceptionRow } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import StatusBadge from "@/components/StatusBadge";

const api = createSupplierApi();

interface ExceptionsListProps {
  exceptions: SupplierExceptionRow[];
  onResolved?: () => void;
}

export function ExceptionsList({ exceptions, onResolved }: ExceptionsListProps) {
  const t = usePortalT();
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  if (exceptions.length === 0) return null;

  const resolve = async (row: SupplierExceptionRow) => {
    setBusy(row.order_id);
    setError(null);
    try {
      const body =
        row.kind === "CREDIT_NOTE_DRAFT" && row.note
          ? { credit_note_id: row.note }
          : undefined;
      await api.resolveSupplierException(row.kind, row.order_id, body);
      onResolved?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.resolve_failed"));
    } finally {
      setBusy(null);
    }
  };

  const canResolve = (kind: string) =>
    kind === "CASH_DISCREPANCY" || kind === "CREDIT_NOTE_DRAFT" || kind === "CREDIT_FREEZE";

  return (
    <>
      {error ? <p className="text-sm text-red-600 mb-2">{error}</p> : null}
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {exceptions.map((row) => (
          <li key={`${row.kind}:${row.order_id}`} className="p-4 md-typescale-body-medium">
            <div className="flex flex-wrap items-center gap-2">
              <span className="md-chip h-6 text-xs">{row.kind}</span>
              <span className="font-mono text-[var(--color-md-primary)]">{row.order_id}</span>
              <StatusBadge state={row.status} />
            </div>
            {row.note ? <p className="mt-2 text-[var(--color-md-outline)]">{row.note}</p> : null}
            <p className="mt-1 text-sm text-[var(--color-md-outline)]">
              Updated {new Date(row.updated_at).toLocaleString()}
            </p>
            {canResolve(row.kind) ? (
              <button
                type="button"
                className="md-btn md-btn-outlined mt-2"
                disabled={busy === row.order_id}
                onClick={() => void resolve(row)}
              >
                {busy === row.order_id ? "Resolving…" : "Resolve"}
              </button>
            ) : null}
          </li>
        ))}
      </ul>
    </>
  );
}
