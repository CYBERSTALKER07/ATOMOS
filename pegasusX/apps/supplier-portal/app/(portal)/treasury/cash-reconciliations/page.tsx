"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { CashReconciliationRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import StatusBadge from "@/components/StatusBadge";

const api = createSupplierApi();

function formatMinor(n: number): string {
  return (n ?? 0).toLocaleString();
}

export default function CashReconciliationsPage() {
  const [rows, setRows] = useState<CashReconciliationRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [note, setNote] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.listCashReconciliations({ status: "PENDING", limit: 100 });
      setRows(resp.reconciliations ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "load_failed");
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const accept = async (id: string) => {
    setBusyId(id);
    try {
      await api.acceptCashReconciliation(id, { note: note.trim() || undefined });
      setNote("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "accept_failed");
    } finally {
      setBusyId(null);
    }
  };

  const writeOff = async (id: string) => {
    setBusyId(id);
    try {
      await api.writeOffCashReconciliation(id, { note: note.trim() || undefined });
      setNote("");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "write_off_failed");
    } finally {
      setBusyId(null);
    }
  };

  return (
    <PageChrome
      title="Cash reconciliations"
      description="Driver-declared cash vs captured CASH payment legs — resolve open discrepancies."
      icon="treasury"
      loading={loading}
      error={error}
      empty={!loading && rows.length === 0}
      emptyMessage="No open cash discrepancies. Drivers auto-accept when declared matches expected."
    >
      <div className="mb-4">
        <label className="block text-sm text-[var(--color-md-outline)] mb-1">Finance note (optional)</label>
        <input
          className="md-input w-full max-w-md"
          placeholder="Verification note for accept / write-off"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
      </div>
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {rows.map((row) => (
          <li key={row.reconciliation_id} className="p-4 md-typescale-body-medium">
            <div className="flex flex-wrap items-center gap-2 mb-2">
              <span className="font-mono text-sm">{row.reconciliation_id}</span>
              <StatusBadge state={row.status} />
              <span className="text-[var(--color-md-outline)]">Driver {row.driver_id}</span>
              {row.route_id ? <span className="text-[var(--color-md-outline)]">Route {row.route_id}</span> : null}
            </div>
            <p>
              Expected {formatMinor(row.expected_cash_minor)} · Declared {formatMinor(row.declared_cash_minor)} · Diff{" "}
              <strong>{formatMinor(row.difference_minor)}</strong>
            </p>
            <p className="text-sm text-[var(--color-md-outline)] mt-1">
              Shift {row.shift_date} · {new Date(row.created_at).toLocaleString()}
            </p>
            {row.driver_note ? <p className="mt-2 text-sm">Driver: {row.driver_note}</p> : null}
            <div className="flex flex-wrap gap-2 mt-3">
              <button
                type="button"
                className="md-btn md-btn-filled"
                disabled={busyId === row.reconciliation_id}
                onClick={() => void accept(row.reconciliation_id)}
              >
                Accept
              </button>
              <button
                type="button"
                className="md-btn md-btn-outlined"
                disabled={busyId === row.reconciliation_id}
                onClick={() => void writeOff(row.reconciliation_id)}
              >
                Write off
              </button>
            </div>
          </li>
        ))}
      </ul>
    </PageChrome>
  );
}
